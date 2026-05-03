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
var fixtureDateTimeRe = regexp.MustCompile(`(?i)(\d{1,2})\s+([\p{L}]+)(?:\s+(\d{4}))?,\s*(\d{1,2}:\d{2})`)
var leagueSeasonYearsRe = regexp.MustCompile(`(\d{4})\s*/\s*(\d{2,4})`)
var playerNumberPrefixRe = regexp.MustCompile(`^\(\d+\)\s*`)
var playerNumberSuffixRe = regexp.MustCompile(`\s+\(\d+\)$`)
var trailingParenRe = regexp.MustCompile(`^(.*?)(\s+\([^)]*\))$`)

const (
	// Round headers ignore at most three distant postponed fixtures when at least four fixtures form the main date cluster.
	roundDateMinCluster     = 4
	roundDateMaxOutliers    = 3
	roundDateOutlierGapDays = 7
)

// Minute keys encode stoppage as MM*100+extra; 45+99 is the first-half ceiling.
const firstHalfMinuteCeiling = 4599

func renderSeasonsWindow(seasons []site.Season, cursor int) []string {
	if len(seasons) == 0 {
		return []string{"(none)"}
	}

	start, end := windowBounds(len(seasons), cursor, 10)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, seasons[i].Label)
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

func atoiOrNeg(s string) int {
	value := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &value); err != nil {
		return -1
	}
	return value
}

func halftimeScore(events []site.MatchEvent) string {
	homeGoals := 0
	awayGoals := 0
	hasSecondHalf := false

	for _, event := range sortedEvents(events) {
		if !event.HasMinute {
			continue
		}
		minute := event.Minute*100 + event.Stoppage
		if minute <= firstHalfMinuteCeiling {
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

const leftPaneWidth = 54

func leagueLayoutWidths(total int) (int, int) {
	if total < 88 {
		return 0, total
	}

	leftWidth := leftPaneWidth
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

	leftWidth := leftPaneWidth
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

func displayRoundLabelWithFixtures(name string, fallback int, fixtures []site.Fixture, leagueTitle string) string {
	label := displayRoundLabel(name, fallback)
	span := roundFixtureDateSpan(fixtures, leagueTitle)
	if span == "" || !strings.HasPrefix(label, "Round ") {
		return label
	}

	roundPart, _, ok := strings.Cut(label, " - ")
	if !ok {
		roundPart = label
	}
	return roundPart + " - " + span
}

func roundFixtureDateSpan(fixtures []site.Fixture, leagueTitle string) string {
	dates := make([]time.Time, 0, len(fixtures))
	for _, fixture := range fixtures {
		date, ok := fixtureDisplayDate(fixture.WhenInfo, leagueTitle)
		if !ok {
			continue
		}
		dates = append(dates, date)
	}
	if len(dates) == 0 {
		return ""
	}

	dates = roundFixtureSpanDates(dates)
	first := dates[0]
	last := dates[len(dates)-1]
	if sameCalendarDate(first, last) {
		return first.Format("Jan 2")
	}
	if first.Year() == last.Year() && first.Month() == last.Month() {
		return fmt.Sprintf("%s-%d", first.Format("Jan 2"), last.Day())
	}
	return fmt.Sprintf("%s-%s", first.Format("Jan 2"), last.Format("Jan 2"))
}

func roundFixtureSpanDates(dates []time.Time) []time.Time {
	ordered := append([]time.Time(nil), dates...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Before(ordered[j]) })

	if len(ordered) <= roundDateMinCluster {
		return ordered
	}

	minKeep := max(roundDateMinCluster, len(ordered)-roundDateMaxOutliers)
	var best []time.Time
	var bestSpan time.Duration
	for keep := len(ordered) - 1; keep >= minKeep; keep-- {
		if window, ok := tightRoundDateWindow(ordered, keep); ok {
			span := window[len(window)-1].Sub(window[0])
			if best == nil || span < bestSpan || (span == bestSpan && len(window) > len(best)) {
				best = window
				bestSpan = span
			}
		}
	}
	if best != nil {
		return best
	}

	return ordered
}

func tightRoundDateWindow(dates []time.Time, keep int) ([]time.Time, bool) {
	var best []time.Time
	var bestSpan time.Duration
	for start := 0; start+keep <= len(dates); start++ {
		end := start + keep
		window := dates[start:end]
		if !hasOnlyDistantRoundDateOutliers(dates, start, end) {
			continue
		}

		span := window[len(window)-1].Sub(window[0])
		if best == nil || span < bestSpan {
			best = window
			bestSpan = span
		}
	}
	if best == nil {
		return nil, false
	}
	return append([]time.Time(nil), best...), true
}

func hasOnlyDistantRoundDateOutliers(dates []time.Time, start, end int) bool {
	gap := time.Duration(roundDateOutlierGapDays) * 24 * time.Hour
	for _, date := range dates[:start] {
		if dates[start].Sub(date) <= gap {
			return false
		}
	}
	for _, date := range dates[end:] {
		if date.Sub(dates[end-1]) <= gap {
			return false
		}
	}
	return true
}

func fixtureDisplayDate(value, leagueTitle string) (time.Time, bool) {
	cleaned := normalizeDisplayText(value)
	matches := fixtureDateTimeRe.FindStringSubmatch(cleaned)
	if len(matches) != 5 {
		return time.Time{}, false
	}

	month := polishMonthNumber(matches[2])
	if month == "" {
		return time.Time{}, false
	}

	day := atoiOrNeg(matches[1])
	monthNum := atoiOrNeg(month)
	year := atoiOrNeg(matches[3])
	if year < 0 {
		year = inferFixtureYear(monthNum, leagueTitle)
	}
	if day <= 0 || monthNum <= 0 || year <= 0 {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(monthNum), day, 0, 0, 0, 0, time.Local), true
}

func sameCalendarDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
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

func formatFixtureDateTime(value, leagueTitle string) string {
	cleaned := normalizeDisplayText(value)
	if cleaned == "" {
		return ""
	}

	matches := fixtureDateTimeRe.FindStringSubmatch(cleaned)
	if len(matches) != 5 {
		return formatFixtureWhenInfo(cleaned)
	}

	month := polishMonthNumber(matches[2])
	if month == "" {
		return formatFixtureWhenInfo(cleaned)
	}

	day := atoiOrNeg(matches[1])
	monthNum := atoiOrNeg(month)
	year := atoiOrNeg(matches[3])
	if year < 0 {
		year = inferFixtureYear(monthNum, leagueTitle)
	}
	if day <= 0 || monthNum <= 0 || year <= 0 {
		return fmt.Sprintf("%02d/%02d %s", max(0, day), max(0, monthNum), matches[4])
	}

	clock, err := time.Parse("15:04", matches[4])
	if err != nil {
		return fmt.Sprintf("%02d/%02d %s", day, monthNum, matches[4])
	}
	when := time.Date(year, time.Month(monthNum), day, clock.Hour(), clock.Minute(), 0, 0, time.Local)
	return when.Format("Mon 02/01 15:04")
}

func inferFixtureYear(month int, leagueTitle string) int {
	matches := leagueSeasonYearsRe.FindStringSubmatch(leagueTitle)
	if len(matches) != 3 {
		return 0
	}
	startYear := atoiOrNeg(matches[1])
	endYear := atoiOrNeg(matches[2])
	if endYear >= 0 && endYear < 100 {
		endYear += (startYear / 100) * 100
		if endYear < startYear {
			endYear += 100
		}
	}
	// 90minut often omits fixture years; Jul-Dec belongs to the season start, Jan-Jun to the season end.
	if month >= 7 {
		return startYear
	}
	return endYear
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
