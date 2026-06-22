package ui

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

type focusArea int

const (
	focusSeasons focusArea = iota
	focusCompetitions
	focusFixtures
)

const unavailableCompetitionMatchDetailsMessage = "Match details unavailable for this competition"
const unavailableFixtureMatchDetailsMessage = "Match details unavailable for this fixture"

var fixtureResultRe = regexp.MustCompile(`^\d+\s*-\s*\d+$`)

type archiveLoader interface {
	LoadArchive(ctx context.Context, archiveURL string) ([]site.Season, int, []site.Competition, error)
	LoadCompetition(ctx context.Context, competitionURL string) (*site.CompetitionMenu, *site.LeaguePage, error)
	LoadLeague(ctx context.Context, leagueURL string) (*site.LeaguePage, error)
	LoadMatch(ctx context.Context, matchURL string) (*site.MatchPage, error)
}

type archiveLoadedMsg struct {
	seasons      []site.Season
	selectedIdx  int
	competitions []site.Competition
	err          error
}

type competitionsLoadedMsg struct {
	seasonKey    string
	competitions []site.Competition
	selectorOnly bool
	err          error
}

type competitionMenuLoadedMsg struct {
	competitionKey string
	menu           *site.CompetitionMenu
	league         *site.LeaguePage
	err            error
}

type competitionMenuState struct {
	title  string
	items  []site.Competition
	cursor int
}

type matchLoadedMsg struct {
	fixtureKey string
	match      *site.MatchPage
	err        error
}

type Model struct {
	service archiveLoader

	width  int
	height int

	loading bool
	err     string
	focus   focusArea

	seasons      []site.Season
	seasonCursor int

	competitions      []site.Competition
	competitionCursor int
	competitionTitle  string
	competitionStack  []competitionMenuState

	league        *site.LeaguePage
	roundCursor   int
	fixtureCursor int

	matchView   bool
	match       *site.MatchPage
	matchScroll int

	selectorVisible bool
	suppressTopBar  bool
	lastFetchAt     time.Time
}

func NewModel(svc archiveLoader) Model {
	return Model{service: svc, focus: focusCompetitions, loading: true}
}

func (m Model) selectorActive() bool {
	return m.selectorVisible
}

func (m *Model) openSelector() {
	m.selectorVisible = true
	if m.focus == focusFixtures {
		m.focus = focusCompetitions
	}
}

func (m *Model) closeSelector() {
	m.selectorVisible = false
	if m.league != nil {
		m.focus = focusFixtures
	}
}

func (m Model) currentRound() *site.Round {
	if m.league == nil || len(m.league.Rounds) == 0 {
		return nil
	}
	if m.roundCursor < 0 || m.roundCursor >= len(m.league.Rounds) {
		return nil
	}
	return &m.league.Rounds[m.roundCursor]
}

func (m Model) currentSeason() *site.Season {
	if len(m.seasons) == 0 {
		return nil
	}
	if m.seasonCursor < 0 || m.seasonCursor >= len(m.seasons) {
		return nil
	}
	return &m.seasons[m.seasonCursor]
}

func (m Model) currentCompetition() *site.Competition {
	if len(m.competitions) == 0 {
		return nil
	}
	if m.competitionCursor < 0 || m.competitionCursor >= len(m.competitions) {
		return nil
	}
	return &m.competitions[m.competitionCursor]
}

func (m *Model) resetCompetitionMenu(title string, items []site.Competition) {
	m.competitionTitle = title
	m.competitions = items
	m.competitionCursor = m.preferredCompetitionIndex()
	m.competitionStack = nil
}

func (m *Model) pushCompetitionMenu(title string, items []site.Competition) {
	m.competitionStack = append(m.competitionStack, competitionMenuState{
		title:  m.competitionTitle,
		items:  append([]site.Competition(nil), m.competitions...),
		cursor: m.competitionCursor,
	})
	m.competitionTitle = title
	m.competitions = items
	m.competitionCursor = 0
}

func (m *Model) popCompetitionMenu() bool {
	if len(m.competitionStack) == 0 {
		return false
	}
	prev := m.competitionStack[len(m.competitionStack)-1]
	m.competitionStack = m.competitionStack[:len(m.competitionStack)-1]
	m.competitionTitle = prev.title
	m.competitions = prev.items
	m.competitionCursor = clamp(prev.cursor, 0, len(prev.items)-1)
	return true
}

func (m Model) currentFixture() *site.Fixture {
	round := m.currentRound()
	if round == nil || len(round.Fixtures) == 0 {
		return nil
	}
	if m.fixtureCursor < 0 || m.fixtureCursor >= len(round.Fixtures) {
		return nil
	}
	return &round.Fixtures[m.fixtureCursor]
}

func (m Model) currentFixtureDrillable() bool {
	fixture := m.currentFixture()
	return fixture != nil && strings.TrimSpace(fixture.MatchURL) != ""
}

func (m Model) unavailableMatchDetailsMessage() string {
	if m.leagueHasDrillableFixtures() {
		return unavailableFixtureMatchDetailsMessage
	}
	return unavailableCompetitionMatchDetailsMessage
}

func (m Model) initialFixtureSelection() (int, int) {
	if m.league == nil || len(m.league.Rounds) == 0 {
		return 0, 0
	}

	if m.leagueHasDrillableFixtures() {
		for roundIdx := len(m.league.Rounds) - 1; roundIdx >= 0; roundIdx-- {
			fixtures := m.league.Rounds[roundIdx].Fixtures
			for fixtureIdx := len(fixtures) - 1; fixtureIdx >= 0; fixtureIdx-- {
				if strings.TrimSpace(fixtures[fixtureIdx].MatchURL) == "" {
					continue
				}
				return roundIdx, fixtureIdx
			}
		}
	}

	for roundIdx := len(m.league.Rounds) - 1; roundIdx >= 0; roundIdx-- {
		fixtures := m.league.Rounds[roundIdx].Fixtures
		for fixtureIdx := len(fixtures) - 1; fixtureIdx >= 0; fixtureIdx-- {
			if !fixtureHasResult(fixtures[fixtureIdx]) {
				continue
			}
			return roundIdx, fixtureIdx
		}
	}

	return clamp(len(m.league.Rounds)-1, 0, len(m.league.Rounds)-1), 0
}

func (m Model) leagueHasDrillableFixtures() bool {
	if m.league == nil {
		return false
	}
	for _, round := range m.league.Rounds {
		for _, fixture := range round.Fixtures {
			if strings.TrimSpace(fixture.MatchURL) != "" {
				return true
			}
		}
	}
	return false
}

func fixtureHasResult(fixture site.Fixture) bool {
	return fixtureResultRe.MatchString(strings.TrimSpace(fixture.Score))
}

func (m Model) preferredCompetitionIndex() int {
	if len(m.competitions) == 0 {
		return 0
	}
	for i, c := range m.competitions {
		// Bias the default load toward Ekstraklasa without persisting user prefs.
		if strings.Contains(strings.ToLower(c.Name), "ekstraklasa") {
			return i
		}
	}
	return 0
}
