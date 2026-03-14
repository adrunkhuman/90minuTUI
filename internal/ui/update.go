package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return m.loadArchiveCmd("http://www.90minut.pl/archsezon.php")
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.toggleFocus()
			return m, nil
		case "up", "k":
			if m.matchView {
				m.scrollMatch(-1)
				return m, nil
			}
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			if m.matchView {
				m.scrollMatch(1)
				return m, nil
			}
			m.moveCursor(1)
			return m, nil
		case "left", "h":
			m.shiftRound(-1)
			return m, nil
		case "right", "l":
			m.shiftRound(1)
			return m, nil
		case "enter":
			return m, m.handleEnter()
		case "esc", "backspace":
			if m.matchView {
				m.matchView = false
				m.match = nil
				m.matchScroll = 0
				m.err = ""
				return m, nil
			}
			if m.league != nil {
				m.err = ""
				if m.selectorVisible {
					m.closeSelector()
				} else {
					m.openSelector()
				}
			}
			return m, nil
		case "r":
			return m, m.handleReload()
		}

	case archiveLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}

		m.err = ""
		m.lastFetchAt = time.Now()
		m.seasons = msg.seasons
		m.seasonCursor = clamp(msg.selectedIdx, 0, len(m.seasons)-1)
		m.competitions = msg.competitions
		m.competitionCursor = m.preferredCompetitionIndex()

		if len(m.competitions) == 0 {
			return m, nil
		}

		m.loading = true
		competition := m.currentCompetition()
		if competition == nil {
			m.loading = false
			return m, nil
		}
		return m, m.loadLeagueCmd(competition.URL, competitionRequestKey(*competition))

	case competitionsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}

		season := m.currentSeason()
		// Drop stale async results if the user moved to another season.
		if season == nil || msg.seasonKey != seasonRequestKey(*season) {
			return m, nil
		}

		m.err = ""
		m.lastFetchAt = time.Now()
		m.competitions = msg.competitions
		m.competitionCursor = m.preferredCompetitionIndex()
		m.matchView = false
		m.match = nil
		m.matchScroll = 0

		if len(m.competitions) == 0 {
			return m, nil
		}

		m.loading = true
		competition := m.currentCompetition()
		if competition == nil {
			m.loading = false
			return m, nil
		}
		return m, m.loadLeagueCmd(competition.URL, competitionRequestKey(*competition))

	case leagueLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}

		competition := m.currentCompetition()
		// Drop stale async results if the user switched competitions.
		if competition == nil || msg.competitionKey != competitionRequestKey(*competition) {
			return m, nil
		}

		m.err = ""
		m.lastFetchAt = time.Now()
		m.matchView = false
		m.match = nil
		m.matchScroll = 0
		m.league = msg.league
		m.roundCursor = clamp(len(msg.league.Rounds)-1, 0, len(msg.league.Rounds)-1)
		m.fixtureCursor = 0
		m.closeSelector()
		return m, nil

	case matchLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.match = nil
			return m, nil
		}

		current := m.currentFixture()
		// Drop stale async results if the selected fixture changed.
		if current == nil || fixtureRequestKey(*current) != msg.fixtureKey {
			return m, nil
		}

		m.err = ""
		m.lastFetchAt = time.Now()
		m.matchView = true
		m.match = msg.match
		m.matchScroll = 0
		return m, nil
	}

	return m, nil
}

func (m *Model) toggleFocus() {
	if !m.selectorActive() {
		return
	}

	order := []focusArea{focusSeasons, focusCompetitions}

	for i := range order {
		if order[i] != m.focus {
			continue
		}
		m.focus = order[(i+1)%len(order)]
		return
	}

	m.focus = focusSeasons
}

func (m *Model) moveCursor(delta int) {
	if !m.selectorActive() {
		round := m.currentRound()
		if round == nil {
			return
		}
		m.fixtureCursor = clamp(m.fixtureCursor+delta, 0, len(round.Fixtures)-1)
		return
	}

	switch m.focus {
	case focusSeasons:
		m.seasonCursor = clamp(m.seasonCursor+delta, 0, len(m.seasons)-1)
	case focusCompetitions:
		m.competitionCursor = clamp(m.competitionCursor+delta, 0, len(m.competitions)-1)
	}
}

func (m *Model) shiftRound(delta int) {
	if m.matchView || m.selectorActive() || m.league == nil {
		return
	}

	m.roundCursor = clamp(m.roundCursor+delta, 0, len(m.league.Rounds)-1)
	m.fixtureCursor = 0
}

func (m *Model) handleEnter() tea.Cmd {
	m.loading = true
	m.err = ""

	if m.selectorActive() {
		switch m.focus {
		case focusSeasons:
			if len(m.seasons) == 0 {
				m.loading = false
				return nil
			}
			m.matchView = false
			m.match = nil
			season := m.currentSeason()
			if season == nil {
				m.loading = false
				return nil
			}
			return m.loadSeasonCompetitionsCmd(season.URL, seasonRequestKey(*season))

		case focusCompetitions:
			if len(m.competitions) == 0 {
				m.loading = false
				return nil
			}
			m.matchView = false
			m.match = nil
			competition := m.currentCompetition()
			if competition == nil {
				m.loading = false
				return nil
			}
			return m.loadLeagueCmd(competition.URL, competitionRequestKey(*competition))
		}

		m.loading = false
		return nil
	}

	fixture := m.currentFixture()
	if fixture == nil {
		m.loading = false
		return nil
	}
	m.matchView = true
	m.match = nil
	m.matchScroll = 0
	return m.loadMatchCmd(fixture.MatchURL, fixtureRequestKey(*fixture))
}

func (m *Model) handleReload() tea.Cmd {
	m.loading = true
	m.err = ""

	if m.matchView {
		fixture := m.currentFixture()
		if fixture == nil {
			m.loading = false
			return nil
		}
		m.match = nil
		m.matchScroll = 0
		return m.loadMatchCmd(fixture.MatchURL, fixtureRequestKey(*fixture))
	}

	m.match = nil
	if len(m.seasons) == 0 {
		return m.loadArchiveCmd("http://www.90minut.pl/archsezon.php")
	}

	if m.selectorActive() && m.focus == focusSeasons {
		season := m.currentSeason()
		if season == nil {
			m.loading = false
			return nil
		}
		return m.loadSeasonCompetitionsCmd(season.URL, seasonRequestKey(*season))
	}

	competition := m.currentCompetition()
	if competition == nil {
		season := m.currentSeason()
		if season == nil {
			m.loading = false
			return nil
		}
		return m.loadSeasonCompetitionsCmd(season.URL, seasonRequestKey(*season))
	}

	m.matchView = false
	return m.loadLeagueCmd(competition.URL, competitionRequestKey(*competition))
}

func (m *Model) scrollMatch(delta int) {
	if !m.matchView || delta == 0 {
		return
	}

	maxScroll := m.matchScrollLimit()
	m.matchScroll = clamp(m.matchScroll+delta, 0, maxScroll)
}
