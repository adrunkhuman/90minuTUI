package site

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseLeaguePageWithoutWidthSelectors(t *testing.T) {
	html := `
	<html><head><title>Test League</title></head><body>
	<table><tr><td><u>1. kolejka</u></td></tr></table>
	<table>
	<tr>
		<td>2026-03-01</td>
		<td>Gornik Leczna</td>
		<td><a href="/mecz.php?id_mecz=123">2-1</a></td>
		<td>Zaglebie Sosnowiec</td>
		<td>18:00</td>
	</tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga-test.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(page.Rounds))
	}
	if page.LeagueKey != "www.90minut.pl/liga/1/liga-test.html" {
		t.Fatalf("unexpected league key: %q", page.LeagueKey)
	}
	if page.Rounds[0].Name != "1. kolejka" {
		t.Fatalf("unexpected round name: %q", page.Rounds[0].Name)
	}
	if len(page.Rounds[0].Fixtures) != 1 {
		t.Fatalf("expected 1 fixture, got %d", len(page.Rounds[0].Fixtures))
	}

	fixture := page.Rounds[0].Fixtures[0]
	if fixture.Home != "Gornik Leczna" || fixture.Away != "Zaglebie Sosnowiec" {
		t.Fatalf("unexpected fixture sides: %#v", fixture)
	}
	if fixture.MatchURL != "/mecz.php?id_mecz=123" {
		t.Fatalf("unexpected match URL: %q", fixture.MatchURL)
	}
	if fixture.MatchID != "123" {
		t.Fatalf("unexpected match id: %q", fixture.MatchID)
	}
}

func TestParseMatchPageWithout480Width(t *testing.T) {
	html := `
	<html><head><title>Match Test</title></head><body>
	<table class="main" width="620">
	<tr><td colspan="3"><b>I liga</b></td></tr>
	<tr><td colspan="3">1 marca 2026, 18:00</td></tr>
	<tr><td>GKS Tychy</td><td>2-1</td><td>Odra Opole</td></tr>
	<tr><td>(12) Jan Kowalski</td><td>-</td><td></td></tr>
	<tr bgcolor="#f4f4f4"><td><a href="/wystepy.php?id=1">Jan Kowalski</a></td><td></td><td><a href="/wystepy.php?id=2">Piotr Nowak</a></td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=555")
	if page == nil {
		t.Fatalf("expected match page")
	}
	if page.Score != "2-1" {
		t.Fatalf("unexpected score: %q", page.Score)
	}
	if page.MatchID != "555" {
		t.Fatalf("unexpected match id: %q", page.MatchID)
	}
	if page.HomeTeam != "GKS Tychy" || page.AwayTeam != "Odra Opole" {
		t.Fatalf("unexpected team names: %q vs %q", page.HomeTeam, page.AwayTeam)
	}
	if len(page.HomeLineup) != 1 || len(page.AwayLineup) != 1 {
		t.Fatalf("expected lineup extraction, home=%d away=%d", len(page.HomeLineup), len(page.AwayLineup))
	}
}

func TestPlayerTimelineEventsKeepsOutgoingAndIncomingSubstitutionPlayers(t *testing.T) {
	player := PlayerLine{
		Name:   "Oskar Lesniak",
		Events: []string{"66' -> Damian Nowak"},
	}

	events := playerTimelineEvents(player, "home")
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].Kind != "SUB" {
		t.Fatalf("expected substitution event, got %#v", events[0])
	}
	if events[0].Text != "Oskar Lesniak -> Damian Nowak" {
		t.Fatalf("unexpected substitution text: %q", events[0].Text)
	}
	if events[0].MinuteText != "66" {
		t.Fatalf("unexpected substitution minute: %q", events[0].MinuteText)
	}
}

func TestParseMatchPageSkipsBreadcrumbLikeMeta(t *testing.T) {
	html := `
	<html><head><title>Match Test</title></head><body>
	<table class="main" width="620">
	<tr><td colspan="3"><b>I liga - Kolejka 1</b></td></tr>
	<tr><td colspan="3">Strona główna</td></tr>
	<tr><td colspan="3">1 marca 2026, 18:00 1234 Jan Kowalski</td></tr>
	<tr><td>GKS Tychy</td><td>2-1</td><td>Odra Opole</td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=555")
	if page == nil {
		t.Fatalf("expected match page")
	}
	if page.Meta != "1 marca 2026, 18:00 1234 Jan Kowalski" {
		t.Fatalf("unexpected meta: %q", page.Meta)
	}
}

func TestParseLeaguePageSeparatesKnockoutStageFromLeagueRound(t *testing.T) {
	html := `
	<html><head><title>Europe</title></head><body>
	<table><tr><td><u>8. kolejka - 28 stycznia</u></td></tr></table>
	<table>
	<tr><td>Team A</td><td><a href="/mecz.php?id_mecz=1">1-0</a></td><td>Team B</td><td>28 stycznia, 21:00 (5000)</td></tr>
	</table>
	<table><tr><td><u>1/8 finału</u></td></tr></table>
	<table>
	<tr><td>Team C</td><td><a href="/mecz.php?id_mecz=2">2-1</a></td><td>Team D</td><td>17 lutego, 21:00 (7000)</td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/europe.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Rounds) != 2 {
		t.Fatalf("expected 2 rounds, got %d", len(page.Rounds))
	}
	if page.Rounds[0].Name != "8. kolejka - 28 stycznia" {
		t.Fatalf("unexpected first round: %q", page.Rounds[0].Name)
	}
	if page.Rounds[1].Name != "1/8 finału" {
		t.Fatalf("unexpected second round: %q", page.Rounds[1].Name)
	}
}

func TestParseFixturesTableSkipsRowsWithMultipleMatchLinks(t *testing.T) {
	html := `
	<table>
	<tr>
		<td>Team A</td>
		<td><a href="/mecz.php?id_mecz=1">1-0</a></td>
		<td>Team B</td>
		<td><a href="/mecz.php?id_mecz=2">2-2</a></td>
	</tr>
	</table>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	fixtures := parseFixturesTable(doc.Find("table").First())
	if len(fixtures) != 0 {
		t.Fatalf("expected matrix-like row to be skipped, got %d fixtures", len(fixtures))
	}
}

func TestParseFixturesTableFallsBackToPlainTextScoresWithoutMatchLinks(t *testing.T) {
	html := `
	<table>
	<tr><td>Slask Wroclaw</td><td>-</td><td>Pogon Tczew</td><td>16 maja, 12:00</td></tr>
	<tr><td colspan="4">w pierwotnym terminie odwolany</td></tr>
	<tr><td><b>Gornik Leczna</b></td><td><b>3-0</b></td><td><b>UKS SMS Lodz</b></td><td>25 marca, 16:00</td></tr>
	</table>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	fixtures := parseFixturesTable(doc.Find("table").First())
	if len(fixtures) != 2 {
		t.Fatalf("expected 2 fixtures, got %d", len(fixtures))
	}
	if fixtures[0].Home != "Slask Wroclaw" || fixtures[0].Away != "Pogon Tczew" || fixtures[0].Score != "-" || fixtures[0].WhenInfo != "16 maja, 12:00" {
		t.Fatalf("unexpected plain-text fixture: %+v", fixtures[0])
	}
	if fixtures[0].MatchURL != "" || fixtures[0].MatchID != "" {
		t.Fatalf("expected empty match details for plain-text score fixture: %+v", fixtures[0])
	}
	if fixtures[1].Home != "Gornik Leczna" || fixtures[1].Away != "UKS SMS Lodz" || fixtures[1].Score != "3-0" {
		t.Fatalf("unexpected second plain-text fixture: %+v", fixtures[1])
	}
}

func TestRoundNameFromTableSkipsNavigationBlocks(t *testing.T) {
	html := `
	<table>
	<tr>
		<td>
			.: <a href="/liga/1/liga14072.html" class="main">Wyniki</a> |
			<a href="/strzelcy.php?id=14072" class="main">Strzelcy</a> |
			<a href="/stats.php?id=14072" class="main">Statystyki</a> |
			<a href="/liga/1/liga14072.html#last" class="main">Ostatnia kolejka</a> :.
		</td>
	</tr>
	</table>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	if name, ok := roundNameFromTable(doc.Find("table").First()); ok {
		t.Fatalf("expected nav block to be ignored, got %q", name)
	}
}

func TestParseLeaguePageNormalizesRoundsByRoundNumber(t *testing.T) {
	html := `
	<html><head><title>Test League</title></head><body>
	<table><tr><td><u>3. kolejka</u></td></tr></table>
	<table>
	<tr><td>Team E</td><td><a href="/mecz.php?id_mecz=3">1-0</a></td><td>Team F</td><td>3 sierpnia, 18:00</td></tr>
	</table>
	<table><tr><td><u>1. kolejka</u></td></tr></table>
	<table>
	<tr><td>Team A</td><td><a href="/mecz.php?id_mecz=1">1-0</a></td><td>Team B</td><td>20 lipca, 18:00</td></tr>
	</table>
	<table><tr><td><u>2. kolejka</u></td></tr></table>
	<table>
	<tr><td>Team C</td><td><a href="/mecz.php?id_mecz=2">2-1</a></td><td>Team D</td><td>27 lipca, 18:00</td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga-test.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Rounds) != 3 {
		t.Fatalf("expected 3 rounds, got %d", len(page.Rounds))
	}

	got := []string{page.Rounds[0].Name, page.Rounds[1].Name, page.Rounds[2].Name}
	want := []string{"1. kolejka", "2. kolejka", "3. kolejka"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected round order: got %#v want %#v", got, want)
		}
	}
}

func TestParseLeaguePageNormalizesFixturesByDate(t *testing.T) {
	html := `
	<html><head><title>Test League</title></head><body>
	<table><tr><td><u>1. kolejka</u></td></tr></table>
	<table>
	<tr><td>Team C</td><td><a href="/mecz.php?id_mecz=3">1-0</a></td><td>Team D</td><td>24 lipca, 20:30</td></tr>
	<tr><td>Team A</td><td><a href="/mecz.php?id_mecz=1">2-1</a></td><td>Team B</td><td>20 lipca, 18:00</td></tr>
	<tr><td>Team E</td><td><a href="/mecz.php?id_mecz=5">0-0</a></td><td>Team F</td><td>odwołany</td></tr>
	<tr><td>Team G</td><td><a href="/mecz.php?id_mecz=7">3-2</a></td><td>Team H</td><td>24 lipca, 18:00</td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga-test.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(page.Rounds))
	}

	fixtures := page.Rounds[0].Fixtures
	if len(fixtures) != 4 {
		t.Fatalf("expected 4 fixtures, got %d", len(fixtures))
	}

	got := []string{fixtures[0].MatchID, fixtures[1].MatchID, fixtures[2].MatchID, fixtures[3].MatchID}
	want := []string{"1", "7", "3", "5"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected fixture order: got %#v want %#v", got, want)
		}
	}
}

func TestValidateMatchPageAllowsPartialWhenTeamsPresent(t *testing.T) {
	page := &MatchPage{
		Title:    "Match",
		HomeTeam: "GKS Tychy",
		AwayTeam: "Odra Opole",
	}

	if err := validateMatchPage(page); err != nil {
		t.Fatalf("expected partial page to be allowed, got %v", err)
	}
}
