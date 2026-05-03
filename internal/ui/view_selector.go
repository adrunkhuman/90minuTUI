package ui

import (
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) selectorSketchView() string {
	leftWidth, rightWidth := leagueLayoutWidths(m.width)
	if leftWidth == 0 {
		return m.selectorOverlayView(m.leagueFixturesPaneViewStyled(rightWidth, true))
	}

	standings := m.standingsPaneViewBounded(leftWidth)
	divider := m.verticalDivider()
	fixtures := m.leagueFixturesPaneViewStyled(rightWidth, true)
	body := lipgloss.JoinHorizontal(lipgloss.Top, standings, divider, fixtures)
	bodyLines := strings.Split(body, "\n")
	totalHeight := max(len(bodyLines), max(1, m.bodyHeightLimit()))
	for len(bodyLines) < totalHeight {
		bodyLines = append(bodyLines, "")
	}

	seasonLines := renderSeasonsWindow(m.seasons, m.seasonCursor)
	competitionLines := selectorCompetitionWidthLines(m.competitions)
	rightHeading := selectorRightHeading(m.competitionTitle)
	popup := m.selectorPopupView(selectorPopupWidth(rightWidth, seasonLines, rightHeading, competitionLines))
	popupLines := strings.Split(popup, "\n")
	top := 1
	if totalHeight > len(popupLines)+2 {
		top = (totalHeight - len(popupLines)) / 2
	}
	rightStart := leftWidth + 1
	left := rightStart + max(0, (rightWidth-lipgloss.Width(popup))/2)

	for i, line := range popupLines {
		if top+i >= len(bodyLines) {
			break
		}
		bodyLines[top+i] = overlayLine(bodyLines[top+i], line, left)
	}

	return strings.Join(bodyLines, "\n")
}

func (m Model) selectorOverlayView(body string) string {
	seasonLines := renderSeasonsWindow(m.seasons, m.seasonCursor)
	rightHeading := "Leagues"
	if strings.TrimSpace(m.competitionTitle) != "" {
		rightHeading = m.competitionTitle
	}
	competitionLines := selectorCompetitionWidthLines(m.competitions)
	popup := m.selectorPopupView(selectorPopupWidth(m.width, seasonLines, rightHeading, competitionLines))
	bodyLines := strings.Split(body, "\n")
	totalHeight := max(len(bodyLines), max(1, m.bodyHeightLimit()))
	for len(bodyLines) < totalHeight {
		bodyLines = append(bodyLines, "")
	}

	popupLines := strings.Split(popup, "\n")
	top := 1
	if totalHeight > len(popupLines)+2 {
		top = (totalHeight - len(popupLines)) / 2
	}
	left := max(0, (m.width-lipgloss.Width(popup))/2)
	for len(bodyLines) < top+len(popupLines) {
		bodyLines = append(bodyLines, "")
	}

	for i, line := range popupLines {
		bodyLines[top+i] = overlayLine(bodyLines[top+i], line, left)
	}

	return strings.Join(bodyLines, "\n")
}

func overlayLine(base, overlay string, leftWidth int) string {
	overlayWidth := ansi.StringWidth(overlay)
	if overlayWidth == 0 {
		return base
	}

	baseWidth := ansi.StringWidth(base)
	if baseWidth < leftWidth {
		base += strings.Repeat(" ", leftWidth-baseWidth)
		baseWidth = leftWidth
	}

	prefix := ansi.Cut(base, 0, leftWidth)
	suffix := ""
	if rightStart := leftWidth + overlayWidth; rightStart < baseWidth {
		suffix = ansi.Cut(base, rightStart, baseWidth)
	}

	return prefix + overlay + suffix
}

func (m Model) selectorPopupView(width int) string {
	innerWidth := max(24, width-2)
	leftWidth, rightWidth := selectorPaneWidths(innerWidth, renderSeasonsWindow(m.seasons, m.seasonCursor))
	seasonRows := selectorSeasonRows(m.seasons, m.seasonCursor)
	competitionRows := selectorCompetitionRows(m.competitions, m.competitionCursor)
	visibleRows := max(len(seasonRows), len(competitionRows))
	seasonStart, _ := windowBounds(len(m.seasons), m.seasonCursor, 10)
	competitionStart, _ := windowBounds(len(m.competitions), m.competitionCursor, 18)
	left := selectorModalPane(leftWidth, "SEASON", seasonRows, m.seasonCursor-seasonStart, visibleRows, m.focus == focusSeasons)
	right := selectorModalPane(rightWidth, selectorRightHeading(m.competitionTitle), competitionRows, m.competitionCursor-competitionStart, visibleRows, m.focus == focusCompetitions)
	divider := selectorModalDivider(max(strings.Count(left, "\n"), strings.Count(right, "\n")) + 1)

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)

	if m.loading {
		content += "\n" + renderPlainPaneLine(styleSubtle.Render("Loading…"), innerWidth, colorBgModal)
	}
	if m.err != "" {
		content += "\n" + renderPlainPaneLine(styleSubtle.Render("Error: "+m.err), innerWidth, colorBgModal)
	}

	panel := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorBorderStrong).
		Background(colorBgModal)

	return panel.Render(strings.TrimRight(content, "\n"))
}

func selectorRightHeading(title string) string {
	cleaned := strings.TrimSpace(title)
	if cleaned == "" || strings.EqualFold(cleaned, "Competitions") {
		return "COMPETITIONS"
	}
	return strings.ToUpper(cleaned)
}

func selectorSeasonRows(seasons []site.Season, cursor int) []string {
	if len(seasons) == 0 {
		return []string{"(none)"}
	}
	start, end := windowBounds(len(seasons), cursor, 10)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, seasons[i].Label)
	}
	return rows
}

func selectorCompetitionRows(items []site.Competition, cursor int) []string {
	if len(items) == 0 {
		return []string{"(none)"}
	}
	start, end := windowBounds(len(items), cursor, 18)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, items[i].Name)
	}
	return rows
}

func selectorModalPane(width int, heading string, rows []string, cursor, visibleRows int, focused bool) string {
	var b strings.Builder
	b.WriteString(renderFullLine(" "+truncate(heading, max(1, width-1)), width, colorBgHeader, colorAccent, true))
	for i := 0; i < visibleRows; i++ {
		b.WriteString("\n")
		row := ""
		if i < len(rows) {
			row = rows[i]
		}
		selected := focused && (i == cursor || (cursor >= len(rows) && i == len(rows)-1))
		b.WriteString(selectorModalRow(row, selected, focused, width))
	}
	return b.String()
}

func selectorModalRow(text string, selected, focused bool, width int) string {
	bg := colorBgModal
	fg := colorTextSecondary
	bar := "  "
	if selected {
		bg = colorBgSelected
		fg = colorAccent
		bar = "▌ "
	}
	return renderFullLine(bar+truncate(text, max(1, width-2)), width, bg, fg, selected)
}

func selectorModalDivider(height int) string {
	parts := make([]string, max(1, height))
	for i := range parts {
		parts[i] = lipgloss.NewStyle().Foreground(colorBorder).Background(colorBgModal).Render("│")
	}
	return strings.Join(parts, "\n")
}

func selectorPaneWidths(total int, seasonLines []string) (int, int) {
	seasonWidth := lipgloss.Width("SEASON")
	for _, line := range seasonLines {
		seasonWidth = max(seasonWidth, lipgloss.Width(line))
	}

	leftWidth := clamp(seasonWidth+4, 18, 18)
	rightWidth := total - leftWidth - 1
	if rightWidth < 16 {
		rightWidth = 16
		leftWidth = max(14, total-rightWidth-1)
	}

	return leftWidth, rightWidth
}
