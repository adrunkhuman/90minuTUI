package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// testEvent creates a site.MatchEvent with minute fields pre-populated so tests
// bypass the site parser boundary while still exercising minute-aware rendering.
func testEvent(minuteText, kind, side, text string) site.MatchEvent {
	m, s, ok := site.ParseMinute(minuteText)
	return site.MatchEvent{
		MinuteText: minuteText,
		Minute:     m,
		Stoppage:   s,
		HasMinute:  ok,
		Kind:       kind,
		TeamSide:   side,
		Text:       text,
	}
}

func TestLeagueSketchViewShowsStandingsFixturesAndStatus(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 18
	m.lastFetchAt = time.Date(2026, time.March, 10, 21, 15, 0, 0, time.UTC)
	m.league.Title = "PKO Bank Polski Ekstraklasa 2025/2026"

	view := m.View()
	for _, want := range []string{
		"PKO Bank Polski Ekstraklasa 2025/2026",
		"# Team",
		"Legia Warszawa",
		"Round 1",
		"Lech Poznan",
		"24/01 20:30",
		"21:15:00",
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

func TestLeagueViewUsesTwoLineTopBar(t *testing.T) {
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
		"# Team",
		"Round 1",
		"LEG 2-1 LEC",
		"Loading…",
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
		Events: []site.MatchEvent{
			testEvent("17", "GOAL", "home", "Krzysztof Kubica 17"),
			testEvent("62", "GOAL", "away", "Samuel Mraz 62"),
		},
		NewsTitle: "PKO BP Ekstraklasa: Bruk-Bet Termalica 1-2 Motor",
		NewsURL:   "http://www.90minut.pl/news/example.html",
	}

	view := m.View()
	for _, want := range []string{
		"PKO Bank Polski Ekstraklasa 2025/2026",
		"Bruk-Bet Termalica Nieciecza",
		"1 – 2",
		"Motor Lublin",
		"K. Kubica",
		"S. Mraz",
		"17'",
		"62'",
		"FT 1 – 2",
		"13 March 2026, 18:00",
		"Attendance 3542",
		"Ref. Damian Kos",
		"Weather 15 C",
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

func TestMatchDetailShowsEventsInScoreHeaderAndLineups(t *testing.T) {
	m := sketchModel()
	m.width = 140
	m.matchView = true
	m.match = &site.MatchPage{
		HomeTeam: "GKS Katowice",
		AwayTeam: "Lechia Gdansk",
		Score:    "2-1",
		Events: []site.MatchEvent{
			testEvent("39", "GOAL", "home", "Wdowiak 39"),
			testEvent("46", "SUB", "home", "Igor Strzalek (86) -> Damian Nowak"),
			testEvent("46", "SUB", "away", "O. Lesniak -> Pllana"),
			testEvent("52", "MISS", "away", "Barkowskij 52 (nk)"),
			testEvent("60", "GOAL", "home", "Szkurin 60"),
			testEvent("70", "GOAL", "away", "Karol Czubak (k) 70"),
			testEvent("85", "RC", "away", "Pllana 85"),
		},
		HomeLineup: []site.PlayerLine{
			{Name: "Wdowiak"},
			{Name: "Igor Strzalek (86)"},
			{Name: "Szkurin"},
			{Name: "Damian Nowak"},
		},
		AwayLineup: []site.PlayerLine{
			{Name: "Karol Czubak (k)"},
			{Name: "Barkowskij"},
			{Name: "O. Lesniak"},
			{Name: "Pllana"},
		},
	}

	view := m.View()
	plainView := ansi.Strip(view)
	_, centerWidth, _ := matchLayoutWidths(m.width)
	content := ansi.Strip(m.matchDetailContent(centerWidth))
	scoreHeader := renderMatchDetailRow("GKS Katowice", matchDetailScore("2-1"), "Lechia Gdansk", centerWidth-4)
	spacerRow := renderMatchDetailRow("", "", "", centerWidth-4)
	if !strings.Contains(content, scoreHeader+"\n"+spacerRow+"\n") {
		t.Fatalf("expected spacer row between score header and event log\n%s", content)
	}

	for _, want := range []string{
		"Wdowiak", "39'", "⚽", // home goal row (minute in center, icon adjacent)
		"HT 1 – 0", // HT divider with score
		"❌", "52'", // away missed penalty row
		"Szkurin", "60'", // second home goal row
		"K. Czubak", "70'", // away goal row
		"■", "85'", // away red card row
		"FT 2 – 1", // FT divider with score
	} {
		if !strings.Contains(plainView, want) {
			t.Fatalf("expected match view to contain %q\n%s", want, view)
		}
	}

	for _, unwanted := range []string{
		"Timeline",
		"↕",
	} {
		if strings.Contains(plainView, unwanted) {
			t.Fatalf("expected view to omit %q\n%s", unwanted, view)
		}
	}

	headerIndexes := []int{
		strings.Index(plainView, "39'"),
		strings.Index(plainView, "HT 1 – 0"),
		strings.Index(plainView, "52'"),
		strings.Index(plainView, "60'"),
		strings.Index(plainView, "70'"),
		strings.Index(plainView, "85'"),
		strings.Index(plainView, "FT 2 – 1"),
	}
	for _, idx := range headerIndexes {
		if idx < 0 {
			t.Fatalf("expected ordered score header in view\n%s", plainView)
		}
	}
	for i := 1; i < len(headerIndexes); i++ {
		if headerIndexes[i-1] >= headerIndexes[i] {
			t.Fatalf("expected score header order 39 -> HT -> 52 -> 60 -> 70 -> 85 -> FT\n%s", plainView)
		}
	}

	for _, want := range []string{
		"46'",      // sub minute visible at outer edge of player name
		"D. Nowak", // sub-on player visible immediately after sub-off
		"■",        // red card badge in event column
	} {
		if !strings.Contains(plainView, want) {
			t.Fatalf("expected lineup section to contain %q\n%s", want, view)
		}
	}
	lineupIdx := strings.Index(plainView, "Lineups")
	if lineupIdx >= 0 {
		lineupSection := plainView[lineupIdx:]
		if strings.Contains(lineupSection, "⚽") {
			t.Fatalf("expected lineup section to omit goal annotations\n%s", lineupSection)
		}
	}
	if strings.Contains(plainView, "Substitutions") {
		t.Fatalf("expected substitution pane to be omitted\n%s", view)
	}
	if !strings.Contains(view, "\x1b[2m(46' D. Nowak)\x1b[0m") {
		t.Fatalf("expected lineup substitution note to be dimmed\n%s", view)
	}
}

func TestFormatLineupPlayerMirrorsSubstitutionNote(t *testing.T) {
	home := formatLineupPlayer(lineupEntry{
		player:     site.PlayerLine{Name: "Igor Strzalek"},
		leftAt:     "46'",
		replacedBy: "Damian Nowak",
	}, "home", 64)
	away := formatLineupPlayer(lineupEntry{
		player:     site.PlayerLine{Name: "J. Wilson-Esbrand"},
		leftAt:     "46'",
		replacedBy: "J. Grzesik",
	}, "away", 64)

	if got := ansi.Strip(home); got != "(46' D. Nowak) I. Strzalek" {
		t.Fatalf("unexpected home lineup substitution label: %q", got)
	}
	if got := ansi.Strip(away); got != "J. Wilson-Esbrand (J. Grzesik 46')" {
		t.Fatalf("unexpected away lineup substitution label: %q", got)
	}
	if !strings.Contains(home, "\x1b[2m(46' D. Nowak)\x1b[0m") {
		t.Fatalf("expected home substitution note to be dimmed, got %q", home)
	}
	if !strings.Contains(away, "\x1b[2m(J. Grzesik 46')\x1b[0m") {
		t.Fatalf("expected away substitution note to be dimmed, got %q", away)
	}
}

func TestFormatLineupPlayerShowsEntryAndExitNotes(t *testing.T) {
	home := formatLineupPlayer(lineupEntry{
		player:     site.PlayerLine{Name: "Oskar Lesniak"},
		enteredAt:  "66'",
		replaced:   "Jason Lokilo",
		leftAt:     "82'",
		replacedBy: "Michal Smith",
	}, "home", 64)
	away := formatLineupPlayer(lineupEntry{
		player:     site.PlayerLine{Name: "Oskar Lesniak"},
		enteredAt:  "66'",
		replaced:   "Jason Lokilo",
		leftAt:     "82'",
		replacedBy: "Michal Smith",
	}, "away", 64)

	if got := ansi.Strip(home); got != "(66' for J. Lokilo) (82' M. Smith) O. Lesniak" {
		t.Fatalf("unexpected home double substitution label: %q", got)
	}
	if got := ansi.Strip(away); got != "O. Lesniak (for J. Lokilo 66') (M. Smith 82')" {
		t.Fatalf("unexpected away double substitution label: %q", got)
	}
}

func TestFormatLineupPlayerShortensOnlySubstitutionNotesWhenNeeded(t *testing.T) {
	got := ansi.Strip(formatLineupPlayer(lineupEntry{
		player:     site.PlayerLine{Name: "Alexandre Verylongsurname"},
		enteredAt:  "66'",
		replaced:   "Christopher Hyperextendedname",
		leftAt:     "82'",
		replacedBy: "Maximilian Unnecessarilylongsurname",
	}, "home", 40))

	if got != "(66' for Hyperextendedname) (82' Unnecessarilylongsurname) A. Verylongsurname" {
		t.Fatalf("unexpected shortened substitution label: %q", got)
	}
}

func TestFormatLineupPlayerShortensNotesForNarrowFallbackRows(t *testing.T) {
	width := lineupPlayerWidth(32)
	if width != 13 {
		t.Fatalf("unexpected narrow lineup player width: %d", width)
	}

	got := ansi.Strip(formatLineupPlayer(lineupEntry{
		player:     site.PlayerLine{Name: "Alexandre Verylongsurname"},
		leftAt:     "82'",
		replacedBy: "Maximilian Unnecessarilylongsurname",
	}, "away", width))

	if got != "A. Verylongsurname (Unnecessarilylongsurname 82')" {
		t.Fatalf("expected narrow-row shortening before truncation, got %q", got)
	}
}

func TestMatchDetailKeepsSubstitutionsOnlyInLineups(t *testing.T) {
	m := sketchModel()
	m.width = 140
	m.matchView = true
	m.match = &site.MatchPage{
		HomeTeam: "Piast Gliwice",
		AwayTeam: "Radomiak Radom",
		Score:    "3-1",
		Events: []site.MatchEvent{
			testEvent("66", "SUB", "home", "Jason Lokilo -> Oskar Lesniak"),
			testEvent("46", "SUB", "away", "J. Wilson-Esbrand -> J. Grzesik"),
		},
		HomeLineup: []site.PlayerLine{{Name: "Jason Lokilo"}},
		AwayLineup: []site.PlayerLine{{Name: "J. Wilson-Esbrand"}},
	}

	view := m.View()
	plainView := ansi.Strip(view)
	if strings.Contains(plainView, "Substitutions") {
		t.Fatalf("expected substitutions pane to be omitted\n%s", view)
	}
	for _, want := range []string{"(66' O. Lesniak) J. Lokilo", "J. Wilson-Esbrand (J. Grzesik 46')"} {
		if !strings.Contains(plainView, want) {
			t.Fatalf("expected lineup to contain %q\n%s", want, view)
		}
	}
}

func TestMatchDetailRowsAnchorTowardCenteredMinuteColumn(t *testing.T) {
	line := renderMatchDetailRow("Wdowiak ⚽", "39'", "Pllana ■", 76)
	minuteIdx := strings.Index(line, "39'")
	leftIdx := strings.Index(line, "Wdowiak ⚽")
	rightIdx := strings.Index(line, "Pllana ■")
	if minuteIdx <= 0 || leftIdx < 0 || rightIdx < 0 {
		t.Fatalf("expected all columns rendered, got %q", line)
	}
	if leftIdx == 0 {
		t.Fatalf("expected left event to anchor toward the minute column, got %q", line)
	}
	if !(leftIdx < minuteIdx && minuteIdx < rightIdx) {
		t.Fatalf("expected centered minute between left and right columns, got %q", line)
	}
}

func TestMatchDetailMinuteColumnStaysFixedForDifferentHomeTextWidths(t *testing.T) {
	short := renderMatchDetailRow("K. Kubica ⚽", "17'", "", 76)
	long := renderMatchDetailRow("B. Wolski (pen) ⚽", "78'", "", 76)
	if strings.Index(short, "17'") != strings.Index(long, "78'") {
		t.Fatalf("expected minute column to stay fixed\nshort: %q\nlong: %q", short, long)
	}
}

func TestFormatMatchMinuteLeftPadsSingleDigitMinute(t *testing.T) {
	if got := formatMatchMinute("9"); got != " 9'" {
		t.Fatalf("unexpected single-digit minute formatting: %q", got)
	}
	if got := formatMatchMinute("17"); got != "17'" {
		t.Fatalf("unexpected double-digit minute formatting: %q", got)
	}
}

func TestMatchDividerSharesCenteredMinuteColumn(t *testing.T) {
	row := renderMatchDetailRow("Wdowiak G", "39'", "S Pllana (4)", 76)
	divider := renderMatchDividerRow("HT 1 - 0", 76)
	// The divider label stays visually centered, and its score dash should remain close
	// to the minute center used by event rows.
	// Divider fill uses '─', so normalise it to '-' before byte-indexing.
	dividerASCII := strings.ReplaceAll(divider, "─", "-")
	rowMid := strings.Index(row, "39'") + 1                // '9' = middle char of "39'"
	dividerMid := strings.Index(dividerASCII, "1 - 0") + 2 // score dash within the divider label
	if diff := rowMid - dividerMid; diff < -2 || diff > 2 {
		t.Fatalf("expected divider score dash to align with minute centre\nrow: %q\ndiv: %q", row, divider)
	}
}

func TestMatchDividerFillsCenterPaddingWithDashes(t *testing.T) {
	divider := renderMatchDividerRow("HT 0 - 0", 76)
	// Padding chars are '─'; the test label stays ASCII so byte positions are stable.
	if strings.Contains(divider, "HT 0 - 0   ") {
		t.Fatalf("expected divider to avoid wide trailing spaces after label, got %q", divider)
	}
	if !strings.Contains(divider, " HT 0 - 0 ") {
		t.Fatalf("expected divider to keep single spaces around label, got %q", divider)
	}
	if !strings.Contains(divider, "─") {
		t.Fatalf("expected divider to use box-drawing fill character, got %q", divider)
	}
}

func TestHeaderEventRowsMinuteInCenterColumn(t *testing.T) {
	rows := headerEventRows([]site.MatchEvent{
		testEvent("17", "GOAL", "home", "Krzysztof Kubica 17"),
		testEvent("30", "GOAL", "away", "Karol Czubak (k) 30"),
	})
	// Two goal rows only (no HT divider — all events in same half)
	if len(rows) != 2 {
		t.Fatalf("expected two header rows, got %d: %#v", len(rows), rows)
	}
	// Label has name + icon only; no minute embedded in label
	if ansi.Strip(rows[0].label) != "K. Kubica ⚽" {
		t.Fatalf("unexpected home scorer label: %q", ansi.Strip(rows[0].label))
	}
	if ansi.Strip(rows[1].label) != "⚽ K. Czubak (pen)" {
		t.Fatalf("unexpected away scorer label: %q", ansi.Strip(rows[1].label))
	}
	// Penalty suffix must still be dimmed
	if !strings.Contains(rows[1].label, "\x1b[2m(pen)\x1b[0m") {
		t.Fatalf("expected scored penalty suffix to be dimmed, got %q", rows[1].label)
	}
	// Center column carries the minute
	if strings.TrimSpace(rows[0].minute) != "17'" {
		t.Fatalf("expected home center to carry minute 17', got %q", rows[0].minute)
	}
	if strings.TrimSpace(rows[1].minute) != "30'" {
		t.Fatalf("expected away center to carry minute 30', got %q", rows[1].minute)
	}
	// Minutes align on the same center axis
	home := renderMatchDetailRow(rows[0].label, rows[0].minute, "", 76)
	away := renderMatchDetailRow("", rows[1].minute, rows[1].label, 76)
	homeMid := strings.Index(home, "17'")
	awayMid := strings.Index(away, "30'")
	if diff := homeMid - awayMid; diff < -1 || diff > 1 {
		t.Fatalf("expected minutes to share centered column\nhome: %q\naway: %q", home, away)
	}
}

func TestHeaderEventRowsIncludesRedCardsAndHTDivider(t *testing.T) {
	rows := headerEventRows([]site.MatchEvent{
		testEvent("39", "GOAL", "home", "Wdowiak 39"),
		testEvent("60", "GOAL", "home", "Szkurin 60"),
		testEvent("85", "RC", "away", "Pllana 85"),
	})
	// Expect: goal (39'), HT divider, goal (60'), red card (85')
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows (goal, HT, goal, RC), got %d: %#v", len(rows), rows)
	}
	if !rows[1].isDivider {
		t.Fatalf("expected row 1 to be HT divider, got %#v", rows[1])
	}
	if rows[1].label != "HT 1 – 0" {
		t.Fatalf("expected HT divider label %q, got %q", "HT 1 – 0", rows[1].label)
	}
	if ansi.Strip(rows[0].label) != "Wdowiak ⚽" {
		t.Fatalf("unexpected first goal label: %q", ansi.Strip(rows[0].label))
	}
	if strings.TrimSpace(rows[0].minute) != "39'" {
		t.Fatalf("expected first goal minute 39' in center, got %q", rows[0].minute)
	}
	if !strings.Contains(ansi.Strip(rows[3].label), "■") {
		t.Fatalf("expected red card row to contain ■, got %q", ansi.Strip(rows[3].label))
	}
	if rows[3].minute != "85'" {
		t.Fatalf("expected red card minute 85', got %q", rows[3].minute)
	}
}

func TestHeaderEventRowsIncludesGoallessHTDividerBeforeSecondHalfEvents(t *testing.T) {
	rows := headerEventRows([]site.MatchEvent{
		testEvent("60", "RC", "away", "Pllana 60"),
	})

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (HT, RC), got %d: %#v", len(rows), rows)
	}
	if !rows[0].isDivider {
		t.Fatalf("expected row 0 to be HT divider, got %#v", rows[0])
	}
	if rows[0].label != "HT 0 – 0" {
		t.Fatalf("expected HT divider label %q, got %q", "HT 0 – 0", rows[0].label)
	}
	if rows[1].minute != "60'" {
		t.Fatalf("expected second-half event minute 60', got %q", rows[1].minute)
	}
}

func TestHeaderEventRowsKeepsHTDividerWhenSecondHalfHasOnlyHiddenEvents(t *testing.T) {
	rows := headerEventRows([]site.MatchEvent{
		testEvent("39", "GOAL", "home", "Wdowiak 39"),
		testEvent("60", "SUB", "home", "Igor Strzalek -> Damian Nowak"),
		testEvent("72", "YC", "away", "Pllana 72"),
	})

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (goal, HT), got %d: %#v", len(rows), rows)
	}
	if rows[0].isDivider {
		t.Fatalf("expected first row to be visible event, got %#v", rows[0])
	}
	if !rows[1].isDivider || rows[1].label != "HT 1 – 0" {
		t.Fatalf("expected final row to be HT divider, got %#v", rows[1])
	}
}

func TestRenderPlayerLineAbbreviatesNameAndDropsEvents(t *testing.T) {
	got := renderPlayerLine(site.PlayerLine{Name: "(86) Igor Strzalek", Events: []string{"YC", "RC"}})
	if got != "I. Strzalek" {
		t.Fatalf("unexpected player line: %q", got)
	}
}

func TestCardAnnotationPrefersRedCardOverEarlierYellow(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		testEvent("20", "YC", "home", "Pllana 20"),
		testEvent("85", "RC", "home", "Pllana 85"),
	}, "home")

	if got := cardAnnotation(site.PlayerLine{Name: "Pllana"}, idx); got != eventPrefix("RC") {
		t.Fatalf("expected red-card badge, got %q", got)
	}
}

func TestAnnotatedLineupMatchesSubstitutionByCompactNameNotSurnameOnly(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		testEvent("60", "SUB", "home", "Jan Kowalski -> Piotr Kowalski"),
	}, "home")

	players := []site.PlayerLine{
		{Name: "Adam Kowalski"},
		{Name: "Jan Kowalski"},
		{Name: "Piotr Kowalski"},
	}

	got := annotatedLineup(players, idx)
	if len(got) != 3 {
		t.Fatalf("expected 3 lineup entries, got %d", len(got))
	}
	if got[0].player.Name != "Adam Kowalski" {
		t.Fatalf("expected unrelated Kowalski to stay first, got %#v", got)
	}
	if got[1].player.Name != "Jan Kowalski" || got[1].replacedBy != "Piotr Kowalski" || got[1].leftAt != "60'" {
		t.Fatalf("expected substituted player to carry entrant note, got %#v", got)
	}
	if got[2].player.Name != "Piotr Kowalski" || got[2].replacedBy != "" {
		t.Fatalf("expected entrant row to stay untouched when already present, got %#v", got)
	}
	if got[2].enteredAt != "60'" || got[2].replaced != "Jan Kowalski" {
		t.Fatalf("expected entrant row to record who they replaced, got %#v", got)
	}
}

func TestAnnotatedLineupDistinguishesSameInitialSameSurname(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		testEvent("60", "SUB", "home", "Jan Kowalski -> Piotr Kowalski"),
	}, "home")

	players := []site.PlayerLine{
		{Name: "Jerzy Kowalski"},
		{Name: "Jan Kowalski"},
		{Name: "Piotr Kowalski"},
	}

	got := annotatedLineup(players, idx)
	if len(got) != 3 {
		t.Fatalf("expected 3 lineup entries, got %d", len(got))
	}
	if got[0].player.Name != "Jerzy Kowalski" {
		t.Fatalf("expected same-initial teammate to stay in place, got %#v", got)
	}
	if got[1].player.Name != "Jan Kowalski" || got[1].replacedBy != "Piotr Kowalski" {
		t.Fatalf("expected substitution to match full name, got %#v", got)
	}
}

func TestAnnotatedLineupAddsMissingEntrantWhenTheyHaveCardEvent(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		testEvent("66", "SUB", "home", "Jason Lokilo -> Oskar Lesniak"),
		testEvent("84", "YC", "home", "Oskar Lesniak 84"),
	}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jason Lokilo"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected starter plus synthetic entrant, got %#v", got)
	}
	if got[0].player.Name != "Jason Lokilo" || got[0].replacedBy != "Oskar Lesniak" || got[0].leftAt != "66'" {
		t.Fatalf("expected starter row to retain substitution note, got %#v", got)
	}
	if got[1].player.Name != "Oskar Lesniak" || got[1].replacedBy != "" {
		t.Fatalf("expected entrant row added for card badge visibility, got %#v", got)
	}
	if got[1].enteredAt != "66'" || got[1].replaced != "Jason Lokilo" {
		t.Fatalf("expected synthetic entrant row to retain entry note, got %#v", got)
	}
}

func TestAnnotatedLineupSkipsMissingEntrantWithoutBadgeEvent(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		testEvent("66", "SUB", "home", "Jason Lokilo -> Oskar Lesniak"),
	}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jason Lokilo"}}, idx)
	if len(got) != 1 {
		t.Fatalf("expected only starter row when entrant has no lineup badge event, got %#v", got)
	}
	if got[0].player.Name != "Jason Lokilo" || got[0].replacedBy != "Oskar Lesniak" || got[0].leftAt != "66'" {
		t.Fatalf("expected starter row to retain substitution note, got %#v", got)
	}
}

func TestAnnotatedLineupSyntheticEntrantKeepsLaterSubstitutionOff(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		testEvent("66", "SUB", "home", "Jason Lokilo -> Oskar Lesniak"),
		testEvent("78", "SUB", "home", "Oskar Lesniak -> Michal Smith"),
		testEvent("72", "YC", "home", "Oskar Lesniak 72"),
	}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jason Lokilo"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected starter plus synthetic entrant, got %#v", got)
	}
	if got[1].player.Name != "Oskar Lesniak" || got[1].enteredAt != "66'" || got[1].replaced != "Jason Lokilo" || got[1].leftAt != "78'" || got[1].replacedBy != "Michal Smith" {
		t.Fatalf("expected synthetic entrant row to keep both substitution notes, got %#v", got[1])
	}
}

func TestRenderLineupRowUsesCenteredSeparatorColumn(t *testing.T) {
	row := renderLineupRow("K. Kubica", "B. Mrozek", 76)
	rowMid := strings.Index(row, "|")
	dividerMid := 76 / 2
	if diff := rowMid - dividerMid; diff < -1 || diff > 1 {
		t.Fatalf("expected lineup separator to share center axis\nrow: %q", row)
	}
	if !strings.Contains(row, "K. Kubica") || !strings.Contains(row, "B. Mrozek") {
		t.Fatalf("expected lineup row to contain both players, got %q", row)
	}
	if strings.Contains(row, "    |    ") {
		t.Fatalf("expected tighter lineup spacing around center separator, got %q", row)
	}
}

func TestRenderLineupHeaderRowUsesBlankCenteredGap(t *testing.T) {
	row := renderLineupRowWithMarker("Piast Gliwice", "Radomiak Radom", " ", 76)

	if !strings.Contains(row, "Piast Gliwice") || !strings.Contains(row, "Radomiak Radom") {
		t.Fatalf("expected lineup header row to contain both team names, got %q", row)
	}
	if strings.Contains(row, "|") {
		t.Fatalf("expected lineup header row to omit separator, got %q", row)
	}

	leftEnd := strings.Index(row, "Piast Gliwice") + len("Piast Gliwice")
	rightStart := strings.Index(row, "Radomiak Radom")
	gapMid := leftEnd + ((rightStart - leftEnd) / 2)
	dividerMid := 76 / 2
	if diff := gapMid - dividerMid; diff < -1 || diff > 1 {
		t.Fatalf("expected lineup header gap to stay centered\nrow: %q", row)
	}
}

func TestMatchDetailContentStylesLineupTeamsWithoutSeparator(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	m := sketchModel()
	m.match = &site.MatchPage{
		HomeTeam:   "Piast Gliwice",
		AwayTeam:   "Radomiak Radom",
		HomeLineup: []site.PlayerLine{{Name: "K. Szymanski"}},
		AwayLineup: []site.PlayerLine{{Name: "F. Majchrowicz"}},
	}

	content := m.matchDetailContent(80)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !strings.Contains(ansi.Strip(line), "Lineups") || i+1 >= len(lines) {
			continue
		}

		header := lines[i+1]
		stripped := ansi.Strip(header)
		if !strings.Contains(stripped, "Piast Gliwice") || !strings.Contains(stripped, "Radomiak Radom") {
			t.Fatalf("expected lineup team header row after section title, got %q", stripped)
		}
		if strings.Contains(stripped, "|") {
			t.Fatalf("expected lineup team header to omit separator, got %q", stripped)
		}
		if !strings.Contains(header, "\x1b[") {
			t.Fatalf("expected lineup team header to include styling, got %q", header)
		}
		return
	}

	t.Fatal("expected match detail content to include lineup team header row")
}

func TestMatchDetailContentShowsFTDividerWithoutVisibleHeaderEvents(t *testing.T) {
	m := sketchModel()
	m.match = &site.MatchPage{
		HomeTeam: "Motor Lublin",
		AwayTeam: "Zaglebie Lubin",
		Score:    "1-0",
		Events: []site.MatchEvent{
			testEvent("46", "SUB", "home", "Jan Kowalski -> Piotr Kowalski"),
			testEvent("72", "YC", "away", "Nowak 72"),
		},
	}

	content := ansi.Strip(m.matchDetailContent(80))
	if !strings.Contains(content, "FT 1 – 0") {
		t.Fatalf("expected FT divider even without visible header events\n%s", content)
	}
}

func TestRenderCenteredTextCentersSectionLabels(t *testing.T) {
	centered := renderCenteredText("Timeline", 21)
	leftPad := strings.Index(centered, "Timeline")
	rightPad := len(centered) - leftPad - len("Timeline")
	if leftPad == 0 || rightPad == 0 {
		t.Fatalf("expected centered padding, got %q", centered)
	}
	if leftPad-rightPad > 1 || rightPad-leftPad > 1 {
		t.Fatalf("expected roughly symmetric padding, got %q", centered)
	}
}

func TestFinalScoreLineUsesMatchScore(t *testing.T) {
	got := finalScoreLine(&site.MatchPage{Score: "2-0"})
	if got != "FT 2 – 0" {
		t.Fatalf("unexpected final score line: %q", got)
	}
}

func TestLayoutWidthsFavorWiderLeftPane(t *testing.T) {
	leagueLeft, leagueRight := leagueLayoutWidths(120)
	if leagueLeft < 42 || leagueLeft >= leagueRight {
		t.Fatalf("expected league layout to reserve a moderate left pane and wider fixtures pane, got left=%d right=%d", leagueLeft, leagueRight)
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
		MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1",
	}}, 0, 5, 80, false)

	if len(lines) != 1 || !strings.Contains(ansi.Strip(lines[0]), "Legia Warszawa") || !strings.Contains(ansi.Strip(lines[0]), "Lech Poznan") || !strings.Contains(ansi.Strip(lines[0]), "24/01 20:30") {
		t.Fatalf("expected full fixture line, got %v", lines)
	}
}

func TestRenderFixtureWindowUsesCompactNamesInMatchSidebar(t *testing.T) {
	lines := renderFixtureWindow([]site.Fixture{{
		Home:     "Legia Warszawa",
		Away:     "Lech Poznan",
		Score:    "2-1",
		WhenInfo: "24 stycznia, 20:30 (16 580)",
		MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1",
	}}, 0, 5, 40, true)

	// Compact fixture: abbreviated names + score, then when-info separated by two spaces (no pipe).
	stripped := ansi.Strip(lines[0])
	if len(lines) != 1 || !strings.Contains(stripped, "LEG 2-1 LEC") || !strings.Contains(stripped, "24/01 20:30") {
		t.Fatalf("expected compact fixture line, got %v", lines)
	}
}

func TestRenderFixtureWindowMarksNonDrillableFixturesWhenSpaceAllows(t *testing.T) {
	lines := renderFixtureWindow([]site.Fixture{{
		Home:     "Legia Warszawa",
		Away:     "Lech Poznan",
		Score:    "-",
		WhenInfo: "24 stycznia, 20:30",
	}}, 0, 5, 120, false)

	if len(lines) != 1 || !strings.Contains(lines[0], "[no details]") {
		t.Fatalf("expected non-drillable marker, got %v", lines)
	}
}

func TestStatusBarViewReflectsFixtureDrillability(t *testing.T) {
	m := sketchModel()
	m.width = 120

	status := m.statusBarView()
	if !strings.Contains(status, "enter: details") {
		t.Fatalf("expected drillable status hint, got %q", status)
	}

	m.league.Rounds[0].Fixtures[0].MatchURL = ""
	m.league.Rounds[0].Fixtures[0].MatchID = ""
	status = m.statusBarView()
	if !strings.Contains(status, "enter: unavail") {
		t.Fatalf("expected non-drillable status hint, got %q", status)
	}
}

func TestRenderFixtureWindowAlignsFullFixtureColumns(t *testing.T) {
	fixtures := []site.Fixture{
		{Home: "Bruk-Bet Termalica Nieciecza", Away: "Motor Lublin", Score: "1-2", WhenInfo: "13 marca, 18:00 (3542)", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1"},
		{Home: "Jagiellonia Bialystok", Away: "Piast Gliwice", Score: "1-2", WhenInfo: "14 marca, 14:45 (16 580)", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2"},
	}
	lines := renderFixtureWindow(fixtures, 0, 5, 84, false)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	// Normalise: strip ANSI then replace the multi-byte cursor marker '›' with a space
	// so both lines have the same 2-byte prefix and byte positions match visual positions.
	norm := func(s string) string { return strings.Replace(ansi.Strip(s), "›", " ", 1) }
	n0, n1 := norm(lines[0]), norm(lines[1])
	if strings.Index(n0[2:], "13/03") != strings.Index(n1[2:], "14/03") {
		t.Fatalf("expected aligned date column, got %q and %q", lines[0], lines[1])
	}
	if strings.Index(n0[2:], "1-2") != strings.Index(n1[2:], "1-2") {
		t.Fatalf("expected score column alignment, got %q and %q", lines[0], lines[1])
	}
}

func TestRenderFixtureWindowKeepsColumnsAlignedWhenDetailsUnavailable(t *testing.T) {
	fixtures := []site.Fixture{
		{Home: "Cracovia", Away: "Arka Gdynia", Score: "-", WhenInfo: "12 kwietnia, 12:15"},
		{Home: "Legia Warszawa", Away: "Gornik Zabrze", Score: "1-1", WhenInfo: "11 kwietnia, 20:15", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=3"},
	}
	lines := renderFixtureWindow(fixtures, 0, 5, 84, false)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	norm := func(s string) string { return strings.Replace(ansi.Strip(s), "›", " ", 1) }
	n0, n1 := norm(lines[0]), norm(lines[1])
	if strings.Index(n0[2:], "Arka Gdynia") != strings.Index(n1[2:], "Gornik Zabrze") {
		t.Fatalf("expected aligned away-team column, got %q and %q", lines[0], lines[1])
	}
	if strings.Index(n0[2:], "12/04") != strings.Index(n1[2:], "11/04") {
		t.Fatalf("expected aligned date column, got %q and %q", lines[0], lines[1])
	}
	if !strings.Contains(lines[0], "[no details]") {
		t.Fatalf("expected unavailable-details marker, got %q", lines[0])
	}
}

func TestStandingsTeamWidthUsesAvailableSpaceWithoutOverexpanding(t *testing.T) {
	rows := []site.StandingRow{
		{Team: "Legia Warszawa"},
		{Team: "Bruk-Bet Termalica Nieciecza"},
	}

	if got := standingsTeamWidth(rows, 60); got != ansi.StringWidth("Bruk-Bet Termalica Nieciecza") {
		t.Fatalf("expected long team to fit when space allows, got %d", got)
	}
	if got := standingsTeamWidth(rows, 80); got != ansi.StringWidth("Bruk-Bet Termalica Nieciecza") {
		t.Fatalf("expected width to stop at longest team, got %d", got)
	}
	if got := standingsTeamWidth(rows, 40); got != 19 {
		t.Fatalf("expected width capped by available space, got %d", got)
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
		"# Team",
		"Legia Warszawa",
		"Lech Poznan",
		"24/01 20:30",
		"Season",
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

func TestSelectorPopupWidthExpandsForVisibleLeagueNames(t *testing.T) {
	seasonLines := renderSeasonsWindow(sketchModel().seasons, 0)
	competitionLines := []string{
		"Decathlon IV liga 2025/2026, grupa: mazowiecka",
		"Keeza Liga okregowa 2025/2026, grupa: Ciechanow-Ostroleka",
	}

	got := selectorPopupWidth(120, seasonLines, "Ligi regionalne 2025/26 - Mazowiecki ZPN", competitionLines)
	if got <= 68 {
		t.Fatalf("expected popup wider than legacy cap for long visible league names, got %d", got)
	}

	short := selectorPopupWidth(120, seasonLines, "Leagues", []string{"Ekstraklasa", "I liga"})
	if short >= got {
		t.Fatalf("expected short content popup narrower than long-content popup, got short=%d long=%d", short, got)
	}
}

func TestSelectorPopupWidthDoesNotDependOnCurrentScrollWindow(t *testing.T) {
	m := sketchModel()
	m.competitions = make([]site.Competition, 0, 24)
	for i := 0; i < 24; i++ {
		name := fmt.Sprintf("League %02d", i)
		if i == 22 {
			name = "Ligi regionalne 2025/26 - Mazowiecki ZPN, grupa: Ciechanow-Ostroleka i okolice"
		}
		m.competitions = append(m.competitions, site.Competition{Name: name})
	}

	seasonLines := renderSeasonsWindow(m.seasons, m.seasonCursor)
	rightHeading := "Leagues"
	allLines := selectorCompetitionWidthLines(m.competitions)
	widthFromAll := selectorPopupWidth(120, seasonLines, rightHeading, allLines)

	visibleNearTop := renderCompetitionWindow(m.competitions, 0)
	visibleNearBottom := renderCompetitionWindow(m.competitions, len(m.competitions)-1)
	widthTop := selectorPopupWidth(120, seasonLines, rightHeading, visibleNearTop)
	widthBottom := selectorPopupWidth(120, seasonLines, rightHeading, visibleNearBottom)

	if widthFromAll < widthTop || widthFromAll < widthBottom {
		t.Fatalf("expected all-items width to cover every scroll window, got all=%d top=%d bottom=%d", widthFromAll, widthTop, widthBottom)
	}
	if widthTop == widthBottom {
		t.Fatalf("expected visible-window widths to differ in this fixture setup, got top=%d bottom=%d", widthTop, widthBottom)
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
	for _, want := range []string{"Season", "Round"} {
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
		m.match.HomeLineup = append(m.match.HomeLineup, site.PlayerLine{Name: fmt.Sprintf("Player%02d", i)})
	}

	view := m.View()
	if got := strings.Count(view, "\n") + 1; got > m.height {
		t.Fatalf("expected match view to fit terminal height, got %d lines for height %d\n%s", got, m.height, view)
	}
	if !strings.Contains(view, "Player01") {
		t.Fatalf("expected initial match view to show top content\n%s", view)
	}
	for _, want := range []string{"# Team", "Round", "LEG 2-1 LEC"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected match view to keep sidebar content %q visible\n%s", want, view)
		}
	}

	m.matchScroll = m.matchScrollLimit()
	view = m.View()
	if strings.Contains(view, "Player01") {
		t.Fatalf("expected scrolled match view to hide top content\n%s", view)
	}
	if !strings.Contains(view, "Player20") {
		t.Fatalf("expected scrolled match view to show later content\n%s", view)
	}
	for _, want := range []string{"# Team", "Round", "LEG 2-1 LEC"} {
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
		m.match.HomeLineup = append(m.match.HomeLineup, site.PlayerLine{Name: fmt.Sprintf("Player%02d", i)})
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
