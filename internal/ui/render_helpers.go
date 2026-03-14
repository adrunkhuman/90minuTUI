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
	for i := start; i < end; i++ {
		whenWidth = max(whenWidth, len([]rune(formatFixtureWhenInfo(fixtures[i].WhenInfo))))
	}
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		line := prefix + fixtureLine(&fixtures[i], width-len([]rune(prefix)), whenWidth, compact)
		if whenInfo := formatFixtureWhenInfo(fixtures[i].WhenInfo); whenInfo != "" {
			line += " | " + whenInfo
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
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		selected := fixture != nil && (strings.EqualFold(rows[i].Team, fixture.Home) || strings.EqualFold(rows[i].Team, fixture.Away))
		lines = append(lines, formatStandingRow(rows[i], selected, width))
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
	line := player.Name
	if len(player.Events) == 0 {
		return line
	}

	line += " [" + strings.Join(player.Events, ", ") + "]"
	return line
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
	case "RC":
		return 1
	case "YC":
		return 2
	case "SUB":
		return 3
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

func formatEventLabel(event site.MatchEvent) string {
	prefix := event.Kind
	if prefix == "" {
		prefix = "EVENT"
	}

	text := trimEventMinute(event)
	if text == "" {
		return prefix
	}

	return prefix + " " + text
}

func renderSideBySide(left, middle, right string, width int) string {
	if width < 30 {
		if middle == "" {
			return left + " | " + right
		}
		return left + " | " + middle + " | " + right
	}

	midWidth := 8
	gap := 2
	sideWidth := max(8, (width-midWidth-(gap*2))/2)

	leftText := padRight(truncate(left, sideWidth), sideWidth)
	midText := padRight(truncate(middle, midWidth), midWidth)
	rightText := truncate(right, sideWidth)

	return leftText + strings.Repeat(" ", gap) + midText + strings.Repeat(" ", gap) + rightText
}

func truncate(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	r := []rune(value)
	if len(r) <= maxLen {
		return value
	}
	if maxLen == 1 {
		return "…"
	}

	return string(r[:maxLen-1]) + "…"
}

func padRight(value string, width int) string {
	pad := width - len([]rune(value))
	if pad <= 0 {
		return value
	}

	return value + strings.Repeat(" ", pad)
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

	leftWidth := clamp(total/2+2, 46, 70)
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

func abbreviatedFixtureLine(fixture *site.Fixture) string {
	if fixture == nil {
		return "--- ?-? ---"
	}

	return fmt.Sprintf("%s %s %s", abbreviateTeamName(fixture.Home), normalizeScore(fixture.Score), abbreviateTeamName(fixture.Away))
}

func fixtureLine(fixture *site.Fixture, width, whenWidth int, compact bool) string {
	if compact {
		return abbreviatedFixtureLine(fixture)
	}
	if fixture == nil {
		return "--- ?-? ---"
	}
	if width <= 0 {
		return fmt.Sprintf("%s %s %s", fixture.Home, normalizeScore(fixture.Score), fixture.Away)
	}

	score := normalizeScore(fixture.Score)
	reserved := len([]rune(score)) + 2
	if whenWidth > 0 {
		reserved += 3 + whenWidth
	}
	nameWidth := max(12, (width-reserved-1)/2)
	home := padRight(truncate(fixture.Home, nameWidth), nameWidth)
	away := padRight(truncate(fixture.Away, nameWidth), nameWidth)
	return home + " " + score + " " + away
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

func formatStandingRow(row site.StandingRow, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	line := fmt.Sprintf("%s%2d %-18s %2d %2d %2d %2d %3d", prefix, row.Position, truncate(row.Team, 18), row.Played, row.Won, row.Drawn, row.Lost, row.Points)
	return truncate(line, max(12, width))
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
			parts = append(parts, "Ref. "+translatePolishDateText(remainder))
		}
		if len(parts) == 0 {
			parts = append(parts, translatePolishDateText(cleanedMeta))
		}
	}

	cleanedWeather := normalizeDisplayText(weather)
	if cleanedWeather != "" {
		parts = append(parts, "Weather "+cleanedWeather)
	}

	return strings.Join(parts, " | ")
}

func trimEventMinute(event site.MatchEvent) string {
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
	}

	return text
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
