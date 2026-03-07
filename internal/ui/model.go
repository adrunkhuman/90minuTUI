package ui

import (
	"context"
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

type focusArea int

const (
	focusSeasons focusArea = iota
	focusCompetitions
	focusFixtures
)

type archiveLoader interface {
	LoadArchive(ctx context.Context, archiveURL string) ([]site.Season, int, []site.Competition, error)
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
	err          error
}

type leagueLoadedMsg struct {
	competitionKey string
	league         *site.LeaguePage
	err            error
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

	league        *site.LeaguePage
	roundCursor   int
	fixtureCursor int

	matchView bool
	match     *site.MatchPage

	sidebarCollapsed bool
}

func NewModel(svc archiveLoader) Model {
	return Model{service: svc, focus: focusCompetitions, loading: true}
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

func (m Model) preferredCompetitionIndex() int {
	if len(m.competitions) == 0 {
		return 0
	}
	for i, c := range m.competitions {
		if strings.Contains(strings.ToLower(c.Name), "ekstraklasa") {
			return i
		}
	}
	return 0
}
