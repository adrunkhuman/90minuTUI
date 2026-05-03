package ui

import (
	"fmt"
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) leagueSketchView() string {
	leftWidth, rightWidth := leagueLayoutWidths(m.width)
	fixtures := m.leagueFixturesPaneView(rightWidth)
	if leftWidth == 0 {
		return fixtures
	}

	standings := m.standingsPaneViewBounded(leftWidth)
	divider := m.verticalDivider()
	return lipgloss.JoinHorizontal(lipgloss.Top, standings, divider, fixtures)
}

func (m Model) verticalDivider() string {
	h := m.bodyHeightLimit()
	if h <= 0 {
		return lipgloss.NewStyle().Foreground(colorBorder).Background(colorBg).Render("│")
	}
	parts := make([]string, h)
	for i := range parts {
		parts[i] = lipgloss.NewStyle().Foreground(colorBorder).Background(colorBg).Render("│")
	}
	return strings.Join(parts, "\n")
}

func (m Model) standingsPaneViewBounded(width int) string {
	base := lipgloss.NewStyle().Width(width).Background(colorBgPane)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.Height(limit).MaxHeight(limit)
	}

	var b strings.Builder
	b.WriteString(paneHeader("STANDINGS", width))
	b.WriteString("\n")

	if m.league == nil || len(m.league.Standings) == 0 {
		b.WriteString(renderPlainPaneLine(" no standings", width, colorBgPane))
		return base.Render(b.String())
	}

	teamWidth := standingsTeamWidth(m.league.Standings, width-2)
	b.WriteString(formatStandingsColumns(teamWidth, width))
	b.WriteString("\n")
	for _, line := range renderPitchStandingsWindow(m.league.Standings, m.currentFixture(), width, m.standingsRowLimit()) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func paneHeader(label string, width int) string {
	text := " " + strings.ToUpper(label)
	return renderFullLine(text, width, colorBgHeader, colorAccent, true)
}

func renderPlainPaneLine(line string, width int, bg lipgloss.Color) string {
	return renderFullLine(ansi.Strip(line), width, bg, colorTextSecondary, false)
}

func formatStandingsColumns(teamWidth, width int) string {
	line := fmt.Sprintf("  %-3s %-*s %2s %2s %2s %2s %3s", "#", teamWidth, "Team", "P", "W", "D", "L", "Pts")
	return renderFullLine(line, width, colorBgPane, colorTextMuted, false)
}

func renderFullLine(text string, width int, bg lipgloss.Color, fg lipgloss.Color, bold bool) string {
	if width <= 0 {
		return ""
	}
	line := padRight(truncate(text, width), width)
	return lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(bold).Render(line)
}

func renderPitchStandingsWindow(rows []site.StandingRow, fixture *site.Fixture, width, maxItems int) []string {
	if len(rows) == 0 || maxItems <= 0 {
		return nil
	}

	start, end := anchoredWindowBounds(len(rows), standingSelectionIndices(rows, fixture), maxItems)
	teamWidth := standingsTeamWidth(rows, width-2)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		selected := fixture != nil && (strings.EqualFold(rows[i].Team, fixture.Home) || strings.EqualFold(rows[i].Team, fixture.Away))
		relegated := isRelegationRow(rows[i], len(rows))
		rowBg := colorRowOdd
		if i%2 == 1 {
			rowBg = colorRowEven
		}
		if selected {
			rowBg = colorBgSelected
		}
		lines = append(lines, formatPitchStandingRow(rows[i], selected, relegated, teamWidth, width, rowBg))
	}

	return lines
}

func isRelegationRow(row site.StandingRow, total int) bool {
	return total >= 6 && row.Position > total-3
}

func formatPitchStandingRow(row site.StandingRow, selected, relegated bool, teamWidth, width int, bg lipgloss.Color) string {
	fg := colorTextPrimary
	if relegated {
		fg = colorLoss
	}
	if selected {
		fg = colorAccent
	}

	bar := "  "
	if selected {
		bar = "▌ "
	}
	line := fmt.Sprintf("%s%-3d %-*s %2d %2d %2d %2d %3d", bar, row.Position, teamWidth, truncate(row.Team, teamWidth), row.Played, row.Won, row.Drawn, row.Lost, row.Points)
	return renderFullLine(line, width, bg, fg, selected)
}

func (m Model) leagueFixturesPaneView(width int) string {
	return m.leagueFixturesPaneViewStyled(width, false)
}

func (m Model) leagueFixturesPaneViewStyled(width int, dim bool) string {
	base := lipgloss.NewStyle().Width(width).Background(colorBgPanel)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.Height(limit).MaxHeight(limit)
	}

	round := m.currentRound()
	if m.league == nil || round == nil {
		return base.Render(renderPlainPaneLine(styleSubtle.Render("no league loaded"), width, colorBgPanel))
	}

	var b strings.Builder
	for _, line := range renderFixtureGridWindow(round.Fixtures, m.fixtureCursor, m.fixtureRowLimit(), width, m.league.Title, dim) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString("\n")
		b.WriteString(renderPlainPaneLine(styleSubtle.Render("Loading…"), width, colorBgPanel))
	}
	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(renderPlainPaneLine(styleSubtle.Render(m.err), width, colorBgPanel))
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func renderFixtureGridWindow(fixtures []site.Fixture, cursor, maxItems, width int, leagueTitle string, dim bool) []string {
	if len(fixtures) == 0 || maxItems <= 0 {
		return nil
	}

	start, end := windowBounds(len(fixtures), cursor, maxItems)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rowBg := colorBgPanel
		if i%2 == 1 {
			rowBg = colorBg
		}
		selected := i == cursor
		if selected {
			rowBg = colorBgSelected
		}
		lines = append(lines, formatFixtureGridRow(&fixtures[i], selected, width, leagueTitle, dim, rowBg))
	}

	return lines
}

func formatFixtureGridRow(fixture *site.Fixture, selected bool, width int, leagueTitle string, dim bool, bg lipgloss.Color) string {
	if fixture == nil {
		return renderFullLine("", width, bg, colorTextMuted, false)
	}

	scoreWidth := 7
	gap := 2
	detailMarker := strings.TrimSpace(fixtureAvailabilitySuffix(fixture, width, false))
	contentWidth := max(1, width-2)
	dateWidth := clamp(ansi.StringWidth("Thu 01/05 20:30"), 10, max(10, contentWidth/3))
	markerDateWidth := ansi.StringWidth("[no details]") + gap + ansi.StringWidth("Thu 01/05 20:30")
	if expandedDateWidth := max(dateWidth, markerDateWidth); (contentWidth-expandedDateWidth-scoreWidth-gap*3)/2 >= 12 {
		dateWidth = expandedDateWidth
	}
	teamWidth := (contentWidth - dateWidth - scoreWidth - gap*3) / 2
	if teamWidth < 6 {
		dateWidth = 0
		teamWidth = max(4, (contentWidth-scoreWidth-gap*2)/2)
	}

	rowColor := colorTextSecondary
	if selected {
		rowColor = colorAccent
	}
	if dim {
		rowColor = colorTextMuted
	}
	bar := "  "
	if selected {
		bar = "▌ "
	}

	home := padLeft(truncate(fixture.Home, teamWidth), teamWidth)
	score := padCenter(displayFixtureScore(fixture.Score), scoreWidth)
	away := padRight(truncate(fixture.Away, teamWidth), teamWidth)
	date := ""
	if dateWidth > 0 {
		dateText := truncate(formatFixtureDateTime(fixture.WhenInfo, leagueTitle), dateWidth)
		if detailMarker != "" && ansi.StringWidth(detailMarker)+gap+ansi.StringWidth(dateText) <= dateWidth {
			dateText = detailMarker + strings.Repeat(" ", gap) + dateText
		}
		date = padLeft(dateText, dateWidth)
	}

	line := bar + home + strings.Repeat(" ", gap) + score + strings.Repeat(" ", gap) + away
	if dateWidth > 0 {
		line += strings.Repeat(" ", gap) + date
	}
	return renderFullLine(line, width, bg, rowColor, selected)
}

func displayFixtureScore(score string) string {
	cleaned := strings.TrimSpace(score)
	if cleaned == "" || cleaned == "-" || cleaned == "–" {
		return "–"
	}
	return strings.ReplaceAll(cleaned, " - ", "-")
}
