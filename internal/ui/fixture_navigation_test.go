package ui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

type recordingLoader struct {
	archiveCalls int
	leagueCalls  int
	matchCalls   int

	seasons      []site.Season
	selectedIdx  int
	competitions []site.Competition
	league       *site.LeaguePage
	match        *site.MatchPage
}

func (l *recordingLoader) LoadArchive(context.Context, string) ([]site.Season, int, []site.Competition, error) {
	l.archiveCalls++
	return l.seasons, l.selectedIdx, l.competitions, nil
}

func (l *recordingLoader) LoadLeague(context.Context, string) (*site.LeaguePage, error) {
	l.leagueCalls++
	return l.league, nil
}

func (l *recordingLoader) LoadMatch(context.Context, string) (*site.MatchPage, error) {
	l.matchCalls++
	return l.match, nil
}

func TestFixtureNavigationDoesNotReloadLeague(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)

	if loader.leagueCalls != 1 {
		t.Fatalf("expected one league load after startup, got %d", loader.leagueCalls)
	}

	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "1" {
		t.Fatalf("expected fixture #1 selected after startup")
	}

	m, cmd := updateModelWithMsg(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("expected no command on fixture cursor move down")
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "2" {
		t.Fatalf("expected fixture #2 selected after moving down")
	}
	if loader.leagueCalls != 1 {
		t.Fatalf("fixture cursor move should not reload league, got %d league loads", loader.leagueCalls)
	}
	if loader.matchCalls != 0 {
		t.Fatalf("fixture cursor move should not load match, got %d match loads", loader.matchCalls)
	}

	m, cmd = updateModelWithMsg(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected no command on fixture cursor move up")
	}
	if loader.leagueCalls != 1 {
		t.Fatalf("switching back to fixture #1 should not reload league, got %d league loads", loader.leagueCalls)
	}
	if loader.matchCalls != 0 {
		t.Fatalf("fixture cursor move should not load match, got %d match loads", loader.matchCalls)
	}

	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "1" {
		t.Fatalf("expected fixture #1 selected after moving back")
	}
}

func TestFixtureEnterLoadsMatchWithoutReloadingLeague(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)

	m, cmd := updateModelWithMsg(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected enter on fixture to return loadMatch command")
	}
	if !m.matchView {
		t.Fatalf("expected immediate transition into match view while loading")
	}
	if m.match != nil {
		t.Fatalf("expected match details to remain empty until load completes")
	}
	if loader.leagueCalls != 1 {
		t.Fatalf("expected league load count to stay at 1 before running match command, got %d", loader.leagueCalls)
	}

	m, cmd = updateModelWithMsg(t, m, cmd())
	if cmd != nil {
		t.Fatalf("expected no chained command after match load")
	}
	if loader.matchCalls != 1 {
		t.Fatalf("expected one match load, got %d", loader.matchCalls)
	}
	if loader.leagueCalls != 1 {
		t.Fatalf("match open should not trigger league reload, got %d league loads", loader.leagueCalls)
	}
	if !m.matchView || m.match == nil || m.match.MatchID != "1" {
		t.Fatalf("expected match view for fixture #1")
	}
}

func TestEscapeFromLeagueViewReturnsToSelector(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)

	m, cmd := updateModelWithMsg(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected no command when returning to selector")
	}
	if m.league != nil {
		t.Fatalf("expected escape to leave league view and return to selector")
	}
	if m.focus != focusCompetitions {
		t.Fatalf("expected selector to focus competitions after leaving league view, got %v", m.focus)
	}
}

func bootstrapLeagueLoadedModel(t *testing.T, loader *recordingLoader) Model {
	t.Helper()

	m := NewModel(loader)
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected init command")
	}

	m, cmd = updateModelWithMsg(t, m, cmd())
	if cmd == nil {
		t.Fatalf("expected archive loaded flow to schedule league load")
	}

	m, cmd = updateModelWithMsg(t, m, cmd())
	for i := 0; i < 4 && cmd != nil; i++ {
		m, cmd = updateModelWithMsg(t, m, cmd())
	}

	if loader.leagueCalls != 1 {
		t.Fatalf("expected one league load after startup, got %d", loader.leagueCalls)
	}
	if m.league == nil {
		t.Fatalf("expected league to be loaded")
	}
	if m.focus != focusFixtures {
		t.Fatalf("expected fixtures focus after league load")
	}

	return m
}

func updateModelWithMsg(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(msg)
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("unexpected model type %T", next)
	}

	return updated, cmd
}

func newRecordingLoader() *recordingLoader {
	return &recordingLoader{
		seasons: []site.Season{{
			Label:    "2024/2025",
			URL:      "http://www.90minut.pl/archsezon.php?id_sezon=97",
			SeasonID: "97",
			Current:  true,
		}},
		selectedIdx: 0,
		competitions: []site.Competition{{
			Name:      "Ekstraklasa",
			URL:       "http://www.90minut.pl/liga/1/liga11233.html",
			LeagueKey: "liga11233",
		}},
		league: &site.LeaguePage{
			Title:     "Ekstraklasa",
			URL:       "http://www.90minut.pl/liga/1/liga11233.html",
			LeagueKey: "liga11233",
			Standings: []site.StandingRow{{Position: 1, Team: "Team A", Played: 1, Won: 1, Drawn: 0, Lost: 0, Points: 3}, {Position: 2, Team: "Team B", Played: 1, Won: 0, Drawn: 0, Lost: 1, Points: 0}},
			Rounds: []site.Round{{
				Name: "1. kolejka",
				Fixtures: []site.Fixture{
					{Home: "Team A", Away: "Team B", Score: "1-0", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1", MatchID: "1"},
					{Home: "Team C", Away: "Team D", Score: "2-2", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2", MatchID: "2"},
				},
			}},
		},
		match: &site.MatchPage{
			URL:      "http://www.90minut.pl/mecz.php?id_mecz=1",
			MatchID:  "1",
			HomeTeam: "Team A",
			AwayTeam: "Team B",
			Score:    "1-0",
		},
	}
}
