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
	<tr bgcolor="#f4f4f4"><td><a href="/wystepy.php">Jan Kowalski</a></td><td></td><td><a href="/wystepy.php?id=2">Piotr Nowak</a></td></tr>
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
	if page.HomeLineup[0].Name != "Jan Kowalski" || page.HomeLineup[0].PlayerID != "" {
		t.Fatalf("expected unlinked home player name with empty id, got %+v", page.HomeLineup[0])
	}
	if page.AwayLineup[0].Name != "Piotr Nowak" || page.AwayLineup[0].PlayerID != "2" {
		t.Fatalf("unexpected linked away player identity: %+v", page.AwayLineup[0])
	}
	if page.Referee != "" || page.RefereeID != "" {
		t.Fatalf("expected empty referee identity, got name=%q id=%q", page.Referee, page.RefereeID)
	}
}

func TestPlayerTimelineEventsKeepsOutgoingAndIncomingSubstitutionPlayers(t *testing.T) {
	player := PlayerLine{
		Name:   "Oskar Lesniak",
		Events: []string{"66' -> Damian Nowak"},
	}

	events := playerTimelineEvents(player, TeamSideHome)
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].Kind != EventKindSubstitution {
		t.Fatalf("expected substitution event, got %#v", events[0])
	}
	if events[0].Text != "Oskar Lesniak -> Damian Nowak" {
		t.Fatalf("unexpected substitution text: %q", events[0].Text)
	}
	if events[0].SubstitutionOut != "Oskar Lesniak" || events[0].SubstitutionIn != "Damian Nowak" {
		t.Fatalf("unexpected structured substitution players: %#v", events[0])
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

func TestParseLeaguePagePreservesChampionsLeagueStageOrder(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_14077.html")

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga14077.html")
	if page == nil {
		t.Fatalf("expected league page")
	}

	want := []struct {
		name  string
		phase RoundPhase
	}{
		{"I runda eliminacyjna - 8-9 lipca, 15-16 lipca", RoundPhaseQualification},
		{"II runda eliminacyjna - 22-23 lipca, 29-30 lipca", RoundPhaseQualification},
		{"III runda eliminacyjna - 5-6 sierpnia, 12-13 sierpnia", RoundPhaseQualification},
		{"IV runda eliminacyjna - 19-20 sierpnia, 26-27 sierpnia", RoundPhaseQualification},
		{"Kolejka 1 - 16-18 września", RoundPhaseGroup},
	}
	if len(page.Rounds) < len(want) {
		t.Fatalf("expected at least %d rounds, got %d", len(want), len(page.Rounds))
	}
	for i, expected := range want {
		if page.Rounds[i].Name != expected.name || page.Rounds[i].Phase != expected.phase {
			t.Fatalf("round %d got %q/%q want %q/%q", i, page.Rounds[i].Name, page.Rounds[i].Phase, expected.name, expected.phase)
		}
	}
	if page.Rounds[4].Section != "Grupa LM" {
		t.Fatalf("expected league-phase section, got %q", page.Rounds[4].Section)
	}
}

func TestParseLeaguePageKeepsRepeatedGroupMatchdaysSeparate(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_12909.html")

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga12909.html")
	if page == nil {
		t.Fatalf("expected league page")
	}

	groupRounds := make([]Round, 0, 12)
	for _, round := range page.Rounds {
		if round.Section == "Grupa A" || round.Section == "Grupa B" {
			groupRounds = append(groupRounds, round)
		}
	}
	if len(groupRounds) < 12 {
		t.Fatalf("expected at least 12 group rounds, got %d", len(groupRounds))
	}

	want := []struct{ section, name string }{
		{"Grupa A", "Kolejka 1 - 19-20 września"},
		{"Grupa A", "Kolejka 2 - 3-4 października"},
		{"Grupa A", "Kolejka 3 - 24-25 października"},
		{"Grupa A", "Kolejka 4 - 7-8 listopada"},
		{"Grupa A", "Kolejka 5 - 28-29 listopada"},
		{"Grupa A", "Kolejka 6 - 12-13 grudnia"},
		{"Grupa B", "Kolejka 1 - 19-20 września"},
	}
	for i, expected := range want {
		if groupRounds[i].Section != expected.section || groupRounds[i].Name != expected.name {
			t.Fatalf("group round %d got %q/%q want %q/%q", i, groupRounds[i].Section, groupRounds[i].Name, expected.section, expected.name)
		}
		if groupRounds[i].Phase != RoundPhaseGroup {
			t.Fatalf("expected group phase for %q/%q, got %q", groupRounds[i].Section, groupRounds[i].Name, groupRounds[i].Phase)
		}
	}
	for _, round := range page.Rounds {
		if !strings.Contains(round.Name, "finału") {
			continue
		}
		if round.Section != "" {
			t.Fatalf("knockout round inherited group section: %+v", round)
		}
		return
	}
	t.Fatalf("expected knockout round after group phase")
}

func TestParseLeaguePageHandlesTournamentGroupsWithoutHeadingDates(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_13459.html")

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga13459.html")
	if page == nil {
		t.Fatalf("expected league page")
	}

	var firstGroupRound *Round
	for i := range page.Rounds {
		if page.Rounds[i].Section == "Grupa A" && page.Rounds[i].Name == "Kolejka 1" {
			firstGroupRound = &page.Rounds[i]
			break
		}
	}
	if firstGroupRound == nil {
		t.Fatalf("expected Grupa A / Kolejka 1 round")
	}
	if firstGroupRound.Phase != RoundPhaseGroup {
		t.Fatalf("expected group phase, got %q", firstGroupRound.Phase)
	}
	if len(firstGroupRound.Fixtures) == 0 || !strings.Contains(firstGroupRound.Fixtures[0].WhenInfo, "14 czerwca") {
		t.Fatalf("expected fixture row date in WhenInfo, got %+v", firstGroupRound.Fixtures)
	}
}

func TestParseLeaguePageKeepsSectionOnlyFixtureGroup(t *testing.T) {
	html := `
	<html><head><title>Section Only</title></head><body>
	<table><tr><td><u>Grupa finałowa</u></td></tr></table>
	<table>
	<tr><td>Team A</td><td><a href="/mecz.php?id_mecz=1">1-0</a></td><td>Team B</td><td>1 maja, 18:00</td></tr>
	</table>
	<table><tr><td><u>1/2 finału</u></td></tr></table>
	<table>
	<tr><td>Team C</td><td><a href="/mecz.php?id_mecz=2">2-0</a></td><td>Team D</td><td>8 maja, 18:00</td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/section-only.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Rounds) != 2 {
		t.Fatalf("expected 2 rounds, got %d", len(page.Rounds))
	}
	if page.Rounds[0].Section != "Grupa finałowa" || page.Rounds[0].Name != "" || page.Rounds[0].Phase != RoundPhaseGroup {
		t.Fatalf("unexpected section-only round: %+v", page.Rounds[0])
	}
	if page.Rounds[1].Section != "" || page.Rounds[1].Name != "1/2 finału" || page.Rounds[1].Phase != RoundPhaseKnockout {
		t.Fatalf("knockout round inherited section context: %+v", page.Rounds[1])
	}
}

func TestParseLeaguePagePreservesSectionedQualifierSourceOrder(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_14042.html")

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga14042.html")
	if page == nil {
		t.Fatalf("expected league page")
	}

	indexes := map[string]int{}
	for i, round := range page.Rounds {
		key := round.Section + "/" + round.Name
		if _, exists := indexes[key]; !exists {
			indexes[key] = i
		}
	}
	groupAEnd, ok := indexes["Grupa A/Kolejka 10 - 16-18 listopada"]
	if !ok {
		t.Fatalf("expected Grupa A final matchday")
	}
	groupBStart, ok := indexes["Grupa B/Kolejka 5 - 4-6 września"]
	if !ok {
		t.Fatalf("expected Grupa B first matchday")
	}
	if groupAEnd >= groupBStart {
		t.Fatalf("expected source order to finish Grupa A before Grupa B")
	}
}

func TestParseLeaguePageDoesNotTreatUnderlinedFixtureTeamsAsSections(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_10529.html")

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga10529.html")
	if page == nil {
		t.Fatalf("expected league page")
	}

	foundRound := false
	foundNextRound := false
	for _, round := range page.Rounds {
		if round.Name != "1/16 finału - 26-27 stycznia" {
			if round.Name == "1/8 finału - 13-14 lutego" {
				foundNextRound = true
				if round.Section != "" {
					t.Fatalf("fixture team section leaked into next round: %+v", round)
				}
			}
			continue
		}
		foundRound = true
		if round.Section != "" {
			t.Fatalf("underlined team name should not become a section: %q", round.Section)
		}
	}
	if !foundRound {
		t.Fatalf("expected 1/16 final round")
	}
	if !foundNextRound {
		t.Fatalf("expected 1/8 final round")
	}
}

func TestParseLeaguePageKeepsPlainLeagueNumericOrder(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_8694.html")

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/0/liga8694.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Rounds) < 37 {
		t.Fatalf("expected at least 37 rounds, got %d", len(page.Rounds))
	}
	if page.Rounds[0].Name != "Kolejka 1 - 16-17 lipca" || page.Rounds[36].Name != "Kolejka 37 - 2-4 czerwca" {
		t.Fatalf("unexpected plain league order: first=%q last=%q", page.Rounds[0].Name, page.Rounds[36].Name)
	}
	for _, round := range page.Rounds[:37] {
		if round.Phase != RoundPhaseRegular || round.Section != "" {
			t.Fatalf("expected plain league round, got %+v", round)
		}
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

func TestParseLeaguePageHandlesSavedAmbiguousLinklessFixture(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_ambiguous_linkless.html")

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga99998.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(page.Rounds))
	}
	if len(page.Rounds[0].Fixtures) != 2 {
		t.Fatalf("expected 2 fixtures, got %d", len(page.Rounds[0].Fixtures))
	}

	first := page.Rounds[0].Fixtures[0]
	if first.Home != "Team A" || first.Away != "Team B" || first.Score != "1-0" {
		t.Fatalf("unexpected first fixture: %+v", first)
	}
	if first.WhenInfo != "walkower 3-0 24 lipca, 18:00" {
		t.Fatalf("unexpected first fixture metadata: %+v", first)
	}
	if first.MatchURL != "" || first.MatchID != "" {
		t.Fatalf("expected linkless fixture, got %+v", first)
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

func TestParseLeaguePageNormalizesFixturesBySeasonAwareDate(t *testing.T) {
	html := `<html><head><title>Test liga 2025/26</title></head><body>
<table class="main"><tr><td class="main"><b>1. kolejka</b></td></tr></table>
<table class="main">
<tr><td class="main">Team C</td><td class="main">-</td><td class="main">Team D</td><td class="main">15 marca, 12:00</td></tr>
<tr><td class="main">Team A</td><td class="main">-</td><td class="main">Team B</td><td class="main">20 listopada, 18:00</td></tr>
</table></body></html>`
	doc, err := decodeAndParse([]byte(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga-test.html")
	if page == nil || len(page.Rounds) != 1 || len(page.Rounds[0].Fixtures) != 2 {
		t.Fatalf("expected one round with two fixtures, got %+v", page)
	}

	first := page.Rounds[0].Fixtures[0]
	if first.Home != "Team A" || first.WhenInfo != "20 listopada, 18:00" {
		t.Fatalf("expected November 2025 fixture before March 2026 fixture, got %+v", page.Rounds[0].Fixtures)
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
