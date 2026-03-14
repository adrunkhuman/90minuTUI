package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

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

func renderFixtureWindow(fixtures []site.Fixture, cursor, maxItems int) []string {
	if len(fixtures) == 0 {
		return nil
	}
	if maxItems <= 0 {
		return nil
	}

	start, end := windowBounds(len(fixtures), cursor, maxItems)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		line := prefix + abbreviatedFixtureLine(&fixtures[i])
		if fixtures[i].WhenInfo != "" {
			line += " | " + fixtures[i].WhenInfo
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

	if event.Text == "" {
		return prefix
	}

	return prefix + " " + event.Text
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

	leftWidth := clamp(total/2, 44, 66)
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

	leftWidth := clamp(total/3, 34, 48)
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
