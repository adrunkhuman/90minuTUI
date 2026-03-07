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
