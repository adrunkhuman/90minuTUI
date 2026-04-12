package ui

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/x/ansi"
)

var polishMonthReplacer = strings.NewReplacer(
	"stycznia", "January",
	"lutego", "February",
	"marca", "March",
	"kwietnia", "April",
	"maja", "May",
	"czerwca", "June",
	"lipca", "July",
	"sierpnia", "August",
	"wrzesnia", "September",
	"września", "September",
	"pazdziernika", "October",
	"października", "October",
	"listopada", "November",
	"grudnia", "December",
)

var roundSuffixRe = regexp.MustCompile(`(?i)\s*-\s*(?:kolejka|runda)\s+(\d+)\s*$`)
var matchDatePrefixRe = regexp.MustCompile(`^\d{1,2}\s+\p{L}+\s+\d{4},\s+\d{1,2}:\d{2}`)
var leadingAttendanceRe = regexp.MustCompile(`^(\d[\d ]*)\b`)
var fixtureWhenInfoRe = regexp.MustCompile(`(?i)^(\d{1,2})\s+([\p{L}]+)(?:\s+\d{4})?,\s*(\d{1,2}:\d{2})`)
var playerNumberPrefixRe = regexp.MustCompile(`^\(\d+\)\s*`)
var playerNumberSuffixRe = regexp.MustCompile(`\s+\(\d+\)$`)
var trailingParenRe = regexp.MustCompile(`^(.*?)(\s+\([^)]*\))$`)
var substitutionMinutePrefixRe = regexp.MustCompile(`^\d+'?\s*`)

func renderSeasonsWindow(seasons []site.Season, cursor int) []string {
	if len(seasons) == 0 {
		return []string{"(none)"}
	}

	start, end := windowBounds(len(seasons), cursor, 10)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		marker := ""
		if seasons[i].Current {
			marker = " *"
		}

		lines = append(lines, fmt.Sprintf("%s%s%s", prefix, seasons[i].Label, marker))
	}

	return lines
}

func renderCompetitionWindow(items []site.Competition, cursor int) []string {
	if len(items) == 0 {
		return []string{"(none)"}
	}

	start, end := windowBounds(len(items), cursor, 18)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		lines = append(lines, prefix+items[i].Name)
	}

	return lines
}

func renderFixtureWindow(fixtures []site.Fixture, cursor, maxItems, width int, compact bool) []string {
	if len(fixtures) == 0 {
		return nil
	}
	if maxItems <= 0 {
		return nil
	}

	start, end := windowBounds(len(fixtures), cursor, maxItems)
	whenWidth := 0
	scoreWidth := 0
	suffixWidth := 0
	for i := start; i < end; i++ {
		whenWidth = max(whenWidth, len([]rune(formatFixtureWhenInfo(fixtures[i].WhenInfo))))
		scoreWidth = max(scoreWidth, ansi.StringWidth(normalizeScore(fixtures[i].Score)))
		suffixWidth = max(suffixWidth, ansi.StringWidth(fixtureAvailabilitySuffix(&fixtures[i], width-2, compact)))
	}
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		isCursor := i == cursor
		// "› " and "  " are both 2 visible chars; accent marker on cursor row.
		var prefix string
		if isCursor {
			prefix = styleAccent.Render("›") + " "
		} else {
			prefix = "  "
		}

		lineWidth := width - 2 // 2-char prefix in both cases
		suffix := fixtureAvailabilitySuffix(&fixtures[i], lineWidth, compact)
		line := prefix + fixtureLine(&fixtures[i], lineWidth-suffixWidth, whenWidth, scoreWidth, compact)
		if suffix != "" {
			line += styleDim.Render(suffix)
		} else if suffixWidth > 0 {
			line += strings.Repeat(" ", suffixWidth)
		}
		if whenInfo := formatFixtureWhenInfo(fixtures[i].WhenInfo); whenInfo != "" {
			line += styleDim.Render("  " + whenInfo)
		}
		lines = append(lines, line)
	}

	return lines
}

func renderStandingsWindow(rows []site.StandingRow, fixture *site.Fixture, width, maxItems int) []string {
	if len(rows) == 0 {
		return nil
	}
	if maxItems <= 0 {
		return nil
	}

	start, end := anchoredWindowBounds(len(rows), standingSelectionIndices(rows, fixture), maxItems)
	teamWidth := standingsTeamWidth(rows, width)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		selected := fixture != nil && (strings.EqualFold(rows[i].Team, fixture.Home) || strings.EqualFold(rows[i].Team, fixture.Away))
		lines = append(lines, formatStandingRow(rows[i], selected, teamWidth, width))
	}

	return lines
}

func standingSelectionIndices(rows []site.StandingRow, fixture *site.Fixture) []int {
	if fixture == nil {
		return nil
	}

	indices := make([]int, 0, 2)
	for i, row := range rows {
		if strings.EqualFold(row.Team, fixture.Home) || strings.EqualFold(row.Team, fixture.Away) {
			indices = append(indices, i)
		}
	}

	return indices
}

func windowBounds(total, cursor, maxItems int) (int, int) {
	if maxItems <= 0 {
		return 0, 0
	}
	if total <= maxItems {
		return 0, total
	}

	half := maxItems / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}

	end := start + maxItems
	if end > total {
		end = total
		start = end - maxItems
	}

	return start, end
}

func anchoredWindowBounds(total int, anchors []int, maxItems int) (int, int) {
	// Keep the highlighted home/away rows visible together when standings overflow.
	if maxItems <= 0 {
		return 0, 0
	}
	if total <= maxItems {
		return 0, total
	}
	if len(anchors) == 0 {
		return windowBounds(total, 0, maxItems)
	}

	minAnchor := anchors[0]
	maxAnchor := anchors[0]
	for _, anchor := range anchors[1:] {
		if anchor < minAnchor {
			minAnchor = anchor
		}
		if anchor > maxAnchor {
			maxAnchor = anchor
		}
	}

	span := maxAnchor - minAnchor + 1
	if span >= maxItems {
		return windowBounds(total, minAnchor, maxItems)
	}

	start := minAnchor - (maxItems-span)/2
	if start < 0 {
		start = 0
	}
	end := start + maxItems
	if end > total {
		end = total
		start = end - maxItems
	}

	return start, end
}

func clamp(v, minV, maxV int) int {
	if maxV < minV {
		return minV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func renderPlayerLine(player site.PlayerLine) string {
	return formatPlayerLabel(player.Name)
}

func sortedEvents(events []site.MatchEvent) []site.MatchEvent {
	ordered := make([]site.MatchEvent, len(events))
	copy(ordered, events)

	sort.SliceStable(ordered, func(i, j int) bool {
		mi, hi := minuteSortKey(ordered[i].MinuteText)
		mj, hj := minuteSortKey(ordered[j].MinuteText)

		if hi != hj {
			return hi
		}
		if hi && mj != mi {
			return mi < mj
		}

		weightI := eventWeight(ordered[i].Kind)
		weightJ := eventWeight(ordered[j].Kind)
		if weightI != weightJ {
			return weightI < weightJ
		}

		return false
	})

	return ordered
}

func eventWeight(kind string) int {
	switch kind {
	case "GOAL":
		return 0
	case "MISS":
		return 1
	case "RC":
		return 2
	case "YC":
		return 3
	case "SUB":
		return 4
	default:
		return 9
	}
}

func minuteSortKey(text string) (int, bool) {
	if text == "" {
		return 0, false
	}

	parts := strings.SplitN(text, "+", 2)
	base := atoiOrNeg(parts[0])
	if base < 0 {
		return 0, false
	}

	extra := 0
	if len(parts) == 2 {
		extra = max(0, atoiOrNeg(parts[1]))
	}

	return base*100 + extra, true
}

func atoiOrNeg(s string) int {
	value := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &value); err != nil {
		return -1
	}
	return value
}

func substitutionPlayers(text string) (string, string) {
	parts := strings.SplitN(normalizeDisplayText(text), "->", 2)
	if len(parts) != 2 {
		return "", ""
	}

	outgoing := normalizeDisplayText(substitutionMinutePrefixRe.ReplaceAllString(strings.TrimSpace(parts[0]), ""))
	incoming := normalizeDisplayText(parts[1])
	if strings.EqualFold(outgoing, "sub") {
		outgoing = ""
	}

	return canonicalPlayerName(outgoing), canonicalPlayerName(incoming)
}

func faintText(text string) string {
	if text == "" {
		return ""
	}
	return "\x1b[2m" + text + "\x1b[0m"
}

func faintPenaltySuffix(text string) string {
	if text == "" || !strings.Contains(text, "(pen)") {
		return text
	}
	return strings.ReplaceAll(text, "(pen)", faintText("(pen)"))
}

func formatLeftEventLabel(kind, text string) string {
	prefix := eventPrefix(kind)
	if text == "" {
		return prefix
	}
	return text + " " + prefix
}

func formatRightEventLabel(kind, text string) string {
	prefix := eventPrefix(kind)
	if text == "" {
		return prefix
	}
	return prefix + " " + text
}

func eventPrefix(kind string) string {
	switch kind {
	case "GOAL":
		return "⚽"
	case "MISS":
		return "❌"
	case "SUB":
		return "↕"
	case "YC":
		return styleYellow.Render("■")
	case "RC":
		return styleRed.Render("■")
	default:
		return "•"
	}
}

func formatPlayerLabel(value string) string {
	cleaned := canonicalPlayerName(value)
	if cleaned == "" {
		return ""
	}

	suffixes := make([]string, 0, 2)
	for {
		matches := trailingParenRe.FindStringSubmatch(cleaned)
		if len(matches) != 3 {
			break
		}
		suffixes = append([]string{strings.TrimSpace(matches[2])}, suffixes...)
		cleaned = normalizeDisplayText(matches[1])
	}

	words := strings.Fields(cleaned)
	if len(words) >= 2 {
		last := words[len(words)-1]
		initials := make([]string, 0, len(words)-1)
		for _, word := range words[:len(words)-1] {
			r := []rune(word)
			if len(r) == 0 {
				continue
			}
			initials = append(initials, string(unicode.ToUpper(r[0]))+".")
		}
		cleaned = strings.TrimSpace(strings.Join(append(initials, last), " "))
	}

	if len(suffixes) > 0 {
		cleaned += " " + strings.Join(suffixes, " ")
	}

	return faintPenaltySuffix(cleaned)
}

func canonicalPlayerName(value string) string {
	cleaned := normalizeDisplayText(value)
	if cleaned == "" {
		return ""
	}

	cleaned = playerNumberPrefixRe.ReplaceAllString(cleaned, "")
	cleaned = playerNumberSuffixRe.ReplaceAllString(cleaned, "")

	suffixes := make([]string, 0, 2)
	for {
		matches := trailingParenRe.FindStringSubmatch(cleaned)
		if len(matches) != 3 {
			break
		}
		inner := strings.TrimSpace(strings.Trim(matches[2], " ()"))
		cleaned = normalizeDisplayText(matches[1])
		if digitsOnly(inner) {
			continue
		}
		suffixes = append([]string{strings.TrimSpace(matches[2])}, suffixes...)
	}

	if len(suffixes) > 0 {
		cleaned += " " + strings.Join(suffixes, " ")
	}

	return cleaned
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func formatMatchMinute(minute string) string {
	cleaned := normalizeDisplayText(minute)
	if cleaned == "" {
		return ""
	}
	if strings.HasSuffix(cleaned, "'") {
		if ansi.StringWidth(cleaned) == 2 {
			return " " + cleaned
		}
		return cleaned
	}
	formatted := cleaned + "'"
	if ansi.StringWidth(formatted) == 2 {
		return " " + formatted
	}
	return formatted
}

func renderDividerLabel(label string, width int) string {
	cleaned := normalizeDisplayText(label)
	if cleaned == "" {
		cleaned = "─"
	}
	if width <= len([]rune(cleaned))+2 {
		return cleaned
	}

	pad := width - len([]rune(cleaned)) - 2
	left := pad / 2
	right := pad - left
	return strings.Repeat("─", left) + " " + cleaned + " " + strings.Repeat("─", right)
}

// Align the divider score dash with the event-minute column when width allows.
func renderMatchDividerRow(label string, width int) string {
	if width < 30 {
		return renderDividerLabel(label, width)
	}

	label = truncate(label, max(1, width-2))
	dashOffset := strings.Index(label, " – ") + 1
	if dashOffset < 1 {
		// Fall back to ASCII dash for labels that don't contain an en-dash.
		dashOffset = strings.Index(label, " - ") + 1
	}
	if dashOffset < 1 {
		return renderDividerLabel(label, width)
	}

	minuteAxis := max(0, (width-7)/2+3)
	leftWidth := max(0, minuteAxis-1-dashOffset)
	rightWidth := max(0, width-leftWidth-ansi.StringWidth(label)-2)

	return strings.Repeat("─", leftWidth) + " " + label + " " + strings.Repeat("─", rightWidth)
}

func matchStatus(page *site.MatchPage) string {
	if page == nil {
		return ""
	}

	text := strings.ToLower(normalizeDisplayText(page.Meta + " " + page.Title))
	switch {
	case strings.Contains(text, "odwo"):
		return "OFF"
	case strings.Contains(text, "przelo") || strings.Contains(text, "przeło"):
		return "PPD"
	case strings.Contains(text, "dogr"):
		return "AET"
	default:
		return ""
	}
}

type scorerLine struct {
	label     string
	minute    string
	side      string
	isDivider bool
}

// Return visible score-header events in minute order; insert HT when play continues after halftime.
func headerEventRows(events []site.MatchEvent) []scorerLine {
	ordered := sortedEvents(events)
	htLabel := halftimeScore(events)
	firstSecondHalfKey := 0
	hasSecondHalfEvent := false
	for _, event := range ordered {
		key, ok := minuteSortKey(event.MinuteText)
		if !ok || key <= 4599 {
			continue
		}
		firstSecondHalfKey = key
		hasSecondHalfEvent = true
		break
	}

	insertedHT := false
	lines := make([]scorerLine, 0, 8)

	for _, event := range ordered {
		key, ok := minuteSortKey(event.MinuteText)
		if !ok {
			continue
		}
		if !insertedHT && htLabel != "" && hasSecondHalfEvent && key >= firstSecondHalfKey {
			lines = append(lines, scorerLine{label: htLabel, isDivider: true})
			insertedHT = true
		}

		switch event.Kind {
		case "GOAL", "MISS", "RC":
		default:
			continue
		}
		if strings.TrimSpace(event.MinuteText) == "" {
			continue
		}

		name := trimEventMinute(event)
		switch event.Kind {
		case "GOAL":
			if name == "" {
				continue
			}
			lines = append(lines, scorerLine{
				label:  formatGoalLabel(name, event.TeamSide),
				minute: formatMatchMinute(event.MinuteText),
				side:   event.TeamSide,
			})
		case "MISS":
			var label string
			if event.TeamSide == "home" {
				label = formatLeftEventLabel("MISS", name)
			} else {
				label = formatRightEventLabel("MISS", name)
			}
			lines = append(lines, scorerLine{
				label:  label,
				minute: formatMatchMinute(event.MinuteText),
				side:   event.TeamSide,
			})
		case "RC":
			var label string
			if event.TeamSide == "home" {
				label = formatLeftEventLabel("RC", name)
			} else {
				label = formatRightEventLabel("RC", name)
			}
			lines = append(lines, scorerLine{
				label:  label,
				minute: formatMatchMinute(event.MinuteText),
				side:   event.TeamSide,
			})
		}
	}

	if !insertedHT && htLabel != "" && hasSecondHalfEvent {
		lines = append(lines, scorerLine{label: htLabel, isDivider: true})
	}

	return lines
}

// formatGoalLabel builds a goal side-label with the icon adjacent to the center column.
// "Name ⚽" (home, icon on right nearest center) or "⚽ Name" (away, icon on left nearest center).
func formatGoalLabel(name, side string) string {
	glyph := eventPrefix("GOAL")
	if side == "home" {
		return name + " " + glyph
	}
	return glyph + " " + name
}

func playerMatchKey(label string) string {
	formatted := normalizeDisplayText(canonicalPlayerName(label))
	if formatted == "" {
		return ""
	}
	return strings.ToLower(formatted)
}

// playerEventIndex maps normalized compact player labels to events for the given side.
// SUB events are indexed under both outgoing and incoming player names.
func playerEventIndex(events []site.MatchEvent, side string) map[string][]site.MatchEvent {
	idx := make(map[string][]site.MatchEvent)
	for _, e := range events {
		if e.TeamSide != side {
			continue
		}
		if e.Kind == "SUB" {
			out, in := substitutionPlayers(e.Text)
			if key := playerMatchKey(out); key != "" {
				idx[key] = append(idx[key], e)
			}
			if key := playerMatchKey(in); key != "" {
				idx[key] = append(idx[key], e)
			}
			continue
		}
		name := eventPlayerText(e)
		if key := playerMatchKey(name); key != "" {
			idx[key] = append(idx[key], e)
		}
	}
	return idx
}

// cardAnnotation returns the YC/RC badge string for a lineup player, intended
// for the dedicated event column next to the centre separator. Empty when clean.
func cardAnnotation(player site.PlayerLine, idx map[string][]site.MatchEvent) string {
	return cardAnnotationName(player.Name, idx)
}

func cardAnnotationName(name string, idx map[string][]site.MatchEvent) string {
	key := playerMatchKey(name)
	if key == "" {
		return ""
	}

	matched, ok := idx[key]
	if !ok {
		return ""
	}

	hasYellow := false
	for _, e := range matched {
		switch e.Kind {
		case "RC":
			return eventPrefix("RC")
		case "YC":
			hasYellow = true
		}
	}
	if hasYellow {
		return eventPrefix("YC")
	}
	return ""
}

type lineupEntry struct {
	player       site.PlayerLine
	enteredAt    string
	replaced     string
	replacedYC   string
	leftAt       string
	replacedBy   string
	replacedByYC string
}

// A lineup row can carry both entry and exit notes for players who came on and were later replaced.
func formatLineupPlayer(entry lineupEntry, side string, maxWidth int) string {
	name := formatPlayerLabel(entry.player.Name)
	if entry.enteredAt == "" && entry.replaced == "" && entry.leftAt == "" && entry.replacedBy == "" {
		return name
	}

	label := lineupPlayerLabel(entry, side, name, false)
	if maxWidth > 0 && ansi.StringWidth(label) > maxWidth {
		shortened := lineupPlayerLabel(entry, side, name, true)
		if ansi.StringWidth(shortened) < ansi.StringWidth(label) {
			return shortened
		}
	}

	return label
}

func lineupPlayerLabel(entry lineupEntry, side, name string, shortenNotes bool) string {
	notes := lineupNotes(entry, side, shortenNotes)
	if len(notes) == 0 {
		return name
	}

	if side == "home" {
		parts := append(notes, name)
		return strings.Join(parts, " ")
	}

	parts := append([]string{name}, notes...)
	return strings.Join(parts, " ")
}

func lineupNotes(entry lineupEntry, side string, shortenNotes bool) []string {
	notes := make([]string, 0, 2)

	if note := entryNote(entry, side, shortenNotes); note != "" {
		notes = append(notes, note)
	}
	if note := exitNote(entry, side, shortenNotes); note != "" {
		notes = append(notes, note)
	}

	return notes
}

func entryNote(entry lineupEntry, side string, shortenNotes bool) string {
	if entry.enteredAt == "" {
		return ""
	}

	replaced := formatSubNoteName(entry.replaced, shortenNotes)
	if side == "home" {
		text := "(" + entry.enteredAt
		if replaced != "" {
			text += " for " + replaced
			if entry.replacedYC != "" {
				text += entry.replacedYC
			}
		}
		return faintText(text + ")")
	}

	text := "(for "
	if replaced != "" {
		text += replaced + " "
	}
	if entry.replacedYC != "" {
		text += entry.replacedYC + " "
	}
	text += entry.enteredAt
	return faintText(text + ")")
}

func exitNote(entry lineupEntry, side string, shortenNotes bool) string {
	if entry.replacedBy == "" {
		return ""
	}

	replacement := formatSubNoteName(entry.replacedBy, shortenNotes)
	text := "("
	if side == "home" && entry.leftAt != "" {
		text += entry.leftAt + " "
	}
	text += replacement
	if entry.replacedByYC != "" {
		text += entry.replacedByYC
	}
	if side != "home" && entry.leftAt != "" {
		text += " " + entry.leftAt
	}
	return faintText(text + ")")
}

// Under width pressure, shorten only substitution-note names so the main player label stays stable.
func formatSubNoteName(name string, surnameOnly bool) string {
	formatted := formatPlayerLabel(name)
	if !surnameOnly {
		return formatted
	}

	cleaned := canonicalPlayerName(name)
	if cleaned == "" {
		return ""
	}

	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return ""
	}

	return faintPenaltySuffix(words[len(words)-1])
}

// Players can collect both entry and exit notes when they are substituted on and off in one match.
func annotateLineupPlayer(player site.PlayerLine, idx map[string][]site.MatchEvent) lineupEntry {
	entry := lineupEntry{player: player}
	key := playerMatchKey(player.Name)
	for _, event := range sortedEvents(idx[key]) {
		if event.Kind != "SUB" {
			continue
		}

		out, in := substitutionPlayers(event.Text)
		minute := strings.TrimSpace(formatMatchMinute(event.MinuteText))
		if playerMatchKey(in) == key {
			entry.enteredAt = minute
			entry.replaced = out
			entry.replacedYC = cardAnnotationName(out, idx)
		}
		if playerMatchKey(out) == key {
			entry.leftAt = minute
			entry.replacedBy = in
			entry.replacedByYC = cardAnnotationName(in, idx)
		}
	}

	return entry
}

// Synthetic entrant rows are only added when a substitute was later replaced again.
func annotatedLineup(players []site.PlayerLine, idx map[string][]site.MatchEvent) []lineupEntry {
	if len(players) == 0 {
		return nil
	}

	byKey := make(map[string]site.PlayerLine, len(players))
	for _, player := range players {
		byKey[playerMatchKey(player.Name)] = player
	}

	entries := make([]lineupEntry, 0, len(players))
	addedSynthetic := make(map[string]bool)
	for _, player := range players {
		entry := annotateLineupPlayer(player, idx)
		entries = append(entries, entry)

		inKey := playerMatchKey(entry.replacedBy)
		if inKey == "" || addedSynthetic[inKey] {
			continue
		}
		if _, exists := byKey[inKey]; exists {
			continue
		}

		synthetic := annotateLineupPlayer(site.PlayerLine{Name: entry.replacedBy}, idx)
		if synthetic.replacedBy == "" {
			continue
		}

		entries = append(entries, synthetic)
		addedSynthetic[inKey] = true
	}

	return entries
}

// Narrow layouts fall back to the generic lineup row, which reserves a smaller per-side text budget.
func lineupPlayerWidth(width int) int {
	if width <= 0 {
		return 0
	}
	if width < 30 {
		return 0
	}
	if width < 36 {
		return max(8, (width-3-2)/2)
	}

	const eventWidth = 1
	const gap = 0
	return max(8, (width-1-2*eventWidth-2*gap)/2)
}

// Keep a dedicated event column next to the centre separator:
//
//	[home name →right] [home events →right] | [away events ←left] [away name ←left]
func renderAnnotatedLineupRow(homePlayer, homeEvents, awayPlayer, awayEvents string, width int) string {
	if width < 36 {
		home := homePlayer
		if homeEvents != "" {
			home += " " + homeEvents
		}
		away := awayPlayer
		if awayEvents != "" {
			away = awayEvents + " " + away
		}
		return renderLineupRow(home, away, width)
	}

	const eventWidth = 1 // one char wide (YC/RC block or empty)
	const gap = 0        // names sit directly against the event column
	playerWidth := lineupPlayerWidth(width)

	leftPlayer := padLeft(truncate(homePlayer, playerWidth), playerWidth)
	leftEvents := padLeft(truncate(homeEvents, eventWidth), eventWidth)
	rightEvents := padRight(truncate(awayEvents, eventWidth), eventWidth)
	rightPlayer := truncate(awayPlayer, playerWidth)

	return leftPlayer + strings.Repeat(" ", gap) + leftEvents + "|" + rightEvents + strings.Repeat(" ", gap) + rightPlayer
}

func halftimeScore(events []site.MatchEvent) string {
	homeGoals := 0
	awayGoals := 0
	hasSecondHalf := false

	for _, event := range sortedEvents(events) {
		minute, ok := minuteSortKey(event.MinuteText)
		if !ok {
			continue
		}
		// minuteSortKey encodes stoppage as MM*100+extra, so 45:59 is the first-half ceiling.
		if minute <= 4599 {
			if event.Kind == "GOAL" {
				if event.TeamSide == "home" {
					homeGoals++
				} else if event.TeamSide == "away" {
					awayGoals++
				}
			}
			continue
		}
		hasSecondHalf = true
	}

	if !hasSecondHalf {
		return ""
	}

	return fmt.Sprintf("HT %d – %d", homeGoals, awayGoals)
}

func finalScoreLine(page *site.MatchPage) string {
	if page == nil {
		return ""
	}

	score := strings.TrimSpace(page.Score)
	if score == "" {
		return ""
	}

	return "FT " + dividerScore(score)
}

func dividerScore(score string) string {
	trimmed := strings.TrimSpace(score)
	if trimmed == "" {
		return "?-?"
	}

	parts := strings.SplitN(trimmed, "-", 2)
	if len(parts) != 2 {
		return normalizeScore(trimmed)
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if left == "" || right == "" {
		return normalizeScore(trimmed)
	}

	return left + " – " + right
}

func matchMetaParts(meta, weather string) []string {
	parts := make([]string, 0, 4)
	cleanedMeta := normalizeDisplayText(meta)
	if cleanedMeta != "" {
		datePart := matchDatePrefixRe.FindString(cleanedMeta)
		remainder := strings.TrimSpace(strings.TrimPrefix(cleanedMeta, datePart))
		if datePart != "" {
			parts = append(parts, translatePolishDateText(datePart))
		}
		if matches := leadingAttendanceRe.FindStringSubmatch(remainder); len(matches) == 2 {
			parts = append(parts, "Attendance "+strings.TrimSpace(matches[1]))
			remainder = strings.TrimSpace(strings.TrimPrefix(remainder, matches[0]))
		}
		if remainder != "" {
			translated := translatePolishDateText(remainder)
			// Drop trailing "(City)" — only venue/city name, not part of the ref's identity.
			refName := strings.TrimSpace(trailingParenRe.ReplaceAllString(translated, "$1"))
			parts = append(parts, "Ref. "+refName)
		}
		if len(parts) == 0 {
			parts = append(parts, translatePolishDateText(cleanedMeta))
		}
	}

	cleanedWeather := normalizeDisplayText(weather)
	if cleanedWeather != "" {
		parts = append(parts, "Weather "+cleanedWeather)
	}

	return parts
}

func renderSideBySide(left, middle, right string, width int) string {
	if width < 30 {
		if middle == "" {
			return left + " | " + right
		}
		return left + " | " + middle + " | " + right
	}

	midWidth := 9
	gap := 1
	sideWidth := max(8, (width-midWidth-(gap*2))/2)

	leftText := padRight(truncate(left, sideWidth), sideWidth)
	midText := padCenter(truncate(middle, midWidth), midWidth)
	rightText := truncate(right, sideWidth)

	return leftText + strings.Repeat(" ", gap) + midText + strings.Repeat(" ", gap) + rightText
}

func renderMatchDetailRow(left, middle, right string, width int) string {
	if width < 30 {
		return renderSideBySide(left, middle, right, width)
	}

	// Keep the minute column visually centered and close to the HT/FT score dash,
	// even as left/right event labels vary in width.
	midWidth := 7
	gap := 0
	sideWidth := max(8, (width-midWidth-(gap*2))/2)

	leftText := padLeft(truncate(left, sideWidth), sideWidth)
	midText := padCenter(truncate(middle, midWidth), midWidth)
	rightText := truncate(right, sideWidth)

	return leftText + strings.Repeat(" ", gap) + midText + strings.Repeat(" ", gap) + rightText
}

func renderLineupRowWithMarker(left, right, marker string, width int) string {
	if width < 30 {
		return renderSideBySide(left, marker, right, width)
	}

	midWidth := 3
	gap := 1
	sideWidth := max(8, (width-midWidth-(gap*2))/2)

	leftText := padLeft(truncate(left, sideWidth), sideWidth)
	midText := padCenter(truncate(marker, midWidth), midWidth)
	rightText := truncate(right, sideWidth)

	return leftText + strings.Repeat(" ", gap) + midText + strings.Repeat(" ", gap) + rightText
}

func renderLineupRow(left, right string, width int) string {
	return renderLineupRowWithMarker(left, right, "|", width)
}

func renderCenteredText(text string, width int) string {
	if width <= 0 {
		return text
	}
	return padCenter(truncate(text, width), width)
}

func truncate(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= maxLen {
		return value
	}
	if maxLen == 1 {
		return "…"
	}

	return ansi.Cut(value, 0, maxLen-1) + "…"
}

func padRight(value string, width int) string {
	pad := width - ansi.StringWidth(value)
	if pad <= 0 {
		return value
	}

	return value + strings.Repeat(" ", pad)
}

func padLeft(value string, width int) string {
	pad := width - ansi.StringWidth(value)
	if pad <= 0 {
		return value
	}

	return strings.Repeat(" ", pad) + value
}

func padCenter(value string, width int) string {
	pad := width - ansi.StringWidth(value)
	if pad <= 0 {
		return value
	}

	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
}

func layoutWidths(total int, collapsed, emphasizeRight bool) (int, int) {
	if total < 40 {
		return 0, total
	}

	if collapsed {
		return 0, total
	}

	leftWidth := 36
	if emphasizeRight {
		leftWidth = clamp(total/4, 28, 42)
	} else {
		leftWidth = clamp(total/3, 32, 50)
	}

	rightWidth := total - leftWidth - 1
	if rightWidth < 40 {
		rightWidth = 40
		leftWidth = max(0, total-rightWidth-1)
	}

	if leftWidth < 24 {
		leftWidth = 0
		rightWidth = total
	}

	return leftWidth, rightWidth
}

func leagueLayoutWidths(total int) (int, int) {
	if total < 88 {
		return 0, total
	}

	leftWidth := clamp(total/3+8, 42, 58)
	rightWidth := total - leftWidth - 1
	if rightWidth < 36 {
		rightWidth = 36
		leftWidth = max(0, total-rightWidth-1)
	}

	return leftWidth, rightWidth
}

func matchLayoutWidths(total int) (int, int, int) {
	if total < 72 {
		return 0, total, 0
	}

	leftWidth := clamp(total/3+4, 38, 54)
	centerWidth := total - leftWidth - 1
	if centerWidth >= 40 {
		return leftWidth, centerWidth, 0
	}

	return 0, total, 0
}

func abbreviateTeamName(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "---"
	}

	var letters []rune
	for _, r := range []rune(clean) {
		if unicode.IsLetter(r) {
			letters = append(letters, unicode.ToUpper(r))
		}
		if len(letters) == 3 {
			break
		}
	}

	if len(letters) == 0 {
		letters = []rune(strings.ToUpper(clean))
	}
	if len(letters) >= 3 {
		return string(letters[:3])
	}

	return padRight(string(letters), 3)
}

func abbreviatedFixtureLine(fixture *site.Fixture, scoreWidth int) string {
	if fixture == nil {
		return "--- " + padCenter("?-?", scoreWidth) + " ---"
	}

	score := padCenter(normalizeScore(fixture.Score), scoreWidth)
	return fmt.Sprintf("%s %s %s", abbreviateTeamName(fixture.Home), score, abbreviateTeamName(fixture.Away))
}

func fixtureLine(fixture *site.Fixture, width, whenWidth, scoreWidth int, compact bool) string {
	if compact {
		return abbreviatedFixtureLine(fixture, scoreWidth)
	}
	if fixture == nil {
		return "--- ?-? ---"
	}
	if width <= 0 {
		return fmt.Sprintf("%s %s %s", fixture.Home, normalizeScore(fixture.Score), fixture.Away)
	}

	score := padCenter(normalizeScore(fixture.Score), scoreWidth)
	reserved := scoreWidth + 2
	if whenWidth > 0 {
		reserved += 3 + whenWidth
	}
	nameWidth := max(12, (width-reserved-1)/2)
	home := padLeft(truncate(fixture.Home, nameWidth), nameWidth)
	away := padRight(truncate(fixture.Away, nameWidth), nameWidth)
	return home + " " + score + " " + away
}

func fixtureAvailabilitySuffix(fixture *site.Fixture, width int, compact bool) string {
	if fixture == nil || strings.TrimSpace(fixture.MatchURL) != "" {
		return ""
	}

	suffix := " [no details]"
	minWidth := 64
	if compact {
		minWidth = 40
	}
	if width < minWidth+ansi.StringWidth(suffix) {
		return ""
	}

	return suffix
}

func normalizeScore(score string) string {
	trimmed := strings.TrimSpace(score)
	if trimmed == "" {
		return "?-?"
	}
	return strings.ReplaceAll(trimmed, "-", "-")
}

func formatFetchTime(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}
	return ts.Format("15:04:05")
}

func standingsTeamWidth(rows []site.StandingRow, width int) int {
	const reserved = 21
	minWidth := ansi.StringWidth("Team")
	maxWidth := max(minWidth, width-reserved)
	if maxWidth <= minWidth {
		return minWidth
	}

	teamWidth := minWidth
	for _, row := range rows {
		teamWidth = max(teamWidth, ansi.StringWidth(row.Team))
	}
	if teamWidth > maxWidth {
		return maxWidth
	}
	return teamWidth
}

func formatStandingRow(row site.StandingRow, selected bool, teamWidth, width int) string {
	line := fmt.Sprintf("  %2d %-*s %2d %2d %2d %2d %3d", row.Position, teamWidth, truncate(row.Team, teamWidth), row.Played, row.Won, row.Drawn, row.Lost, row.Points)
	line = truncate(line, max(12, width))
	if selected {
		return styleBold.Render(line)
	}
	return line
}

func parseRoundNumber(name string, fallback int) string {
	for _, field := range strings.Fields(name) {
		if _, err := strconv.Atoi(field); err == nil {
			return field
		}
		trimmed := strings.TrimRight(field, ".")
		if _, err := strconv.Atoi(trimmed); err == nil {
			return trimmed
		}
	}

	if fallback <= 0 {
		return "?"
	}
	return strconv.Itoa(fallback)
}

func displayCompetitionLabel(label string) string {
	cleaned := normalizeDisplayText(label)
	if cleaned == "" {
		return ""
	}

	if matches := roundSuffixRe.FindStringSubmatch(cleaned); len(matches) == 2 {
		base := strings.TrimSpace(roundSuffixRe.ReplaceAllString(cleaned, ""))
		if base != "" {
			return base + " - Round " + matches[1]
		}
	}

	return translatePolishDateText(cleaned)
}

func displayRoundLabel(name string, fallback int) string {
	cleaned := normalizeDisplayText(name)
	if cleaned == "" {
		return "Round " + parseRoundNumber("", fallback)
	}

	lower := strings.ToLower(cleaned)
	if lower == "wyniki" {
		return "Results"
	}
	if strings.Contains(lower, "fina") {
		return translatePolishStageLabel(cleaned)
	}
	if strings.Contains(lower, "bara") || strings.Contains(lower, "play-off") || strings.Contains(lower, "playoff") {
		return translatePolishStageLabel(cleaned)
	}

	if strings.Contains(lower, "kolejka") || strings.Contains(lower, "runda") {
		number := parseRoundNumber(cleaned, fallback)
		if idx := strings.Index(cleaned, "-"); idx >= 0 {
			suffix := translatePolishDateText(strings.TrimSpace(cleaned[idx+1:]))
			if suffix != "" {
				return "Round " + number + " - " + suffix
			}
		}
		return "Round " + number
	}

	return translatePolishDateText(cleaned)
}

func displayMatchMeta(meta, weather string) string {
	return strings.Join(matchMetaParts(meta, weather), " | ")
}

// matchMetaDisplay splits meta parts into a prominent date line and a secondary
// details line (attendance, ref, weather). Returns empty strings when absent.
func matchMetaDisplay(meta, weather string) (date, details string) {
	parts := matchMetaParts(meta, weather)
	if len(parts) == 0 {
		return "", ""
	}
	if matchDatePrefixRe.MatchString(parts[0]) {
		date = parts[0]
		if len(parts) > 1 {
			details = strings.Join(parts[1:], "  ·  ")
		}
		return
	}
	details = strings.Join(parts, "  ·  ")
	return
}

func trimEventMinute(event site.MatchEvent) string {
	text := eventPlayerText(event)
	return formatPlayerLabel(text)
}

func eventPlayerText(event site.MatchEvent) string {
	text := normalizeDisplayText(event.Text)
	if text == "" || event.MinuteText == "" {
		return text
	}

	minute := regexp.QuoteMeta(event.MinuteText)
	re := regexp.MustCompile(`^` + minute + `'?\s*(?:->\s*)?`)
	text = strings.TrimSpace(re.ReplaceAllString(text, ""))
	re = regexp.MustCompile(`\s+` + minute + `(\s*(?:\([^)]*\))?)$`)
	text = strings.TrimSpace(re.ReplaceAllString(text, `$1`))

	if event.Kind == "SUB" {
		text = strings.TrimSpace(strings.TrimPrefix(text, "->"))
		text = normalizeSubstitutionText(text)
	}
	if event.Kind == "GOAL" {
		text = strings.ReplaceAll(text, "(k)", "(pen)")
	}
	if event.Kind == "MISS" {
		text = strings.ReplaceAll(text, "(nk)", "(pen)")
	}

	return canonicalPlayerName(text)
}

func normalizeSubstitutionText(text string) string {
	cleaned := normalizeDisplayText(text)
	if cleaned == "" {
		return ""
	}
	return playerNumberSuffixRe.ReplaceAllString(cleaned, "")
}

func normalizeDisplayText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func translatePolishDateText(value string) string {
	return polishMonthReplacer.Replace(normalizeDisplayText(value))
}

func formatFixtureWhenInfo(value string) string {
	cleaned := normalizeDisplayText(value)
	if cleaned == "" {
		return ""
	}

	if matches := fixtureWhenInfoRe.FindStringSubmatch(cleaned); len(matches) == 4 {
		if month := polishMonthNumber(matches[2]); month != "" {
			return fmt.Sprintf("%02s/%s %s", matches[1], month, matches[3])
		}
	}

	if idx := strings.Index(cleaned, "("); idx > 0 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}

	return translatePolishDateText(cleaned)
}

func polishMonthNumber(value string) string {
	switch strings.ToLower(normalizeDisplayText(value)) {
	case "stycznia":
		return "01"
	case "lutego":
		return "02"
	case "marca":
		return "03"
	case "kwietnia":
		return "04"
	case "maja":
		return "05"
	case "czerwca":
		return "06"
	case "lipca":
		return "07"
	case "sierpnia":
		return "08"
	case "wrzesnia", "września":
		return "09"
	case "pazdziernika", "października":
		return "10"
	case "listopada":
		return "11"
	case "grudnia":
		return "12"
	default:
		return ""
	}
}

func translatePolishStageLabel(value string) string {
	cleaned := translatePolishDateText(value)
	replacements := []struct{ old, new string }{
		{"1/16 finału", "Round of 32"},
		{"1/8 finału", "Round of 16"},
		{"1/4 finału", "Quarter-finals"},
		{"1/2 finału", "Semi-finals"},
		{"finał", "Final"},
		{"baraże", "Play-offs"},
		{"baraż", "Play-off"},
	}
	for _, replacement := range replacements {
		cleaned = strings.ReplaceAll(cleaned, replacement.old, replacement.new)
	}

	return cleaned
}
