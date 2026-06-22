package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() tea.View {
	view := tea.NewView(m.viewContent())
	view.AltScreen = true
	return view
}

func (m Model) viewContent() string {
	if m.width == 0 {
		return "Loading..."
	}

	topBar := m.topBarView()
	body := m.leagueSketchView()
	if m.matchView {
		body = m.matchSketchView()
	}
	if m.selectorActive() {
		body = m.selectorSketchView()
	}
	body = fitLines(body, m.bodyHeightLimit())

	if topBar != "" {
		return topBar + "\n" + body + "\n" + m.statusBarView()
	}
	return body + "\n" + m.statusBarView()
}

func (m Model) bodyHeightLimit() int {
	reserved := 1 // status bar
	if bar := m.topBarView(); bar != "" {
		reserved += strings.Count(bar, "\n") + 1
	}
	if m.height <= reserved {
		return 0
	}
	return m.height - reserved
}

func (m Model) topBarView() string {
	if m.suppressTopBar || m.league == nil {
		return ""
	}

	label := displayCompetitionLabel(m.league.Title)
	if label == "" {
		return ""
	}

	right := ""
	if round := m.currentRound(); round != nil {
		right = topBarRoundMeta(*round, m.roundCursor+1, len(m.league.Rounds), m.league.Title)
	}

	return renderFullLine(barLine(label, right, m.width), m.width, colorBgHeader, colorTextPrimary, true)
}

func barLine(left, right string, width int) string {
	inner := width - 2
	if inner <= 0 {
		return ""
	}
	if right == "" {
		text := truncate(left, inner)
		pad := max(0, inner-ansi.StringWidth(text))
		return " " + text + strings.Repeat(" ", pad) + " "
	}
	rightW := ansi.StringWidth(right)
	leftMax := max(0, inner-rightW-1)
	leftText := truncate(left, leftMax)
	leftW := ansi.StringWidth(leftText)
	gap := max(1, inner-leftW-rightW)
	return " " + leftText + strings.Repeat(" ", gap) + right + " "
}

func topBarRoundMeta(round site.Round, roundIdx, total int, leagueTitle string) string {
	label := displayRoundLabelForRound(round, roundIdx, leagueTitle)
	roundPart, datePart, ok := strings.Cut(label, " - ")
	if !ok || strings.TrimSpace(datePart) == "" {
		return fmt.Sprintf("%s / %d", label, total)
	}
	return fmt.Sprintf("%s · %s / %d", roundPart, strings.TrimSpace(datePart), total)
}

func displayRoundLabelForRound(round site.Round, fallback int, leagueTitle string) string {
	label := displayRoundLabelWithFixtures(round.Name, fallback, round.Fixtures, leagueTitle)
	section := displayRoundSection(round.Section)
	if section == "" {
		return label
	}
	return section + " · " + label
}

func displayRoundSection(section string) string {
	cleaned := normalizeDisplayText(section)
	if cleaned == "" {
		return ""
	}
	return translatePolishDateText(cleaned)
}

func (m Model) statusBarView() string {
	var parts []statusBarItem

	switch {
	case m.selectorActive():
		parts = []statusBarItem{{"j/k", "move"}, {"h/l", "pane"}, {"enter", "load"}}
		if len(m.competitionStack) > 0 {
			parts[2] = statusBarItem{"enter", "open"}
			parts = append(parts, statusBarItem{"esc", "back"})
		} else if m.league != nil {
			parts = append(parts, statusBarItem{"esc", "close"})
		}
		parts = append(parts, statusBarItem{"q", "quit"})

	case m.matchView:
		parts = []statusBarItem{{"j/k", "fixture"}, {"h/l", "round"}, {"pgup/pgdn", "scroll"}, {"esc", "back"}, {"r", "reload"}, {"q", "quit"}}

	default:
		enterHint := "details"
		if !m.currentFixtureDrillable() {
			enterHint = "unavail"
		}
		parts = []statusBarItem{{"j/k", "move"}, {"h/l", "round"}, {"enter", enterHint}, {"esc", "selector"}, {"r", "reload"}, {"q", "quit"}}
	}

	left := renderStatusItems(parts)

	right := formatFetchTime(m.lastFetchAt)
	if m.loading {
		right = "loading…"
	}
	if m.err != "" && !m.loading {
		right = "error  " + right
	}

	inner := m.width - 2
	leftW := ansi.StringWidth(left)
	rightW := ansi.StringWidth(right)
	gap := max(2, inner-leftW-rightW)
	if leftW+gap+rightW > inner {
		left = ansi.Cut(left, 0, max(0, inner-rightW-2))
		leftW = ansi.StringWidth(left)
		gap = max(1, inner-leftW-rightW)
	}

	status := statusSpace(1) + left + statusSpace(gap) + statusText(right) + statusSpace(1)
	return status + statusSpace(max(0, m.width-ansi.StringWidth(status)))
}

func renderStatusItems(items []statusBarItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, renderStatusKey(item.key)+statusSpace(1)+statusText(item.label))
	}
	return strings.Join(parts, statusSpace(2))
}

func renderStatusKey(key string) string {
	return lipgloss.NewStyle().Foreground(colorAccent).Background(colorBgSelected).Bold(true).Render(" " + key + " ")
}

func statusText(text string) string {
	return styleDim.Copy().Background(colorStatusBar).Render(text)
}

func statusSpace(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Background(colorStatusBar).Render(strings.Repeat(" ", width))
}

func selectorPopupWidth(total int, seasonLines []string, rightHeading string, competitionLines []string) int {
	if total <= 0 {
		return 36
	}

	leftWidth, rightWidth := selectorContentWidths(seasonLines, rightHeading, competitionLines)
	desired := max(64, leftWidth+1+rightWidth+2) // divider + border
	maxWidth := max(1, total-2)
	return clamp(desired, min(40, maxWidth), maxWidth)
}

func selectorContentWidths(seasonLines []string, rightHeading string, competitionLines []string) (int, int) {
	seasonWidth := lipgloss.Width("SEASON")
	for _, line := range seasonLines {
		seasonWidth = max(seasonWidth, lipgloss.Width(line))
	}
	leftWidth := clamp(seasonWidth+4, 18, 18)

	rightWidth := lipgloss.Width(rightHeading)
	for _, line := range competitionLines {
		rightWidth = max(rightWidth, lipgloss.Width(line))
	}
	rightWidth = max(16, rightWidth+1)

	return leftWidth, rightWidth
}

func selectorCompetitionWidthLines(items []site.Competition) []string {
	if len(items) == 0 {
		return nil
	}

	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, normalizeDisplayText(item.Name))
	}
	return lines
}

// matchDetailScore formats the score for the match header with an en-dash,
// giving it more visual weight than the compact fixture list format.
func matchDetailScore(score string) string {
	trimmed := strings.TrimSpace(score)
	if !hasFinalScore(trimmed) {
		return "vs"
	}
	parts := strings.SplitN(trimmed, "-", 2)
	return strings.TrimSpace(parts[0]) + " – " + strings.TrimSpace(parts[1])
}

func hasFinalScore(score string) bool {
	return fixtureResultRe.MatchString(strings.TrimSpace(score))
}

func (m Model) matchViewportHeight() int {
	return m.bodyHeightLimit()
}

func (m Model) matchDetailWidth() int {
	_, centerWidth, _ := matchLayoutWidths(m.width)
	return centerWidth
}

func (m Model) matchScrollLimit() int {
	if !m.matchView || m.match == nil {
		return 0
	}
	viewport := m.matchViewportHeight()
	if viewport <= 0 {
		return 0
	}
	lines := strings.Split(m.matchDetailContent(m.matchDetailWidth()), "\n")
	return max(0, len(lines)-viewport)
}

func clipLines(content string, offset, height int) string {
	if height <= 0 || content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= height {
		return content
	}
	start := clamp(offset, 0, len(lines)-height)
	end := min(len(lines), start+height)
	return strings.Join(lines[start:end], "\n")
}

func fitLines(content string, height int) string {
	if height <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
