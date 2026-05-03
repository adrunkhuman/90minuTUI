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

func TestLeagueSketchViewShowsStandingsFixturesAndStatus(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.height = 18
	m.lastFetchAt = time.Date(2026, time.March, 10, 21, 15, 0, 0, time.UTC)
	m.league.Title = "PKO Bank Polski Ekstraklasa 2025/2026"

	view := m.View()
	for _, want := range []string{
		"PKO Bank Polski Ekstraklasa 2025/2026",
		"#   Team",
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
		"#   Team",
		"Round 1",
		"FIXTURES",
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
		Events: []site.MatchEvent{{
			MinuteText: "17",
			Kind:       "GOAL",
			TeamSide:   "home",
			Text:       "Krzysztof Kubica 17",
		}, {
			MinuteText: "62",
			Kind:       "GOAL",
			TeamSide:   "away",
			Text:       "Samuel Mraz 62",
		}},
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
		"HT 1 – 0",
		"Fri 13 March 2026, 18:00",
		"Att. 3 542",
		"Ref. Damian Kos",
		"15°",
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
			{MinuteText: "39", Kind: "GOAL", TeamSide: "home", Text: "Wdowiak 39"},
			{MinuteText: "46", Kind: "SUB", TeamSide: "home", Text: "Igor Strzalek (86) -> Damian Nowak"},
			{MinuteText: "46", Kind: "SUB", TeamSide: "away", Text: "O. Lesniak -> Pllana"},
			{MinuteText: "52", Kind: "MISS", TeamSide: "away", Text: "Barkowskij 52 (nk)"},
			{MinuteText: "60", Kind: "GOAL", TeamSide: "home", Text: "Szkurin 60"},
			{MinuteText: "70", Kind: "GOAL", TeamSide: "away", Text: "Karol Czubak (k) 70"},
			{MinuteText: "85", Kind: "RC", TeamSide: "away", Text: "Pllana 85"},
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

	for _, want := range []string{
		"GKS Katowice", "2 – 1", "Lechia Gdansk",
		"TIMELINE",
		"HT 1 – 0",
		"FT 2 – 1",
		"Wdowiak", "39'",
		"Szkurin", "60'",
		"K. Czubak", "70'",
		"LINEUPS",
	} {
		if !strings.Contains(plainView, want) {
			t.Fatalf("expected match view to contain %q\n%s", want, view)
		}
	}

	for _, unwanted := range []string{
		"❌",
		"↕",
	} {
		if strings.Contains(plainView, unwanted) {
			t.Fatalf("expected view to omit %q\n%s", unwanted, view)
		}
	}

	for _, want := range []string{
		"46'",
		"D. Nowak", // sub-on player remains visible without crowding the lineup row
		"■",        // card badge in the dedicated card slot
	} {
		if !strings.Contains(plainView, want) {
			t.Fatalf("expected lineup section to contain %q\n%s", want, view)
		}
	}
	lineupIdx := strings.Index(plainView, "LINEUPS")
	if lineupIdx < 0 {
		t.Fatalf("expected lineup section\n%s", view)
	}
	lineupSection := plainView[lineupIdx:]
	if strings.Contains(lineupSection, "⚽") {
		t.Fatalf("expected lineup section to omit goal annotations\n%s", lineupSection)
	}
	if strings.Contains(plainView, "Substitutions") {
		t.Fatalf("expected substitution pane to be omitted\n%s", view)
	}
	if !strings.Contains(lineupSection, "(46' D. Nowak) I. Strzalek") {
		t.Fatalf("expected inline substitution annotation in lineup section\n%s", lineupSection)
	}
}

func TestRenderLineupPlayerRowColorsCards(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	row := renderLineupPlayerRow(
		"Booked Home", lineupCardMarker{color: colorYellow, ok: true},
		"Sent Off Away", lineupCardMarker{color: colorRed, ok: true},
		64,
	)

	if !strings.Contains(row, lipgloss.NewStyle().Foreground(colorYellow).Background(colorBgPanel).Bold(true).Render("■")) {
		t.Fatalf("expected yellow card marker to be colored\n%q", row)
	}
	if !strings.Contains(row, lipgloss.NewStyle().Foreground(colorRed).Background(colorBgPanel).Bold(true).Render("■")) {
		t.Fatalf("expected red card marker to be colored\n%q", row)
	}
	plain := ansi.Strip(row)
	divider := strings.IndexRune(plain, '│')
	if divider < 0 {
		t.Fatalf("expected center divider, got %q", plain)
	}
	dividerCol := ansi.StringWidth(plain[:divider])
	runes := []rune(plain)
	if string(runes[dividerCol-2:dividerCol]) != "■ " || string(runes[dividerCol+1:dividerCol+3]) != " ■" {
		t.Fatalf("expected card markers one cell away from center divider, got %q", plain)
	}
}

func TestAwayBookedSubstituteUsesColoredInlineCard(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	idx := playerEventIndex([]site.MatchEvent{
		{MinuteText: "81", Kind: "SUB", TeamSide: "away", Text: "Jan Starter -> Sebastian Kubiak"},
		{MinuteText: "85", Kind: "RC", TeamSide: "away", Text: "Sebastian Kubiak 85"},
	}, "away")
	entry := annotateLineupPlayer(site.PlayerLine{Name: "Jan Starter"}, idx)
	row := renderLineupPlayerRow("", lineupCardMarker{}, lineupDisplayName(entry, "away", 64), lineupCardForEntry(entry, idx), 96)
	plain := ansi.Strip(row)

	if !strings.Contains(plain, "J. Starter (■ S. Kubiak 81')") {
		t.Fatalf("expected substitute note with inline card, got %q", plain)
	}
	if strings.Contains(plain, "│■ ") {
		t.Fatalf("expected substitute card to stay out of starter card slot, got %q", plain)
	}
	if !strings.Contains(row, lipgloss.NewStyle().Foreground(colorRed).Background(colorBgPanel).Bold(true).Render("■")) {
		t.Fatalf("expected booked substitute card to keep red styling\n%q", row)
	}
}

func TestMatchDetailShowsAbbreviatedAwaySubstituteCardInline(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	m := sketchModel()
	m.match = &site.MatchPage{
		HomeTeam: "Home",
		AwayTeam: "Away",
		Score:    "1-1",
		Events: []site.MatchEvent{
			{MinuteText: "81", Kind: "SUB", TeamSide: "away", Text: "Jan Starter -> Sebastian Kubiak"},
			{MinuteText: "85", Kind: "YC", TeamSide: "away", Text: "S. Kubiak 85"},
		},
		AwayLineup: []site.PlayerLine{{Name: "Jan Starter"}},
	}

	content := m.matchDetailContent(96)
	plain := ansi.Strip(content)
	if !strings.Contains(plain, "J. Starter (■ S. Kubiak 81')") {
		t.Fatalf("expected substitute note with inline card\n%s", plain)
	}
	if strings.Contains(plain, "│■ ") {
		t.Fatalf("expected substitute card to stay out of starter card slot\n%s", plain)
	}
	if !strings.Contains(content, lipgloss.NewStyle().Foreground(colorYellow).Background(colorBgPanel).Bold(true).Render("■")) {
		t.Fatalf("expected substitute card to keep yellow styling\n%s", content)
	}
}

func TestHomeBookedReplacementDoesNotUseStarterCardSlot(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	m := sketchModel()
	m.match = &site.MatchPage{
		HomeTeam: "Home",
		AwayTeam: "Away",
		Score:    "1-1",
		Events: []site.MatchEvent{
			{MinuteText: "77", Kind: "SUB", TeamSide: "home", Text: "Pawel Kun -> Kacper Chodyna"},
			{MinuteText: "85", Kind: "YC", TeamSide: "home", Text: "K. Chodyna 85"},
		},
		HomeLineup: []site.PlayerLine{{Name: "Pawel Kun"}},
	}

	content := m.matchDetailContent(96)
	plain := ansi.Strip(content)
	if !strings.Contains(plain, "(77' K. Chodyna■) P. Kun") {
		t.Fatalf("expected replacement card attached to replacement note\n%s", plain)
	}
	if strings.Contains(plain, "P. Kun ■│") {
		t.Fatalf("expected replacement card not to look like starter card\n%s", plain)
	}
	if !strings.Contains(content, lipgloss.NewStyle().Foreground(colorYellow).Background(colorBgPanel).Bold(true).Render("■")) {
		t.Fatalf("expected replacement card to keep yellow styling\n%s", content)
	}
}

func TestRenderLineupPlayerRowMutesSubstitutionNotes(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	row := renderLineupPlayerRow("(46' D. Nowak) I. Strzalek", lineupCardMarker{}, "J. Wilson-Esbrand (J. Grzesik 46')", lineupCardMarker{}, 96)
	noteStyle := lipgloss.NewStyle().Foreground(colorTextMuted).Background(colorBgPanel)
	nameStyle := lipgloss.NewStyle().Foreground(colorTextSecondary).Background(colorBgPanel)

	if !strings.Contains(row, noteStyle.Render("(")) || !strings.Contains(row, noteStyle.Render("4")) {
		t.Fatalf("expected substitution note to use muted style\n%q", row)
	}
	if !strings.Contains(row, nameStyle.Render("I")) || !strings.Contains(row, nameStyle.Render("J")) {
		t.Fatalf("expected player names to keep lineup body style\n%q", row)
	}
}

func TestRenderLineupPlayerRowReservesBlankCardSlots(t *testing.T) {
	row := ansi.Strip(renderLineupPlayerRow("Home", lineupCardMarker{}, "Away", lineupCardMarker{}, 41))
	runes := []rune(row)
	divider := strings.IndexRune(row, '│')
	if divider < 0 {
		t.Fatalf("expected centered divider with card slots, got %q", row)
	}
	dividerCol := ansi.StringWidth(row[:divider])
	if dividerCol < 2 || dividerCol+3 > len(runes) {
		t.Fatalf("expected centered divider with card slots, got %q", row)
	}
	if string(runes[dividerCol-2:dividerCol]) != "  " {
		t.Fatalf("expected blank home card slot before divider, got %q", row)
	}
	if string(runes[dividerCol+1:dividerCol+3]) != "  " {
		t.Fatalf("expected blank away card slot after divider, got %q", row)
	}
}

func TestRenderLineupsLabelCentersELetter(t *testing.T) {
	line := ansi.Strip(renderLineupsLabel(41))
	start := strings.Index(line, "LINEUPS")
	if start < 0 {
		t.Fatalf("expected LINEUPS label, got %q", line)
	}
	visibleStart := ansi.StringWidth(line[:start])
	if visibleStart+3 != 41/2 {
		t.Fatalf("expected E in LINEUPS at center, got label start %d in %q", start, line)
	}
}

func TestRenderSectionLabelCentersFourthLetter(t *testing.T) {
	line := ansi.Strip(renderSectionLabel("TIMELINE", 41))
	start := strings.Index(line, "TIMELINE")
	if start < 0 {
		t.Fatalf("expected TIMELINE label, got %q", line)
	}
	visibleStart := ansi.StringWidth(line[:start])
	if visibleStart+3 != 41/2 {
		t.Fatalf("expected E in TIMELINE at center, got label start %d in %q", start, line)
	}
}

func TestGoalTimelineRowsKeepSideAssignment(t *testing.T) {
	rows := goalTimelineRows([]site.MatchEvent{
		{MinuteText: "12", Kind: "GOAL", TeamSide: "home", Text: "Home Scorer 12"},
		{MinuteText: "44", Kind: "GOAL", TeamSide: "away", Text: "Away Scorer 44"},
		{MinuteText: "89", Kind: "GOAL", TeamSide: "home", Text: "Late Winner 89"},
	})

	if len(rows) != 3 {
		t.Fatalf("expected three chronological timeline rows, got %+v", rows)
	}
	if rows[0].home != "H. Scorer 12'" || rows[0].away != "—" {
		t.Fatalf("expected first goal on home side, got %+v", rows[0])
	}
	if rows[1].home != "—" || rows[1].away != "44' A. Scorer" {
		t.Fatalf("expected second goal on away side, got %+v", rows[1])
	}
	if rows[2].home != "L. Winner 89'" || rows[2].away != "—" {
		t.Fatalf("expected third goal on home side, got %+v", rows[2])
	}
}

func TestHalftimeScoreDisplayAvoidsInventingSparseHT(t *testing.T) {
	if got := halftimeScoreDisplay(&site.MatchPage{Score: "1-0"}); got != "HT —" {
		t.Fatalf("expected sparse match to avoid invented HT score, got %q", got)
	}
	got := halftimeScoreDisplay(&site.MatchPage{Events: []site.MatchEvent{{MinuteText: "90", Kind: "GOAL", TeamSide: "home", Text: "Winner 90"}}})
	if got != "HT 0 – 0" {
		t.Fatalf("expected second-half goal to imply goalless HT, got %q", got)
	}
}

func TestMatchDetailOrdersScorersAroundHTAndFT(t *testing.T) {
	m := sketchModel()
	m.match = &site.MatchPage{
		HomeTeam: "Home",
		AwayTeam: "Away",
		Score:    "1-1",
		Events: []site.MatchEvent{
			{MinuteText: "35", Kind: "GOAL", TeamSide: "home", Text: "First Scorer 35"},
			{MinuteText: "56", Kind: "GOAL", TeamSide: "away", Text: "Second Scorer 56"},
		},
	}

	content := ansi.Strip(m.matchDetailContent(80))
	first := strings.Index(content, "F. Scorer 35'")
	ht := strings.Index(content, "HT 1 – 0")
	second := strings.Index(content, "56' S. Scorer")
	ft := strings.Index(content, "FT 1 – 1")
	if first < 0 || ht < 0 || second < 0 || ft < 0 {
		t.Fatalf("expected scorer, HT, and FT rows\n%s", content)
	}
	if !(first < ht && ht < second && second < ft) {
		t.Fatalf("expected first-half goals before HT, second-half goals before FT\n%s", content)
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

func TestMatchDetailKeepsInlineSubstitutionAnnotationsInLineups(t *testing.T) {
	m := sketchModel()
	m.width = 140
	m.matchView = true
	m.match = &site.MatchPage{
		HomeTeam: "Piast Gliwice",
		AwayTeam: "Radomiak Radom",
		Score:    "3-1",
		Events: []site.MatchEvent{
			{MinuteText: "66", Kind: "SUB", TeamSide: "home", Text: "Jason Lokilo -> Oskar Lesniak"},
			{MinuteText: "46", Kind: "SUB", TeamSide: "away", Text: "J. Wilson-Esbrand -> J. Grzesik"},
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
		{MinuteText: "17", Kind: "GOAL", TeamSide: "home", Text: "Krzysztof Kubica 17"},
		{MinuteText: "30", Kind: "GOAL", TeamSide: "away", Text: "Karol Czubak (k) 30"},
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
		{MinuteText: "39", Kind: "GOAL", TeamSide: "home", Text: "Wdowiak 39"},
		{MinuteText: "60", Kind: "GOAL", TeamSide: "home", Text: "Szkurin 60"},
		{MinuteText: "85", Kind: "RC", TeamSide: "away", Text: "Pllana 85"},
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
		{MinuteText: "60", Kind: "RC", TeamSide: "away", Text: "Pllana 60"},
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
		{MinuteText: "39", Kind: "GOAL", TeamSide: "home", Text: "Wdowiak 39"},
		{MinuteText: "60", Kind: "SUB", TeamSide: "home", Text: "Igor Strzalek -> Damian Nowak"},
		{MinuteText: "72", Kind: "YC", TeamSide: "away", Text: "Pllana 72"},
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
		{MinuteText: "20", Kind: "YC", TeamSide: "home", Text: "Pllana 20"},
		{MinuteText: "85", Kind: "RC", TeamSide: "home", Text: "Pllana 85"},
	}, "home")

	if got := cardAnnotation(site.PlayerLine{Name: "Pllana"}, idx); got != eventPrefix("RC") {
		t.Fatalf("expected red-card badge, got %q", got)
	}
}

func TestCardAnnotationDoesNotApplyAbbreviatedCardToFullSameInitialNames(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{{
		MinuteText: "85",
		Kind:       "YC",
		TeamSide:   "home",
		Text:       "J. Kowalski 85",
	}}, "home")

	if got := cardAnnotation(site.PlayerLine{Name: "Jan Kowalski"}, idx); got != "" {
		t.Fatalf("expected full Jan Kowalski not to inherit abbreviated card, got %q", got)
	}
	if got := cardAnnotation(site.PlayerLine{Name: "Jerzy Kowalski"}, idx); got != "" {
		t.Fatalf("expected full Jerzy Kowalski not to inherit abbreviated card, got %q", got)
	}
	if got := cardAnnotation(site.PlayerLine{Name: "J. Kowalski"}, idx); ansi.Strip(got) != "■" {
		t.Fatalf("expected abbreviated row to match abbreviated card, got %q", got)
	}
}

func TestAnnotatedLineupMatchesSubstitutionByCompactNameNotSurnameOnly(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{{
		MinuteText: "60",
		Kind:       "SUB",
		TeamSide:   "home",
		Text:       "Jan Kowalski -> Piotr Kowalski",
	}}, "home")

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

func TestAnnotatedLineupMatchesAbbreviatedPlayerToFullSubstitution(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{{
		MinuteText: "60",
		Kind:       "SUB",
		TeamSide:   "away",
		Text:       "Jan Starter -> Sebastian Kubiak",
	}}, "away")

	got := annotatedLineup([]site.PlayerLine{{Name: "S. Kubiak"}}, idx)
	if len(got) != 1 {
		t.Fatalf("expected one lineup entry, got %#v", got)
	}
	if got[0].enteredAt != "60'" || got[0].replaced != "Jan Starter" {
		t.Fatalf("expected abbreviated lineup player to receive substitution note, got %#v", got[0])
	}
}

func TestAnnotatedLineupDistinguishesSameInitialSameSurname(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{{
		MinuteText: "60",
		Kind:       "SUB",
		TeamSide:   "home",
		Text:       "Jan Kowalski -> Piotr Kowalski",
	}}, "home")

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
	if got[0].enteredAt != "" || got[0].leftAt != "" || got[0].replacedBy != "" {
		t.Fatalf("expected same-initial teammate to remain unannotated, got %#v", got[0])
	}
	if got[1].player.Name != "Jan Kowalski" || got[1].replacedBy != "Piotr Kowalski" {
		t.Fatalf("expected substitution to match full name, got %#v", got)
	}
}

func TestAnnotatedLineupDoesNotApplyAbbreviatedSubstitutionToFullSameInitialNames(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{{
		MinuteText: "60",
		Kind:       "SUB",
		TeamSide:   "home",
		Text:       "J. Kowalski -> Piotr Kowalski",
	}}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jan Kowalski"}, {Name: "Jerzy Kowalski"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected no synthetic rows for ambiguous abbreviated substitution, got %#v", got)
	}
	for _, entry := range got {
		if entry.enteredAt != "" || entry.leftAt != "" || entry.replacedBy != "" {
			t.Fatalf("expected full same-initial names to stay unannotated for abbreviated event, got %#v", got)
		}
	}
}

func TestAnnotatedLineupDoesNotApplyFullSubstitutionToAmbiguousAbbreviatedRow(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{{
		MinuteText: "60",
		Kind:       "SUB",
		TeamSide:   "home",
		Text:       "Jan Kowalski -> Piotr Kowalski",
	}}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "J. Kowalski"}, {Name: "Jerzy Kowalski"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected no synthetic rows for ambiguous abbreviated lineup row, got %#v", got)
	}
	if got[0].leftAt != "" || got[0].replacedBy != "" {
		t.Fatalf("expected ambiguous abbreviated row to stay unannotated, got %#v", got[0])
	}
}

func TestAnnotatedLineupDoesNotDuplicateExistingAbbreviatedSyntheticEntrant(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		{MinuteText: "60", Kind: "SUB", TeamSide: "away", Text: "Jan Starter -> Sebastian Kubiak"},
		{MinuteText: "78", Kind: "SUB", TeamSide: "away", Text: "Sebastian Kubiak -> Adam Next"},
	}, "away")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jan Starter"}, {Name: "S. Kubiak"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected no duplicate synthetic entrant for existing abbreviated row, got %#v", got)
	}
	if got[1].player.Name != "S. Kubiak" || got[1].enteredAt != "60'" || got[1].leftAt != "78'" || got[1].replacedBy != "Adam Next" {
		t.Fatalf("expected existing abbreviated row to carry both substitution notes, got %#v", got[1])
	}
}

func TestAnnotatedLineupDoesNotDuplicateExistingFullEntrantFromAbbreviatedSubstitution(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		{MinuteText: "60", Kind: "SUB", TeamSide: "away", Text: "Jan Starter -> S. Kubiak"},
		{MinuteText: "78", Kind: "SUB", TeamSide: "away", Text: "Sebastian Kubiak -> Adam Next"},
	}, "away")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jan Starter"}, {Name: "Sebastian Kubiak"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected no duplicate synthetic row for existing full entrant, got %#v", got)
	}
	if got[1].enteredAt != "60'" || got[1].replaced != "Jan Starter" || got[1].leftAt != "78'" || got[1].replacedBy != "Adam Next" {
		t.Fatalf("expected existing full entrant to carry both substitution notes, got %#v", got[1])
	}
}

func TestAnnotatedLineupDoesNotAttachAmbiguousAbbreviatedCardToSubstituteNote(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		{MinuteText: "60", Kind: "SUB", TeamSide: "home", Text: "Starter One -> Jan Kowalski"},
		{MinuteText: "85", Kind: "YC", TeamSide: "home", Text: "J. Kowalski 85"},
	}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Starter One"}, {Name: "Jerzy Kowalski"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected no synthetic rows for substitute without later exit, got %#v", got)
	}
	if got[0].replacedByYC.ok {
		t.Fatalf("expected ambiguous abbreviated card not to attach to Jan Kowalski note, got %#v", got[0])
	}
}

func TestAnnotatedLineupAllowsDistinctFullSyntheticEntrantsWithSameCompactKey(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		{MinuteText: "60", Kind: "SUB", TeamSide: "home", Text: "Starter One -> Oskar Lesniak"},
		{MinuteText: "61", Kind: "SUB", TeamSide: "home", Text: "Starter Two -> Olaf Lesniak"},
		{MinuteText: "78", Kind: "SUB", TeamSide: "home", Text: "Oskar Lesniak -> Next One"},
		{MinuteText: "79", Kind: "SUB", TeamSide: "home", Text: "Olaf Lesniak -> Next Two"},
	}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Starter One"}, {Name: "Starter Two"}}, idx)
	if len(got) != 4 {
		t.Fatalf("expected two distinct synthetic entrants, got %#v", got)
	}
	oskar := lineupEntryByName(got, "Oskar Lesniak")
	olaf := lineupEntryByName(got, "Olaf Lesniak")
	if oskar.player.Name == "" || olaf.player.Name == "" {
		t.Fatalf("expected distinct same-compact synthetic entrants, got %#v", got)
	}
	if oskar.enteredAt != "60'" || oskar.leftAt != "78'" || oskar.replacedBy != "Next One" {
		t.Fatalf("unexpected Oskar Lesniak entry: %#v", oskar)
	}
	if olaf.enteredAt != "61'" || olaf.leftAt != "79'" || olaf.replacedBy != "Next Two" {
		t.Fatalf("unexpected Olaf Lesniak entry: %#v", olaf)
	}
}

func lineupEntryByName(entries []lineupEntry, name string) lineupEntry {
	for _, entry := range entries {
		if entry.player.Name == name {
			return entry
		}
	}
	return lineupEntry{}
}

func TestAnnotatedLineupKeepsBookedEntrantInReplacementNote(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{
		{MinuteText: "66", Kind: "SUB", TeamSide: "home", Text: "Jason Lokilo -> Oskar Lesniak"},
		{MinuteText: "84", Kind: "YC", TeamSide: "home", Text: "Oskar Lesniak 84"},
	}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jason Lokilo"}}, idx)
	if len(got) != 1 {
		t.Fatalf("expected only starter row, got %#v", got)
	}
	if got[0].player.Name != "Jason Lokilo" || got[0].replacedBy != "Oskar Lesniak" || got[0].leftAt != "66'" || !got[0].replacedByYC.ok || got[0].replacedByYC.color != colorYellow {
		t.Fatalf("expected starter row to retain substitution note, got %#v", got)
	}
}

func TestAnnotatedLineupSkipsMissingEntrantWithoutBadgeEvent(t *testing.T) {
	idx := playerEventIndex([]site.MatchEvent{{
		MinuteText: "66",
		Kind:       "SUB",
		TeamSide:   "home",
		Text:       "Jason Lokilo -> Oskar Lesniak",
	}}, "home")

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
		{MinuteText: "66", Kind: "SUB", TeamSide: "home", Text: "Jason Lokilo -> Oskar Lesniak"},
		{MinuteText: "78", Kind: "SUB", TeamSide: "home", Text: "Oskar Lesniak -> Michal Smith"},
		{MinuteText: "72", Kind: "YC", TeamSide: "home", Text: "Oskar Lesniak 72"},
	}, "home")

	got := annotatedLineup([]site.PlayerLine{{Name: "Jason Lokilo"}}, idx)
	if len(got) != 2 {
		t.Fatalf("expected starter plus synthetic entrant, got %#v", got)
	}
	if got[1].player.Name != "Oskar Lesniak" || got[1].enteredAt != "66'" || got[1].replaced != "Jason Lokilo" || got[1].leftAt != "78'" || got[1].replacedBy != "Michal Smith" {
		t.Fatalf("expected synthetic entrant row to keep both substitution notes, got %#v", got[1])
	}
}

func TestFormatLineupPlayerShowsBookedReplacementInsideHomeNote(t *testing.T) {
	home := formatLineupPlayer(lineupEntry{
		player:       site.PlayerLine{Name: "Oskar Jakubczyk"},
		leftAt:       "72'",
		replacedBy:   "Michal Rzuchowski",
		replacedByYC: lineupCardMarker{color: colorYellow, ok: true},
	}, "home", 64)

	if got := ansi.Strip(home); got != "(72' M. Rzuchowski■) O. Jakubczyk" {
		t.Fatalf("unexpected home booked-replacement label: %q", got)
	}
}

func TestFormatLineupPlayerShowsBookedReplacedPlayerInsideAwayNote(t *testing.T) {
	away := formatLineupPlayer(lineupEntry{
		player:     site.PlayerLine{Name: "Jakub Sypek"},
		enteredAt:  "46'",
		replaced:   "Jan Kowalczyk",
		replacedYC: lineupCardMarker{color: colorYellow, ok: true},
	}, "away", 64)

	if got := ansi.Strip(away); got != "J. Sypek (for ■ J. Kowalczyk 46')" {
		t.Fatalf("unexpected away booked-replaced label: %q", got)
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
	row := ansi.Strip(renderLineupHeaderRow("Piast Gliwice", "Radomiak Radom", 76))

	if !strings.Contains(row, "Piast Gliwice") || !strings.Contains(row, "Radomiak Radom") {
		t.Fatalf("expected lineup header row to contain both team names, got %q", row)
	}
	if strings.Contains(row, "|") || strings.Contains(row, "│") {
		t.Fatalf("expected lineup header row to omit separator, got %q", row)
	}

	leftEnd := strings.Index(row, "Piast Gliwice") + len("Piast Gliwice")
	rightStart := strings.Index(row, "Radomiak Radom")
	gapMid := leftEnd + ((rightStart - leftEnd) / 2)
	dividerMid := 76 / 2
	if diff := gapMid - dividerMid; diff < -1 || diff > 1 {
		t.Fatalf("expected lineup header gap to stay centered\nrow: %q", row)
	}
	if rightStart-leftEnd < 3 {
		t.Fatalf("expected team names to sit away from the center, got %q", row)
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
		if !strings.Contains(ansi.Strip(line), "LINEUPS") || i+1 >= len(lines) {
			continue
		}

		header := lines[i+1]
		stripped := ansi.Strip(header)
		if !strings.Contains(stripped, "Piast Gliwice") || !strings.Contains(stripped, "Radomiak Radom") {
			t.Fatalf("expected lineup team header row after section title, got %q", stripped)
		}
		if strings.Contains(stripped, "│") {
			t.Fatalf("expected lineup team header to omit center divider, got %q", stripped)
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
			{MinuteText: "46", Kind: "SUB", TeamSide: "home", Text: "Jan Kowalski -> Piotr Kowalski"},
			{MinuteText: "72", Kind: "YC", TeamSide: "away", Text: "Nowak 72"},
		},
	}

	content := ansi.Strip(m.matchDetailContent(80))
	if !strings.Contains(content, "FT 1 – 0") {
		t.Fatalf("expected final score block even without visible header events\n%s", content)
	}
}

func TestMatchDetailContentDoesNotShowFTForUnknownScore(t *testing.T) {
	m := sketchModel()
	m.match = &site.MatchPage{
		HomeTeam: "Motor Lublin",
		AwayTeam: "Zaglebie Lubin",
		Score:    "-",
	}

	content := ansi.Strip(m.matchDetailContent(80))
	if !strings.Contains(content, "Motor Lublin") || !strings.Contains(content, "vs") || !strings.Contains(content, "Zaglebie Lubin") {
		t.Fatalf("expected unknown-score match title, got\n%s", content)
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "FT ") {
			t.Fatalf("expected unknown-score match to omit final-score row\n%s", content)
		}
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

func TestMatchMetaAxisLineCentersOddMiddleSection(t *testing.T) {
	width := 80
	line := matchMetaAxisLine([]string{
		"Wed 29 April 2026, 21:00",
		"Att. 68 421",
		"Ref. Danny Desmond Makkelie",
	}, width)
	middle := "Att. 68 421"
	start := strings.Index(line, middle)
	if start < 0 {
		t.Fatalf("expected metadata line to contain middle section, got %q", line)
	}

	got := ansi.StringWidth(line[:start]) + ansi.StringWidth(middle)/2
	if want := scorelineAxisColumn(width); got != want {
		t.Fatalf("expected middle metadata section to align with score axis %d, got %d in %q", want, got, line)
	}
}

func TestMatchMetaAxisLineCentersEvenMiddleSeparator(t *testing.T) {
	width := 120
	line := matchMetaAxisLine([]string{
		"Wed 29 April 2026, 21:00",
		"Att. 68 421",
		"Ref. Danny Desmond Makkelie",
		"15°",
	}, width)
	middle := "Att. 68 421  ·  Ref. Danny Desmond Makkelie"
	start := strings.Index(line, middle)
	if start < 0 {
		t.Fatalf("expected metadata line to contain middle separator, got %q", line)
	}

	got := ansi.StringWidth(line[:start]) + ansi.StringWidth("Att. 68 421  ")
	if want := scorelineAxisColumn(width); got != want {
		t.Fatalf("expected middle metadata separator to align with score axis %d, got %d in %q", want, got, line)
	}
}

func TestMatchMetaAxisLineDropsDateBeforeTruncatingAttendance(t *testing.T) {
	line := matchMetaAxisLine([]string{
		"Sat 2 May 2026, 20:15",
		"Att. 8 470",
		"Ref. Karol Arys",
		"15°",
	}, 70)

	if strings.Contains(line, "May") || strings.Contains(line, "02/05") {
		t.Fatalf("expected date to be dropped at narrow width, got %q", line)
	}
	if !strings.Contains(line, "Att. 8 470") {
		t.Fatalf("expected full attendance to remain visible, got %q", line)
	}
	if !strings.Contains(line, "Ref. Karol Arys") {
		t.Fatalf("expected full referee to remain visible, got %q", line)
	}
}

func scorelineAxisColumn(width int) int {
	scoreline := matchTitleLine("Home", "1-1", "Away", width)
	dash := strings.Index(scoreline, "–")
	if dash < 0 {
		return -1
	}
	return ansi.StringWidth(scoreline[:dash])
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

func TestLeagueViewMarksNonDrillableFixturesWhenSpaceAllows(t *testing.T) {
	m := sketchModel()
	m.width = 160
	m.league.Rounds[0].Fixtures[0].MatchURL = ""
	m.league.Rounds[0].Fixtures[0].MatchID = ""
	m.league.Rounds[0].Fixtures[0].Score = "-"

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "[no details]") {
		t.Fatalf("expected league fixture grid to mark non-drillable fixture\n%s", view)
	}
}

func TestFixtureGridNonDrillableMarkerDoesNotReplaceDate(t *testing.T) {
	line := ansi.Strip(formatFixtureGridRow(&site.Fixture{
		Home:     "Legia Warszawa",
		Away:     "Lech Poznan",
		Score:    "-",
		WhenInfo: "1 maja, 20:30",
	}, false, 84, "Ekstraklasa", false, colorBgPanel))
	drillable := ansi.Strip(formatFixtureGridRow(&site.Fixture{
		Home:     "Legia Warszawa",
		Away:     "Lech Poznan",
		Score:    "0-1",
		WhenInfo: "2 maja, 20:15",
		MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1",
	}, false, 84, "Ekstraklasa", false, colorBgPanel))

	if !strings.Contains(line, "01/05 20:30") {
		t.Fatalf("expected date to remain visible, got %q", line)
	}
	if !strings.Contains(line, "[no details]") {
		t.Fatalf("expected no-details marker when space allows, got %q", line)
	}
	if strings.Index(line, "[no details]") > strings.Index(line, "01/05 20:30") {
		t.Fatalf("expected no-details marker before date, got %q", line)
	}
	dateColumn := ansi.StringWidth(line[:strings.Index(line, "01/05 20:30")])
	drillableDateColumn := ansi.StringWidth(drillable[:strings.Index(drillable, "02/05 20:15")])
	if dateColumn != drillableDateColumn {
		t.Fatalf("expected date column to stay aligned, got %q and %q", line, drillable)
	}
	if ansi.StringWidth(line[:strings.Index(line, "Lech Poznan")]) != ansi.StringWidth(drillable[:strings.Index(drillable, "Lech Poznan")]) {
		t.Fatalf("expected fixture columns to stay aligned, got %q and %q", line, drillable)
	}
}

func TestFixtureGridSuppressesDetailsMarkerBeforeOverTruncatingTeams(t *testing.T) {
	line := ansi.Strip(formatFixtureGridRow(&site.Fixture{
		Home:     "GKS Katowice",
		Away:     "Bruk-Bet Termalica Nieciecza",
		Score:    "-",
		WhenInfo: "3 maja, 12:15",
	}, false, 57, "Ekstraklasa", false, colorBgPanel))

	if strings.Contains(line, "[no details]") {
		t.Fatalf("expected details marker to be suppressed before crushing team names, got %q", line)
	}
	if !strings.Contains(line, "GKS Katowice") {
		t.Fatalf("expected home team to remain readable, got %q", line)
	}
	if !strings.Contains(line, "03/05 12:15") {
		t.Fatalf("expected date to remain visible, got %q", line)
	}
}

func TestStatusBarViewReflectsFixtureDrillability(t *testing.T) {
	m := sketchModel()
	m.width = 120

	status := ansi.Strip(m.statusBarView())
	if !strings.Contains(status, "enter  details") {
		t.Fatalf("expected drillable status hint, got %q", status)
	}

	m.league.Rounds[0].Fixtures[0].MatchURL = ""
	m.league.Rounds[0].Fixtures[0].MatchID = ""
	status = ansi.Strip(m.statusBarView())
	if !strings.Contains(status, "enter  unavail") {
		t.Fatalf("expected non-drillable status hint, got %q", status)
	}
}

func TestStatusBarViewLeagueViewIncludesReloadHint(t *testing.T) {
	m := sketchModel()
	m.width = 120

	status := ansi.Strip(m.statusBarView())
	if !strings.Contains(status, "r  reload") {
		t.Fatalf("expected reload hint in league status bar, got %q", status)
	}
}

func TestStatusBarPaintsSpacerBackground(t *testing.T) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	m := sketchModel()
	m.width = 120
	status := m.statusBarView()

	if ansi.StringWidth(status) != m.width {
		t.Fatalf("expected status bar width %d, got %d", m.width, ansi.StringWidth(status))
	}
	if !strings.Contains(status, statusSpace(2)) {
		t.Fatalf("expected status bar separators to carry status background\n%q", status)
	}
	if !strings.Contains(status, statusText("never")) {
		t.Fatalf("expected clock to carry status background\n%q", status)
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

func TestRenderPitchStandingsWindowHighlightsHomeAndAway(t *testing.T) {
	rows := []site.StandingRow{
		{Position: 1, Team: "Cracovia", Played: 30, Points: 38},
		{Position: 2, Team: "Pogon Szczecin", Played: 30, Points: 38},
		{Position: 3, Team: "Lech Poznan", Played: 31, Points: 55},
	}
	fixture := &site.Fixture{Home: "Cracovia", Away: "Pogon Szczecin"}

	lines := renderPitchStandingsWindow(rows, fixture, 54, 3)
	if len(lines) != 3 {
		t.Fatalf("expected standings lines, got %d", len(lines))
	}
	if !strings.Contains(ansi.Strip(lines[0]), "▌") || !strings.Contains(ansi.Strip(lines[1]), "▌") {
		t.Fatalf("expected both fixture teams to be selected\n%q\n%q", lines[0], lines[1])
	}
	if strings.Contains(ansi.Strip(lines[2]), "▌") {
		t.Fatalf("expected non-fixture team to stay unselected\n%q", lines[2])
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
		"#   Team",
		"Legia Warszawa",
		"Lech Poznan",
		"24/01 20:30",
		"SEASON",
		"2024/2025",
		"Ekstraklasa",
		"esc  close",
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
		if strings.Contains(line, "SEASON") && strings.Contains(line, "COMPETITIONS") {
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
	for _, want := range []string{"SEASON", "Round"} {
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
	for _, want := range []string{"#   Team", "Round", "FIXTURES", "LEG 2-1 LEC"} {
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
	for _, want := range []string{"#   Team", "Round", "FIXTURES", "LEG 2-1 LEC"} {
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
	for _, want := range []string{"Team 01", "Team 16", "FIXTURES", "LEG 2-1 LEC"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected match sidebar to keep full standings and minilist content %q\n%s", want, view)
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
