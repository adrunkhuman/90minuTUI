package ui

import (
	"context"
	"errors"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

func testKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

func testRune(value rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: value, Text: string(value)})
}

func testCtrl(value rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: value, Mod: tea.ModCtrl})
}

type recordingLoader struct {
	archiveCalls int
	leagueCalls  int
	menuCalls    int
	matchCalls   int

	seasons      []site.Season
	selectedIdx  int
	competitions []site.Competition
	menus        map[string]*site.CompetitionMenu
	league       *site.LeaguePage
	match        *site.MatchPage

	// compErr, when non-nil, is returned by LoadCompetition instead of the league.
	compErr error
}

func (l *recordingLoader) LoadArchive(context.Context, string) ([]site.Season, int, []site.Competition, error) {
	l.archiveCalls++
	return l.seasons, l.selectedIdx, l.competitions, nil
}

func (l *recordingLoader) LoadLeague(context.Context, string) (*site.LeaguePage, error) {
	l.leagueCalls++
	return l.league, nil
}

func (l *recordingLoader) LoadCompetition(_ context.Context, rawURL string) (*site.CompetitionMenu, *site.LeaguePage, error) {
	l.menuCalls++
	if l.compErr != nil {
		return nil, nil, l.compErr
	}
	if menu, ok := l.menus[rawURL]; ok {
		return menu, nil, nil
	}
	l.leagueCalls++
	return nil, l.league, nil
}

func (l *recordingLoader) LoadMatch(context.Context, string) (*site.MatchPage, error) {
	l.matchCalls++
	return l.match, nil
}

func TestFixtureNavigationDoesNotReloadLeague(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	if loader.leagueCalls != 1 {
		t.Fatalf("expected one league load after startup, got %d", loader.leagueCalls)
	}

	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "1" {
		t.Fatalf("expected fixture #1 selected after startup")
	}

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyDown))
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

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyUp))
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

func TestPrintableFixtureNavigationKeys(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testRune('j'))
	if cmd != nil {
		t.Fatalf("expected no command on printable fixture move")
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "2" {
		t.Fatalf("expected printable j to select fixture #2")
	}

	m, cmd = updateModelWithMsg(t, m, testRune('k'))
	if cmd != nil {
		t.Fatalf("expected no command on printable fixture move back")
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "1" {
		t.Fatalf("expected printable k to select fixture #1")
	}

	m, cmd = updateModelWithMsg(t, m, testRune('l'))
	if cmd != nil {
		t.Fatalf("expected no command on printable round move")
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "3" {
		t.Fatalf("expected printable l to select next round fixture")
	}

	m, cmd = updateModelWithMsg(t, m, testRune('h'))
	if cmd != nil {
		t.Fatalf("expected no command on printable round move back")
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "1" {
		t.Fatalf("expected printable h to select previous round fixture")
	}
}

func TestPrintableReloadAndQuitKeys(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)

	m, cmd := updateModelWithMsg(t, m, testRune('r'))
	if cmd == nil {
		t.Fatalf("expected printable r to return reload command")
	}
	m, cmd = updateModelWithMsg(t, m, cmd())
	if cmd != nil {
		t.Fatalf("expected no chained command after reload")
	}
	if loader.leagueCalls != 2 {
		t.Fatalf("expected reload to load league again, got %d league loads", loader.leagueCalls)
	}

	_, cmd = updateModelWithMsg(t, m, testRune('q'))
	if cmd == nil {
		t.Fatalf("expected printable q to return quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected printable q command to emit tea.QuitMsg")
	}
}

func TestLoadingStateQuitKeys(t *testing.T) {
	for name, msg := range map[string]tea.Msg{
		"q":      testRune('q'),
		"ctrl+c": testCtrl('c'),
	} {
		t.Run(name, func(t *testing.T) {
			m := NewModel(newRecordingLoader())
			m.loading = true

			_, cmd := updateModelWithMsg(t, m, msg)
			if cmd == nil {
				t.Fatalf("expected %s while loading to return quit command", name)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("expected %s while loading to emit tea.QuitMsg", name)
			}
		})
	}
}

func TestLeagueLoadPrefersLatestCompletedFixtureWhenLeagueHasNoMatchLinks(t *testing.T) {
	loader := newRecordingLoader()
	loader.league.Rounds = []site.Round{
		{
			Name: "1. kolejka",
			Fixtures: []site.Fixture{
				{Home: "Team A", Away: "Team B", Score: "1-0"},
				{Home: "Team C", Away: "Team D", Score: "2-2"},
			},
		},
		{
			Name: "2. kolejka",
			Fixtures: []site.Fixture{
				{Home: "Team E", Away: "Team F", Score: "-"},
				{Home: "Team G", Away: "Team H", Score: "-"},
			},
		},
	}
	m := bootstrapLeagueLoadedModel(t, loader)

	if m.roundCursor != 0 {
		t.Fatalf("expected latest completed round selected, got %d", m.roundCursor)
	}
	if m.fixtureCursor != 1 {
		t.Fatalf("expected latest completed fixture selected, got %d", m.fixtureCursor)
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.Home != "Team C" || fixture.Score != "2-2" {
		t.Fatalf("expected latest completed fixture selected, got %+v", fixture)
	}
}

func TestLeagueLoadPrefersLatestDrillableFixtureWhenLeagueHasMixedLinks(t *testing.T) {
	loader := newRecordingLoader()
	loader.league.Rounds = []site.Round{
		{
			Name:     "1. kolejka",
			Fixtures: []site.Fixture{{Home: "Team A", Away: "Team B", Score: "1-0", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1", MatchID: "1"}},
		},
		{
			Name: "2. kolejka",
			Fixtures: []site.Fixture{
				{Home: "Team C", Away: "Team D", Score: "-"},
				{Home: "Team E", Away: "Team F", Score: "2-1", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2", MatchID: "2"},
			},
		},
	}
	m := bootstrapLeagueLoadedModel(t, loader)

	if m.roundCursor != 1 {
		t.Fatalf("expected latest round with drillable fixture selected, got %d", m.roundCursor)
	}
	if m.fixtureCursor != 1 {
		t.Fatalf("expected latest drillable fixture selected, got %d", m.fixtureCursor)
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "2" {
		t.Fatalf("expected latest drillable fixture selected, got %+v", fixture)
	}
}

func TestCompetitionEnterOpensIIILigaSubmenu(t *testing.T) {
	loader := newRecordingLoader()
	loader.competitions = []site.Competition{
		{Name: "PKO Bank Polski Ekstraklasa 2025/2026", URL: "http://www.90minut.pl/liga/1/liga14072.html", LeagueKey: "liga14072"},
		{Name: "III liga 2025/26", URL: "http://www.90minut.pl/ligireg.php?poziom=4&id_sezon=107", LeagueKey: "www.90minut.pl/ligireg.php?id_sezon=107&poziom=4"},
	}
	loader.menus = map[string]*site.CompetitionMenu{
		"http://www.90minut.pl/ligireg.php?poziom=4&id_sezon=107": {
			Title: "III liga 2025/26",
			Items: []site.Competition{{Name: "III liga 2025/26, gr. I", URL: "http://www.90minut.pl/liga/1/liga14154.html", LeagueKey: "liga14154"}},
		},
	}
	m := bootstrapLeagueLoadedModel(t, loader)

	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	m.competitionCursor = 1
	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected submenu load command")
	}
	m, cmd = updateModelWithMsg(t, m, cmd())
	if cmd != nil {
		t.Fatalf("expected submenu load to settle without chained command")
	}
	if !m.selectorVisible {
		t.Fatalf("expected selector to stay visible on submenu open")
	}
	if got := m.competitionTitle; got != "III liga 2025/26" {
		t.Fatalf("unexpected submenu title: %q", got)
	}
	if len(m.competitions) != 1 || m.competitions[0].Name != "III liga 2025/26, gr. I" {
		t.Fatalf("unexpected submenu items: %+v", m.competitions)
	}
	if len(m.competitionStack) != 1 {
		t.Fatalf("expected one previous menu on stack, got %d", len(m.competitionStack))
	}
	if loader.menuCalls < 2 {
		t.Fatalf("expected startup load plus submenu load, got %d", loader.menuCalls)
	}
}

func TestCompetitionEnterOpensWomenTierSubmenu(t *testing.T) {
	loader := newRecordingLoader()
	loader.competitions = []site.Competition{
		{Name: "Orlen Ekstraliga kobiet 2025/2026", URL: "http://www.90minut.pl/liga/1/liga14141.html", LeagueKey: "liga14141"},
		{Name: "III liga kobiet 2025/2026", URL: "http://www.90minut.pl/archiwum.php#women-tier=iii-liga-kobiet", LeagueKey: "women-tier:www.90minut.pl/archiwum.php:iii-liga-kobiet"},
	}
	loader.menus = map[string]*site.CompetitionMenu{
		"http://www.90minut.pl/archiwum.php#women-tier=iii-liga-kobiet": {
			Title: "III liga kobiet 2025/2026",
			Items: []site.Competition{{Name: "III liga kobiet 2025/2026, grupa: I", URL: "http://www.90minut.pl/liga/1/liga14578.html", LeagueKey: "liga14578"}},
		},
	}
	m := bootstrapLeagueLoadedModel(t, loader)

	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	m.competitionCursor = 1
	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected women tier submenu load command")
	}
	m, cmd = updateModelWithMsg(t, m, cmd())
	if cmd != nil {
		t.Fatalf("expected women tier submenu load to settle without chained command")
	}
	if got := m.competitionTitle; got != "III liga kobiet 2025/2026" {
		t.Fatalf("unexpected women submenu title: %q", got)
	}
	if len(m.competitions) != 1 || m.competitions[0].Name != "III liga kobiet 2025/2026, grupa: I" {
		t.Fatalf("unexpected women submenu items: %+v", m.competitions)
	}
}

func TestCompetitionEnterOpensFutsalTierSubmenu(t *testing.T) {
	loader := newRecordingLoader()
	loader.competitions = []site.Competition{
		{Name: "Fogo Futsal Ekstraklasa 2025/2026", URL: "http://www.90minut.pl/liga/1/liga14148.html", LeagueKey: "liga14148"},
		{Name: "I liga futsalu 2025/2026", URL: "http://www.90minut.pl/archiwum.php#futsal-tier=i-liga-futsalu", LeagueKey: "futsal-tier:www.90minut.pl/archiwum.php:i-liga-futsalu"},
	}
	loader.menus = map[string]*site.CompetitionMenu{
		"http://www.90minut.pl/archiwum.php#futsal-tier=i-liga-futsalu": {
			Title: "I liga futsalu 2025/2026",
			Items: []site.Competition{{Name: "I liga futsalu 2025/2026, grupa: południowa", URL: "http://www.90minut.pl/liga/1/liga14625.html", LeagueKey: "liga14625"}},
		},
	}
	m := bootstrapLeagueLoadedModel(t, loader)

	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	m.competitionCursor = 1
	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected futsal tier submenu load command")
	}
	m, cmd = updateModelWithMsg(t, m, cmd())
	if cmd != nil {
		t.Fatalf("expected futsal tier submenu load to settle without chained command")
	}
	if got := m.competitionTitle; got != "I liga futsalu 2025/2026" {
		t.Fatalf("unexpected futsal submenu title: %q", got)
	}
	if len(m.competitions) != 1 || m.competitions[0].Name != "I liga futsalu 2025/2026, grupa: południowa" {
		t.Fatalf("unexpected futsal submenu items: %+v", m.competitions)
	}
}

func TestCompetitionSubmenuEscReturnsToPreviousMenu(t *testing.T) {
	loader := newRecordingLoader()
	loader.competitions = []site.Competition{
		{Name: "PKO Bank Polski Ekstraklasa 2025/2026", URL: "http://www.90minut.pl/liga/1/liga14072.html", LeagueKey: "liga14072"},
		{Name: "Ligi regionalne 2025/26", URL: "http://www.90minut.pl/ligireg.php?id_sezon=107", LeagueKey: "www.90minut.pl/ligireg.php?id_sezon=107"},
	}
	loader.menus = map[string]*site.CompetitionMenu{
		"http://www.90minut.pl/ligireg.php?id_sezon=107": {
			Title: "Ligi regionalne 2025/26",
			Items: []site.Competition{{Name: "Dolnoslaski ZPN", URL: "http://www.90minut.pl/ligireg-16.html", LeagueKey: "www.90minut.pl/ligireg-16.html"}},
		},
		"http://www.90minut.pl/ligireg-16.html": {
			Title: "Ligi regionalne 2025/26 - Dolnoslaski ZPN",
			Items: []site.Competition{{Name: "IV liga 2025/2026, grupa: dolnoslaska", URL: "http://www.90minut.pl/liga/1/liga14169.html", LeagueKey: "liga14169"}},
		},
	}
	m := bootstrapLeagueLoadedModel(t, loader)

	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	m.competitionCursor = 1
	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	m, _ = updateModelWithMsg(t, m, cmd())
	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	m, _ = updateModelWithMsg(t, m, cmd())

	if got := m.competitionTitle; got != "Ligi regionalne 2025/26 - Dolnoslaski ZPN" {
		t.Fatalf("unexpected nested submenu title: %q", got)
	}

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	if cmd != nil {
		t.Fatalf("expected esc in submenu to pop without command")
	}
	if got := m.competitionTitle; got != "Ligi regionalne 2025/26" {
		t.Fatalf("expected previous submenu title, got %q", got)
	}
	if len(m.competitions) != 1 || m.competitions[0].Name != "Dolnoslaski ZPN" {
		t.Fatalf("expected previous submenu items restored, got %+v", m.competitions)
	}
}

func TestStaleCompetitionLoadErrorDoesNotOverwriteCurrentSelection(t *testing.T) {
	loader := newRecordingLoader()
	loader.competitions = []site.Competition{
		{Name: "Ekstraklasa", URL: "http://www.90minut.pl/liga/1/liga11233.html", LeagueKey: "liga11233"},
		{Name: "Ligi regionalne 2025/26", URL: "http://www.90minut.pl/ligireg.php?id_sezon=107", LeagueKey: "www.90minut.pl/ligireg.php?id_sezon=107"},
	}
	m := bootstrapLeagueLoadedModel(t, loader)

	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	m.competitionCursor = 1
	staleKey := competitionRequestKey(m.competitions[m.competitionCursor])
	m.competitionCursor = 0
	currentKey := competitionRequestKey(m.competitions[m.competitionCursor])

	m, cmd := updateModelWithMsg(t, m, competitionMenuLoadedMsg{competitionKey: staleKey, err: errors.New("stale submenu failed")})
	if cmd != nil {
		t.Fatalf("expected stale async error to be ignored")
	}
	if m.err != "" {
		t.Fatalf("expected stale error to be ignored, got %q", m.err)
	}

	m, cmd = updateModelWithMsg(t, m, competitionMenuLoadedMsg{competitionKey: currentKey, err: errors.New("current submenu failed")})
	if cmd != nil {
		t.Fatalf("expected current async error to settle without command")
	}
	if m.err != "current submenu failed" {
		t.Fatalf("expected current error to be shown, got %q", m.err)
	}
}

func TestFixtureEnterLoadsMatchWithoutReloadingLeague(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
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

func TestFixtureEnterKeepsLeagueViewWhenFixtureHasNoDetails(t *testing.T) {
	loader := newRecordingLoader()
	loader.league.Rounds[0].Fixtures[0].MatchURL = ""
	loader.league.Rounds[0].Fixtures[0].MatchID = ""
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd != nil {
		t.Fatalf("expected no command for non-drillable fixture")
	}
	if m.matchView {
		t.Fatalf("expected league view to stay open for non-drillable fixture")
	}
	if m.match != nil {
		t.Fatalf("expected no match payload for non-drillable fixture")
	}
	if m.err != unavailableFixtureMatchDetailsMessage {
		t.Fatalf("unexpected unavailable-details message: %q", m.err)
	}
	if loader.matchCalls != 0 {
		t.Fatalf("expected no match loads, got %d", loader.matchCalls)
	}
}

func TestMatchViewNavigationFallsBackToLeagueWhenNextFixtureHasNoDetails(t *testing.T) {
	loader := newRecordingLoader()
	loader.league.Rounds[0].Fixtures[1].MatchURL = ""
	loader.league.Rounds[0].Fixtures[1].MatchID = ""
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected match load command on enter")
	}
	m, _ = updateModelWithMsg(t, m, cmd())

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyDown))
	if cmd != nil {
		t.Fatalf("expected no load command for non-drillable fixture in match view")
	}
	if m.matchView {
		t.Fatalf("expected navigation to return to league view for non-drillable fixture")
	}
	if m.match != nil {
		t.Fatalf("expected stale match details to clear when fixture has no details")
	}
	if m.err != unavailableFixtureMatchDetailsMessage {
		t.Fatalf("unexpected unavailable-details message: %q", m.err)
	}
	if loader.matchCalls != 1 {
		t.Fatalf("expected no extra match load, got %d", loader.matchCalls)
	}
}

func TestFixtureEnterUsesFixtureSpecificMessageWhenLeagueHasOtherDetails(t *testing.T) {
	loader := newRecordingLoader()
	loader.league.Rounds[0].Fixtures[0].MatchURL = ""
	loader.league.Rounds[0].Fixtures[0].MatchID = ""
	loader.league.Rounds[0].Fixtures[1].MatchURL = "http://www.90minut.pl/mecz.php?id_mecz=2"
	loader.league.Rounds[0].Fixtures[1].MatchID = "2"
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd != nil {
		t.Fatalf("expected no command for non-drillable fixture")
	}
	if m.err != unavailableFixtureMatchDetailsMessage {
		t.Fatalf("unexpected fixture-specific message: %q", m.err)
	}
}

func TestMatchViewNavigationLoadsAdjacentFixture(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected match load command on enter")
	}
	m, _ = updateModelWithMsg(t, m, cmd())

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyDown))
	if cmd == nil {
		t.Fatalf("expected moving in match view to load adjacent fixture")
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "2" {
		t.Fatalf("expected second fixture selected in match view")
	}
	if !m.matchView || !m.loading || m.match != nil {
		t.Fatalf("expected match view to stay open and reload selected fixture")
	}

	m, _ = updateModelWithMsg(t, m, cmd())
	if loader.matchCalls != 2 {
		t.Fatalf("expected second match load after navigating in match view, got %d", loader.matchCalls)
	}
}

func TestMatchViewRoundNavigationLoadsFirstFixture(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected match load command on enter")
	}
	m, _ = updateModelWithMsg(t, m, cmd())
	m.fixtureCursor = 1

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyRight))
	if cmd == nil {
		t.Fatalf("expected round change in match view to load first fixture")
	}
	if m.roundCursor != 1 {
		t.Fatalf("expected second round selected, got %d", m.roundCursor)
	}
	if m.fixtureCursor != 0 {
		t.Fatalf("expected first fixture selected after round change, got %d", m.fixtureCursor)
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "3" {
		t.Fatalf("expected first fixture from next round selected")
	}

	m, _ = updateModelWithMsg(t, m, cmd())
	if loader.matchCalls != 2 {
		t.Fatalf("expected new match load after round change, got %d", loader.matchCalls)
	}
}

func TestMatchViewScrollKeysDoNotChangeFixture(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected match load command on enter")
	}
	m, _ = updateModelWithMsg(t, m, cmd())
	m.match.HomeLineup = make([]site.PlayerLine, 0, 60)
	for i := 1; i <= 60; i++ {
		m.match.HomeLineup = append(m.match.HomeLineup, site.PlayerLine{Name: fmt.Sprintf("Player%02d", i)})
	}

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyPgDown))
	if cmd != nil {
		t.Fatalf("expected scroll keys not to load another fixture")
	}
	if m.matchScroll == 0 {
		t.Fatalf("expected page down to scroll match content")
	}
	if fixture := m.currentFixture(); fixture == nil || fixture.MatchID != "1" {
		t.Fatalf("expected current fixture to stay selected while scrolling")
	}
	if loader.matchCalls != 1 {
		t.Fatalf("expected no extra match loads while scrolling, got %d", loader.matchCalls)
	}
}

func TestMatchViewRoundNavigationClearsDetailForEmptyRound(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)
	m.roundCursor = 0
	m.fixtureCursor = 0
	m.league.Rounds = []site.Round{
		m.league.Rounds[0],
		{Name: "2. kolejka"},
	}

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatalf("expected match load command on enter")
	}
	m, _ = updateModelWithMsg(t, m, cmd())

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyRight))
	if cmd != nil {
		t.Fatalf("expected no load command for empty round")
	}
	if m.roundCursor != 1 {
		t.Fatalf("expected empty round selected, got %d", m.roundCursor)
	}
	if m.match != nil {
		t.Fatalf("expected stale match details to clear on empty round")
	}
	if !m.matchView {
		t.Fatalf("expected match view to stay open on empty round")
	}
}

func TestEscapeFromLeagueViewTogglesSelectorPopup(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)

	m, cmd := updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	if cmd != nil {
		t.Fatalf("expected no command when opening selector popup")
	}
	if m.league == nil {
		t.Fatalf("expected league to stay loaded when opening selector popup")
	}
	if !m.selectorVisible {
		t.Fatalf("expected escape to open selector popup")
	}
	if m.focus != focusCompetitions {
		t.Fatalf("expected selector popup to focus competitions, got %v", m.focus)
	}

	m, cmd = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	if cmd != nil {
		t.Fatalf("expected no command when closing selector popup")
	}
	if m.selectorVisible {
		t.Fatalf("expected second escape to close selector popup")
	}
	if m.focus != focusFixtures {
		t.Fatalf("expected closing selector popup to restore fixtures focus, got %v", m.focus)
	}
}

func TestAnchoredWindowBoundsKeepsSelectedStandingVisible(t *testing.T) {
	start, end := anchoredWindowBounds(30, []int{1, 18}, 6)
	if !(start <= 1 && 1 < end) {
		t.Fatalf("expected anchored window to include at least one selected row, got start=%d end=%d", start, end)
	}
}

func TestSelectorNotVisibleAfterSuccessfulStartup(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)

	if m.selectorVisible {
		t.Fatalf("expected selector to be closed after successful league load")
	}
	if m.focus != focusFixtures {
		t.Fatalf("expected fixtures focus after startup, got %v", m.focus)
	}
}

func TestSelectorBecomesVisibleAfterCompetitionLoadError(t *testing.T) {
	loader := newRecordingLoader()
	loader.compErr = fmt.Errorf("network error")

	m := NewModel(loader)
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected init command")
	}

	// Drain the startup chain to completion.
	m, cmd = updateModelWithMsg(t, m, cmd())
	if cmd == nil {
		t.Fatalf("expected archive load to schedule competition load")
	}
	for cmd != nil {
		m, cmd = updateModelWithMsg(t, m, cmd())
	}

	if m.league != nil {
		t.Fatalf("expected no league after competition load error")
	}
	if !m.selectorVisible {
		t.Fatalf("expected selector to be open after competition load error")
	}
	if m.err == "" {
		t.Fatalf("expected error message to be set")
	}
}

func TestSelectorBecomesVisibleWhenArchiveHasNoCompetitions(t *testing.T) {
	loader := newRecordingLoader()
	loader.competitions = nil

	m := NewModel(loader)
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected init command")
	}

	// Drain the startup chain; archive loads but no competition load follows.
	for cmd != nil {
		m, cmd = updateModelWithMsg(t, m, cmd())
	}

	if m.league != nil {
		t.Fatalf("expected no league when archive has no competitions")
	}
	if !m.selectorVisible {
		t.Fatalf("expected selector to be open when archive has no competitions")
	}
}

func TestToggleFocusCyclesBetweenSeasonsAndCompetitions(t *testing.T) {
	loader := newRecordingLoader()
	m := bootstrapLeagueLoadedModel(t, loader)

	// Open the selector popup.
	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyEsc))
	if !m.selectorVisible {
		t.Fatalf("expected selector open after escape")
	}
	if m.focus != focusCompetitions {
		t.Fatalf("expected competitions focus after opening selector, got %v", m.focus)
	}

	// Tab cycles to seasons.
	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyTab))
	if m.focus != focusSeasons {
		t.Fatalf("expected seasons focus after first tab, got %v", m.focus)
	}

	// Tab cycles back to competitions.
	m, _ = updateModelWithMsg(t, m, testKey(tea.KeyTab))
	if m.focus != focusCompetitions {
		t.Fatalf("expected competitions focus after second tab, got %v", m.focus)
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

	for cmd != nil {
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
	m.width = 120
	m.height = 40

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
		menus:       map[string]*site.CompetitionMenu{},
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
			Rounds: []site.Round{
				{
					Name: "1. kolejka",
					Fixtures: []site.Fixture{
						{Home: "Team A", Away: "Team B", Score: "1-0", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1", MatchID: "1"},
						{Home: "Team C", Away: "Team D", Score: "2-2", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2", MatchID: "2"},
					},
				},
				{
					Name: "2. kolejka",
					Fixtures: []site.Fixture{
						{Home: "Team E", Away: "Team F", Score: "0-0", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=3", MatchID: "3"},
					},
				},
			},
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
