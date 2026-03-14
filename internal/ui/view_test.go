package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

func TestLeagueSketchViewShowsStandingsFixturesAndStatus(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 18
	m.lastFetchAt = time.Date(2026, time.March, 10, 21, 15, 0, 0, time.UTC)
	m.league.Title = "PKO Bank Polski Ekstraklasa 2025/2026"

	view := m.View()
	for _, want := range []string{
		"PKO Bank Polski Ekstraklasa 2025/2026",
		"Standings",
		"# Team",
		"Legia Warszawa",
		"Fixtures",
		"Round 1",
		"Legia Warszawa 2-1 Lech Poznan | 24/01 20:30",
		"fetched: 21:15:00",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q\n%s", want, view)
		}
	}
}

func TestLeagueSketchViewShowsLeagueTitleOnlyOnce(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 18
	m.league.Title = "PKO Bank Polski Ekstraklasa 2025/2026"

	view := m.View()
	if got := strings.Count(view, "PKO Bank Polski Ekstraklasa 2025/2026"); got != 1 {
		t.Fatalf("expected league title once, got %d\n%s", got, view)
	}
}

func TestLeagueViewUsesTopContextBar(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 18
	m.league.Title = "PKO Bank Polski Ekstraklasa 2025/2026"

	lines := strings.Split(m.View(), "\n")
	if len(lines) == 0 || !strings.Contains(lines[0], "PKO Bank Polski Ekstraklasa 2025/2026") {
		t.Fatalf("expected top line to show competition context\n%s", m.View())
	}
	if strings.Contains(m.View(), "Fixtures\nPKO Bank Polski Ekstraklasa 2025/2026") {
		t.Fatalf("expected fixtures pane to avoid repeating competition context\n%s", m.View())
	}
}

func TestStartupViewDoesNotFlashSelectorPopup(t *testing.T) {
	m := NewModel(nil)
	m.width = 120
	m.height = 18

	view := m.View()
	if strings.Contains(view, "Season + league") {
		t.Fatalf("expected startup view to avoid selector popup\n%s", view)
	}
	if got := strings.Count(view, "\n") + 1; got != m.height {
		t.Fatalf("expected startup view to fill terminal height, got %d lines for height %d\n%s", got, m.height, view)
	}
}

func TestMatchSketchViewShowsLoadingState(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.matchView = true
	m.loading = true

	view := m.View()
	for _, want := range []string{
		"Standings",
		"Fixtures",
		"LEG 2-1 LEC",
		"Loading match details...",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q\n%s", want, view)
		}
	}
}

func TestMatchDetailRemovesRedundantMetadata(t *testing.T) {
	m := sketchModel()
	m.width = 140
	m.matchView = true
	m.league.Title = "PKO Bank Polski Ekstraklasa 2025/2026"
	m.match = &site.MatchPage{
		HomeTeam:    "Bruk-Bet Termalica Nieciecza",
		AwayTeam:    "Motor Lublin",
		Score:       "1-2",
		Competition: "PKO Bank Polski Ekstraklasa 2025/2026 - Kolejka 25",
		Meta:        "13 marca 2026, 18:00 3542 Damian Kos",
		Weather:     "15 C",
		Events: []site.MatchEvent{{
			MinuteText: "17",
			Kind:       "GOAL",
			TeamSide:   "home",
			Text:       "Krzysztof Kubica 17",
		}},
		NewsTitle: "PKO BP Ekstraklasa: Bruk-Bet Termalica 1-2 Motor",
		NewsURL:   "http://www.90minut.pl/news/example.html",
	}

	view := m.View()
	for _, want := range []string{
		"PKO Bank Polski Ekstraklasa 2025/2026",
		"13 March 2026, 18:00 | Attendance 3542 | Ref. Damian Kos | Weather 15 C",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected match view to contain %q\n%s", want, view)
		}
	}
	for _, unwanted := range []string{
		"Strona główna",
		"Kolejka 25",
		"PKO Bank Polski Ekstraklasa 2025/2026 - Round 25",
		"GOAL Krzysztof Kubica 17",
		"Related News",
		"http://www.90minut.pl/news/example.html",
	} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("expected match view to omit %q\n%s", unwanted, view)
		}
	}
}

func TestLayoutWidthsFavorWiderLeftPane(t *testing.T) {
	leagueLeft, leagueRight := leagueLayoutWidths(120)
	if leagueLeft <= leagueRight/2 {
		t.Fatalf("expected league left pane to be materially wider, got left=%d right=%d", leagueLeft, leagueRight)
	}

	matchLeft, matchCenter, _ := matchLayoutWidths(120)
	if matchLeft < 40 {
		t.Fatalf("expected match left pane widened, got %d", matchLeft)
	}
	if matchCenter <= matchLeft {
		t.Fatalf("expected match center pane to remain dominant, got left=%d center=%d", matchLeft, matchCenter)
	}
}

func TestFormatFixtureWhenInfoShortensDateAndDropsAttendance(t *testing.T) {
	if got := formatFixtureWhenInfo("28 stycznia, 21:00 (51 719)"); got != "28/01 21:00" {
		t.Fatalf("unexpected fixture when info: %q", got)
	}
}

func TestRenderFixtureWindowUsesFullNamesOutsideMatchSidebar(t *testing.T) {
	lines := renderFixtureWindow([]site.Fixture{{
		Home:     "Legia Warszawa",
		Away:     "Lech Poznan",
		Score:    "2-1",
		WhenInfo: "24 stycznia, 20:30 (16 580)",
	}}, 0, 5, false)

	if len(lines) != 1 || !strings.Contains(lines[0], "Legia Warszawa 2-1 Lech Poznan | 24/01 20:30") {
		t.Fatalf("expected full fixture line, got %v", lines)
	}
}

func TestRenderFixtureWindowUsesCompactNamesInMatchSidebar(t *testing.T) {
	lines := renderFixtureWindow([]site.Fixture{{
		Home:     "Legia Warszawa",
		Away:     "Lech Poznan",
		Score:    "2-1",
		WhenInfo: "24 stycznia, 20:30 (16 580)",
	}}, 0, 5, true)

	if len(lines) != 1 || !strings.Contains(lines[0], "LEG 2-1 LEC | 24/01 20:30") {
		t.Fatalf("expected compact fixture line, got %v", lines)
	}
}

func TestLeagueViewCanShowSelectorPopup(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 40
	m.selectorVisible = true
	m.focus = focusCompetitions

	view := m.View()
	for _, want := range []string{
		"Standings",
		"Legia Warszawa 2-1 Lech Poznan",
		"Season + league",
		"2024/2025",
		"Ekstraklasa",
		"esc: close",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q\n%s", want, view)
		}
	}
	if got := strings.Count(view, "\n") + 1; got != m.height {
		t.Fatalf("expected popup view to fill terminal height, got %d lines for height %d\n%s", got, m.height, view)
	}
}

func TestSelectorPopupPlacesSeasonsAndLeaguesSideBySide(t *testing.T) {
	m := sketchModel()
	m.focus = focusCompetitions

	popup := m.selectorPopupView(60)
	for _, line := range strings.Split(popup, "\n") {
		if strings.Contains(line, "Season") && strings.Contains(line, "Leagues") {
			return
		}
	}

	t.Fatalf("expected popup headings on the same row\n%s", popup)
}

func TestSelectorPaneWidthsFavorLeagues(t *testing.T) {
	left, right := selectorPaneWidths(40, renderSeasonsWindow(sketchModel().seasons, 0))
	if left >= right {
		t.Fatalf("expected leagues pane wider than seasons pane, got left=%d right=%d", left, right)
	}
}

func TestOverlayLinePreservesContentOutsidePopup(t *testing.T) {
	got := overlayLine("left-center-right", "     POPUP", 5)
	if !strings.HasPrefix(got, "left-") {
		t.Fatalf("expected left side preserved, got %q", got)
	}
	if !strings.HasSuffix(got, "ht") {
		t.Fatalf("expected right side preserved, got %q", got)
	}
}

func TestSelectorPopupHandlesShortTerminal(t *testing.T) {
	m := sketchModel()
	m.width = 60
	m.height = 6
	m.selectorVisible = true
	m.focus = focusCompetitions

	view := m.View()
	for _, want := range []string{"Season + league", "Fixtures"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q\n%s", want, view)
		}
	}
}

func TestLeagueViewClipsListsToTerminalHeight(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 12
	m.league.Standings = make([]site.StandingRow, 0, 20)
	m.league.Rounds[0].Fixtures = make([]site.Fixture, 0, 20)
	for i := 1; i <= 20; i++ {
		team := fmt.Sprintf("Team %02d", i)
		m.league.Standings = append(m.league.Standings, site.StandingRow{Position: i, Team: team, Played: 8, Won: 4, Drawn: 2, Lost: 2, Points: 14 - (i / 2)})
	}
	for i := 1; i <= 20; i++ {
		home := fmt.Sprintf("Team %02d", i)
		away := fmt.Sprintf("Team %02d", (i%20)+1)
		m.league.Rounds[0].Fixtures = append(m.league.Rounds[0].Fixtures, site.Fixture{Home: home, Away: away, Score: "1-0", WhenInfo: fmt.Sprintf("slot-%02d", i)})
	}
	m.fixtureCursor = 17

	view := m.View()
	if got := strings.Count(view, "\n") + 1; got > m.height {
		t.Fatalf("expected clipped view to fit terminal height, got %d lines for height %d\n%s", got, m.height, view)
	}
	for _, want := range []string{"Team 18", "Team 19", "slot-18"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected clipped view to keep current context %q visible\n%s", want, view)
		}
	}
}

func TestMatchViewScrollsLongContent(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 12
	m.matchView = true
	m.match = &site.MatchPage{
		Title:       "Long match",
		HomeTeam:    "Team 18",
		AwayTeam:    "Team 19",
		Score:       "1-0",
		Competition: "Ekstraklasa",
	}
	for i := 1; i <= 20; i++ {
		m.match.Events = append(m.match.Events, site.MatchEvent{MinuteText: fmt.Sprintf("%d", i), TeamSide: "home", Kind: "SUB", Text: fmt.Sprintf("event-%02d", i)})
	}

	view := m.View()
	if got := strings.Count(view, "\n") + 1; got > m.height {
		t.Fatalf("expected match view to fit terminal height, got %d lines for height %d\n%s", got, m.height, view)
	}
	if !strings.Contains(view, "event-01") {
		t.Fatalf("expected initial match view to show top content\n%s", view)
	}
	for _, want := range []string{"Standings", "Fixtures", "LEG 2-1 LEC"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected match view to keep sidebar content %q visible\n%s", want, view)
		}
	}

	m.matchScroll = 12
	view = m.View()
	if !strings.Contains(view, "event-13") {
		t.Fatalf("expected scrolled match view to show later content\n%s", view)
	}
	if strings.Contains(view, "event-01") {
		t.Fatalf("expected scrolled match view to hide top content\n%s", view)
	}
	for _, want := range []string{"Standings", "Fixtures", "LEG 2-1 LEC"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected scrolled match view to keep sidebar content %q visible\n%s", want, view)
		}
	}
	if got := strings.Count(view, "\n") + 1; got != m.height {
		t.Fatalf("expected match view to fill terminal height, got %d lines for height %d\n%s", got, m.height, view)
	}
}

func TestMatchViewShowsFullStandingsWhenHeightAllows(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 27
	m.matchView = true
	m.match = &site.MatchPage{Title: "Tall match", HomeTeam: "Team 01", AwayTeam: "Team 02", Score: "1-0"}
	m.league.Standings = make([]site.StandingRow, 0, 16)
	for i := 1; i <= 16; i++ {
		m.league.Standings = append(m.league.Standings, site.StandingRow{Position: i, Team: fmt.Sprintf("Team %02d", i), Played: 30, Won: 10, Drawn: 10, Lost: 10, Points: 40 - i})
	}

	view := m.View()
	for _, want := range []string{"Team 01", "Team 16"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected match sidebar to show full standings including %q\n%s", want, view)
		}
	}
}

func TestMatchViewScrollLimitIsClamped(t *testing.T) {
	m := sketchModel()
	m.width = 100
	m.height = 10
	m.matchView = true
	m.match = &site.MatchPage{Title: "Compact match"}
	for i := 1; i <= 20; i++ {
		m.match.Events = append(m.match.Events, site.MatchEvent{MinuteText: fmt.Sprintf("%d", i), TeamSide: "home", Kind: "SUB", Text: fmt.Sprintf("event-%02d", i)})
	}

	limit := m.matchScrollLimit()
	if limit <= 0 {
		t.Fatalf("expected positive scroll limit for tall match content")
	}
}

func sketchModel() Model {
	return Model{
		width:  120,
		height: 40,
		focus:  focusFixtures,
		seasons: []site.Season{{
			Label:    "2024/2025",
			URL:      "http://www.90minut.pl/archsezon.php?id_sezon=101",
			SeasonID: "101",
			Current:  true,
		}},
		competitions: []site.Competition{{
			Name:      "Ekstraklasa",
			URL:       "http://www.90minut.pl/liga/1/liga11233.html",
			LeagueKey: "liga11233",
		}},
		league: &site.LeaguePage{
			Title: "Ekstraklasa",
			Standings: []site.StandingRow{
				{Position: 1, Team: "Legia Warszawa", Played: 24, Won: 16, Drawn: 5, Lost: 3, Points: 53},
				{Position: 2, Team: "Lech Poznan", Played: 24, Won: 15, Drawn: 4, Lost: 5, Points: 49},
			},
			Rounds: []site.Round{{
				Name: "1. kolejka",
				Fixtures: []site.Fixture{
					{Home: "Legia Warszawa", Away: "Lech Poznan", Score: "2-1", WhenInfo: "24 stycznia, 20:30 (16 580)", MatchID: "1", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1"},
					{Home: "Rakow Czestochowa", Away: "Pogon Szczecin", Score: "1-1", MatchID: "2", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2"},
				},
			}},
		},
	}
}
