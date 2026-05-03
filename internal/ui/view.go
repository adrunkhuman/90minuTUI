package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() string {
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
		right = topBarRoundMeta(round.Name, m.roundCursor+1, len(m.league.Rounds))
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

func topBarRoundMeta(roundName string, roundIdx, total int) string {
	label := displayRoundLabel(roundName, roundIdx)
	roundPart, datePart, ok := strings.Cut(label, " - ")
	if !ok || strings.TrimSpace(datePart) == "" {
		return fmt.Sprintf("%s / %d", label, total)
	}
	return fmt.Sprintf("%s · %s / %d", roundPart, strings.TrimSpace(datePart), total)
}

func (m Model) selectorSketchView() string {
	leftWidth, rightWidth := leagueLayoutWidths(m.width)
	if leftWidth == 0 {
		return m.selectorOverlayView(m.leagueFixturesPaneViewStyled(rightWidth, true))
	}

	standings := m.standingsPaneView(leftWidth)
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
	rightWidth := total - leftWidth - 1 // 1 for the │ divider
	if rightWidth < 16 {
		rightWidth = 16
		leftWidth = max(14, total-rightWidth-1)
	}

	return leftWidth, rightWidth
}

func (m Model) leagueSketchView() string {
	leftWidth, rightWidth := leagueLayoutWidths(m.width)
	fixtures := m.leagueFixturesPaneView(rightWidth)
	if leftWidth == 0 {
		return fixtures
	}

	standings := m.standingsPaneView(leftWidth)
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

func (m Model) standingsPaneView(width int) string {
	return m.standingsPaneViewBounded(width)
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

func (m Model) matchSketchView() string {
	leftWidth, centerWidth, _ := matchLayoutWidths(m.width)
	if leftWidth == 0 {
		return m.matchDetailPaneView(centerWidth)
	}

	sidebar := m.matchSidebarView(leftWidth)
	divider := m.verticalDivider()
	detail := m.matchDetailPaneView(centerWidth)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, divider, detail)
}

func (m Model) matchSidebarView(width int) string {
	standingsHeight, fixturesHeight := m.matchSidebarHeights()
	parts := make([]string, 0, 2)

	if standingsHeight > 0 {
		standingsModel := m
		standingsModel.height = standingsHeight + 1
		standingsModel.suppressTopBar = true
		parts = append(parts, standingsModel.standingsPaneViewBounded(width))
	}
	if fixturesHeight > 0 {
		fixturesModel := m
		fixturesModel.height = fixturesHeight + 1
		fixturesModel.suppressTopBar = true
		parts = append(parts, fixturesModel.matchFixtureRailView(width))
	}

	if len(parts) == 0 {
		return ""
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) matchSidebarHeights() (int, int) {
	total := m.bodyHeightLimit()
	if total <= 0 {
		return 0, 0
	}

	minFixtures := m.matchFixtureMinHeight()
	fullStandings := m.standingsContentHeight()
	if fullStandings > 0 && total >= fullStandings+minFixtures {
		return fullStandings, total - fullStandings
	}

	if total < 12 {
		return max(4, total/2), max(0, total-max(4, total/2))
	}

	standings := clamp(total/2, 8, 11)
	fixtures := total - standings
	if fixtures < minFixtures {
		fixtures = minFixtures
		standings = max(4, total-fixtures)
	}

	return standings, fixtures
}

func (m Model) matchFixtureRailView(width int) string {
	base := lipgloss.NewStyle().Width(width).Background(colorBgPane)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.Height(limit).MaxHeight(limit)
	}

	round := m.currentRound()
	if round == nil {
		return base.Render(renderPlainPaneLine(styleSubtle.Render("no fixtures"), width, colorBgPane))
	}

	var b strings.Builder
	b.WriteString(paneHeader("FIXTURES", width))
	b.WriteString("\n")
	for _, line := range renderRoundMiniGridWindow(round.Fixtures, m.fixtureCursor, m.matchFixtureRowLimit(), width, m.league.Title) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.loading && m.match == nil {
		b.WriteString("\n")
		b.WriteString(renderPlainPaneLine(styleSubtle.Render("Loading…"), width, colorBgPane))
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func renderRoundMiniGridWindow(fixtures []site.Fixture, cursor, maxLines, width int, leagueTitle string) []string {
	if len(fixtures) == 0 || maxLines <= 0 {
		return nil
	}
	if len(fixtures)*2 <= maxLines {
		lines := make([]string, 0, len(fixtures)*2)
		for i := range fixtures {
			top, bottom := formatRoundMiniCard(&fixtures[i], i == cursor, width, leagueTitle)
			lines = append(lines, top, bottom)
		}
		return lines
	}

	const cols = 2
	visibleRows := max(1, maxLines/2)
	visibleItems := visibleRows * cols
	start, end := windowBounds(len(fixtures), cursor, visibleItems)
	end = min(len(fixtures), start+visibleItems)
	colWidth := max(8, width/cols)
	rows := max(1, (end-start+1)/cols)
	lines := make([]string, 0, rows*2)

	for row := 0; row < rows; row++ {
		leftIdx := start + row
		rightIdx := start + rows + row
		firstTop, firstBottom := blankRoundMiniCard(colWidth, colorBgPane)
		secondTop, secondBottom := blankRoundMiniCard(width-colWidth, colorBgPane)
		if leftIdx < end {
			firstTop, firstBottom = formatRoundMiniCard(&fixtures[leftIdx], leftIdx == cursor, colWidth, leagueTitle)
		}
		if rightIdx < end {
			secondTop, secondBottom = formatRoundMiniCard(&fixtures[rightIdx], rightIdx == cursor, width-colWidth, leagueTitle)
		}
		lines = append(lines, firstTop+secondTop, firstBottom+secondBottom)
	}

	if len(lines) > maxLines {
		return lines[:maxLines]
	}
	return lines
}

func blankRoundMiniCard(width int, bg lipgloss.Color) (string, string) {
	blank := renderFullLine("", width, bg, colorTextMuted, false)
	return blank, blank
}

func formatRoundMiniCard(fixture *site.Fixture, selected bool, width int, leagueTitle string) (string, string) {
	bg := colorBgPane
	teamColor := colorTextSecondary
	dateColor := colorTextMuted
	if selected {
		bg = colorBgSelected
		teamColor = colorAccent
		dateColor = colorAccentDim
	}
	if fixture == nil {
		return blankRoundMiniCard(width, bg)
	}

	code := abbreviatedFixtureLine(fixture, 3)
	date := formatFixtureDateTime(fixture.WhenInfo, leagueTitle)
	codeLine := renderFullLine(" "+code, width, bg, teamColor, selected)
	dateLine := renderFullLine(" "+date, width, bg, dateColor, false)
	return codeLine, dateLine
}

func (m Model) matchFixtureRowLimit() int {
	limit := m.bodyHeightLimit()
	if limit == 0 {
		round := m.currentRound()
		if round == nil {
			return 0
		}
		return len(round.Fixtures)
	}

	reserved := 1 // fixtures pane header
	if m.loading && m.match == nil {
		reserved += 2
	}

	return max(0, limit-reserved)
}

func (m Model) matchFixtureMinHeight() int {
	return 4
}

func (m Model) standingsContentHeight() int {
	if m.league == nil || len(m.league.Standings) == 0 {
		return 2
	}
	return 2 + len(m.league.Standings) // pane header + column header + rows
}

func (m Model) standingsRowLimit() int {
	limit := m.bodyHeightLimit()
	if limit == 0 {
		return len(m.league.Standings)
	}
	return max(0, limit-2) // pane header + column header
}

func (m Model) fixtureRowLimit() int {
	limit := m.bodyHeightLimit()
	if limit == 0 {
		round := m.currentRound()
		if round == nil {
			return 0
		}
		return len(round.Fixtures)
	}

	reserved := 0
	if m.loading {
		reserved += 2
	}
	if m.err != "" {
		reserved += 2
	}

	return max(0, limit-reserved)
}

func (m Model) matchDetailPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Background(colorBgPanel)
	if limit := m.matchViewportHeight(); limit > 0 {
		base = base.Height(limit).MaxHeight(limit)
	}

	if m.loading && m.match == nil {
		return base.Render(renderPlainPaneLine(styleSubtle.Render("Loading…"), width, colorBgPanel))
	}

	if m.err != "" && m.match == nil {
		return base.Render(renderPlainPaneLine(styleSubtle.Render("Error: "+m.err), width, colorBgPanel))
	}

	if m.match == nil {
		return base.Render(renderPlainPaneLine(styleSubtle.Render("no match loaded"), width, colorBgPanel))
	}

	content := m.matchDetailContent(width)
	return base.Render(clipLines(content, m.matchScroll, m.matchViewportHeight()))
}

type statusBarItem struct {
	key   string
	label string
}

func (m Model) statusBarView() string {
	var parts []statusBarItem

	switch {
	case m.selectorActive():
		parts = []statusBarItem{{"j/k", "move"}, {"tab", "focus"}, {"enter", "load"}}
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
		lines = append(lines, item.Name)
	}
	return lines
}

func (m Model) matchDetailContent(width int) string {
	var b strings.Builder
	width = max(20, width)

	b.WriteString(renderBlankPanelLine(width))
	b.WriteString("\n")
	b.WriteString(renderFullLine(matchTitleLine(m.match.HomeTeam, matchDetailScore(m.match.Score), m.match.AwayTeam, width), width, colorBgPanel, colorTextPrimary, true))
	b.WriteString("\n")
	if meta := renderMatchMetaPanelLine(m.match.Meta, m.match.Weather, width); meta != "" {
		b.WriteString(meta)
		b.WriteString("\n")
	}
	b.WriteString(renderPanelRule(width))
	b.WriteString("\n")
	b.WriteString(renderSectionLabel("TIMELINE", width))
	b.WriteString("\n")
	firstHalfEvents, secondHalfEvents := splitTimelineRows(m.match.Events)
	for _, row := range firstHalfEvents {
		b.WriteString(renderTimelineRow(row, width))
		b.WriteString("\n")
	}
	b.WriteString(renderScoreAxisLine(halftimeScoreDisplay(m.match), width, colorTextMuted, false))
	b.WriteString("\n")
	for _, row := range secondHalfEvents {
		b.WriteString(renderTimelineRow(row, width))
		b.WriteString("\n")
	}
	if hasFinalScore(m.match.Score) {
		b.WriteString(renderScoreAxisLine("FT "+matchDetailScore(m.match.Score), width, colorTextPrimary, true))
		b.WriteString("\n")
	}

	if status := matchStatus(m.match); status != "" {
		b.WriteString(renderCenteredPanelLine(status, width, colorTextMuted, false))
		b.WriteString("\n")
	}

	if len(m.match.HomeLineup) > 0 || len(m.match.AwayLineup) > 0 {
		b.WriteString(renderLineupsLabel(width))
		b.WriteString("\n")
		b.WriteString(renderLineupHeaderRow(m.match.HomeTeam, m.match.AwayTeam, width))
		b.WriteString("\n")

		homeIdx := playerEventIndex(m.match.Events, "home")
		awayIdx := playerEventIndex(m.match.Events, "away")
		homeEntries := annotatedLineup(m.match.HomeLineup, homeIdx)
		awayEntries := annotatedLineup(m.match.AwayLineup, awayIdx)
		maxPlayers := max(len(homeEntries), len(awayEntries))
		playerWidth := lineupPlayerNameWidth(width)

		for i := 0; i < maxPlayers; i++ {
			var hEntry, aEntry lineupEntry
			if i < len(homeEntries) {
				hEntry = homeEntries[i]
			}
			if i < len(awayEntries) {
				aEntry = awayEntries[i]
			}

			b.WriteString(renderLineupPlayerRow(
				lineupDisplayName(hEntry, "home", playerWidth), lineupCardForEntry(hEntry, homeIdx),
				lineupDisplayName(aEntry, "away", playerWidth), lineupCardForEntry(aEntry, awayIdx),
				width,
			))
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderBlankPanelLine(width int) string {
	return renderFullLine("", width, colorBgPanel, colorTextSecondary, false)
}

func renderPanelRule(width int) string {
	return renderFullLine(strings.Repeat("─", max(0, width)), width, colorBgPanel, colorBorder, false)
}

func renderCenteredPanelLine(text string, width int, fg lipgloss.Color, bold bool) string {
	return renderFullLine(padCenter(truncate(text, width), width), width, colorBgPanel, fg, bold)
}

func matchTitleLine(home, score, away string, width int) string {
	if score == "vs" {
		return axisPlainLine(home, " vs ", away, width)
	}
	homeScore, awayScore, ok := splitScoreAxis(score)
	if !ok || width < 32 {
		return axisPlainLine(home+" "+score, "", away, width)
	}
	return axisPlainLine(home+" "+homeScore, " – ", awayScore+" "+away, width)
}

func renderScoreAxisLine(text string, width int, fg lipgloss.Color, bold bool) string {
	left, right, ok := splitScoreAxis(text)
	if !ok {
		return renderCenteredPanelLine(text, width, fg, bold)
	}
	return renderFullLine(axisPlainLine(left, " – ", right, width), width, colorBgPanel, fg, bold)
}

func splitScoreAxis(text string) (string, string, bool) {
	left, right, ok := strings.Cut(text, " – ")
	if ok {
		return strings.TrimSpace(left), strings.TrimSpace(right), true
	}
	left, right, ok = strings.Cut(text, "-")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(left), strings.TrimSpace(right), true
}

func axisPlainLine(left, center, right string, width int) string {
	centerWidth := ansi.StringWidth(center)
	if centerWidth == 0 {
		centerWidth = 1
	}
	leftWidth := max(0, (width-centerWidth)/2)
	rightWidth := max(0, width-leftWidth-centerWidth)
	return padLeft(truncate(left, leftWidth), leftWidth) + center + padRight(truncate(right, rightWidth), rightWidth)
}

func renderMatchMetaPanelLine(meta, weather string, width int) string {
	parts := matchMetaDisplayParts(meta, weather)
	if len(parts) == 0 {
		return ""
	}
	return renderFullLine(matchMetaAxisLine(parts, width), width, colorBgPanel, colorTextMuted, false)
}

func matchMetaDisplayParts(meta, weather string) []string {
	parts := matchMetaParts(meta, weather)
	displayParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if display := compactMatchMetaPart(part); display != "" {
			displayParts = append(displayParts, display)
		}
	}
	return displayParts
}

func matchMetaAxisLine(parts []string, width int) string {
	if len(parts) == 0 {
		return ""
	}
	parts = fitMatchMetaParts(parts, width)
	if len(parts) == 1 {
		return padCenter(truncate(parts[0], width), width)
	}

	separator := "  ·  "
	if len(parts)%2 == 0 {
		half := len(parts) / 2
		return axisPlainLine(strings.Join(parts[:half], separator), separator, strings.Join(parts[half:], separator), width)
	}

	middle := len(parts) / 2
	left := strings.Join(parts[:middle], separator) + separator
	right := separator + strings.Join(parts[middle+1:], separator)
	return axisPlainLine(left, parts[middle], right, width)
}

func fitMatchMetaParts(parts []string, width int) []string {
	if !matchMetaAxisTruncatesAttendance(parts, width) {
		return parts
	}

	withoutDate := make([]string, 0, len(parts))
	for _, part := range parts {
		if isMatchMetaDate(part) {
			continue
		}
		withoutDate = append(withoutDate, part)
	}
	if len(withoutDate) > 0 {
		return withoutDate
	}
	return parts
}

func matchMetaAxisTruncatesAttendance(parts []string, width int) bool {
	for _, part := range parts {
		if !strings.HasPrefix(part, "Att. ") {
			continue
		}
		return !strings.Contains(matchMetaAxisLineRaw(parts, width), part)
	}
	return false
}

func matchMetaAxisLineRaw(parts []string, width int) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return padCenter(truncate(parts[0], width), width)
	}

	separator := "  ·  "
	if len(parts)%2 == 0 {
		half := len(parts) / 2
		return axisPlainLine(strings.Join(parts[:half], separator), separator, strings.Join(parts[half:], separator), width)
	}

	middle := len(parts) / 2
	left := strings.Join(parts[:middle], separator) + separator
	right := separator + strings.Join(parts[middle+1:], separator)
	return axisPlainLine(left, parts[middle], right, width)
}

func isMatchMetaDate(value string) bool {
	parsed, err := time.Parse("Mon 2 January 2006, 15:04", value)
	return err == nil && !parsed.IsZero()
}

func compactMatchMetaPart(part string) string {
	switch {
	case strings.HasPrefix(part, "Attendance "):
		return "Att. " + groupDigits(strings.TrimPrefix(part, "Attendance "))
	case strings.HasPrefix(part, "Weather "):
		return compactWeather(strings.TrimPrefix(part, "Weather "))
	default:
		return formatMatchDateWithDay(part)
	}
}

func groupDigits(value string) string {
	digits := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	if digits == "" {
		return strings.TrimSpace(value)
	}
	parts := make([]string, 0, (len(digits)+2)/3)
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	parts = append([]string{digits}, parts...)
	return strings.Join(parts, " ")
}

func compactWeather(value string) string {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.TrimSuffix(cleaned, " C")
	cleaned = strings.TrimSuffix(cleaned, "C")
	if cleaned == "" || strings.Contains(cleaned, "°") {
		return cleaned
	}
	return cleaned + "°"
}

func formatMatchDateWithDay(value string) string {
	parsed, err := time.Parse("2 January 2006, 15:04", value)
	if err != nil {
		return value
	}
	return parsed.Format("Mon 2 January 2006, 15:04")
}

func halftimeScoreAlways(events []site.MatchEvent) string {
	homeGoals := 0
	awayGoals := 0
	for _, event := range sortedEvents(events) {
		if !event.HasMinute || event.Minute*100+event.Stoppage > 4599 || event.Kind != "GOAL" {
			continue
		}
		if event.TeamSide == "home" {
			homeGoals++
		} else if event.TeamSide == "away" {
			awayGoals++
		}
	}
	return fmt.Sprintf("HT %d – %d", homeGoals, awayGoals)
}

func halftimeScoreDisplay(page *site.MatchPage) string {
	if page == nil || len(page.Events) == 0 {
		return "HT —"
	}
	return halftimeScoreAlways(page.Events)
}

type timelineRow struct {
	home   string
	away   string
	marker string
	color  lipgloss.Color
}

func splitTimelineRows(events []site.MatchEvent) ([]timelineRow, []timelineRow) {
	firstHalf := make([]timelineRow, 0, 4)
	secondHalf := make([]timelineRow, 0, 4)
	for _, event := range sortedEvents(events) {
		marker, color, ok := timelineMarker(event.Kind)
		if !ok {
			continue
		}
		label := timelineEventLabel(event)
		if label == "" {
			continue
		}
		row := timelineRow{home: "—", away: "—", marker: marker, color: color}
		switch event.TeamSide {
		case "home":
			row.home = label
		case "away":
			row.away = label
		default:
			continue
		}

		if event.HasMinute && event.Minute*100+event.Stoppage <= 4599 {
			firstHalf = append(firstHalf, row)
		} else {
			secondHalf = append(secondHalf, row)
		}
	}

	return firstHalf, secondHalf
}

func timelineMarker(kind string) (string, lipgloss.Color, bool) {
	switch kind {
	case "GOAL":
		return "⚽", colorAccent, true
	case "MISS":
		return "❌", colorLoss, true
	case "RC":
		return "■", colorRed, true
	default:
		return "", colorTextMuted, false
	}
}

func timelineEventLabel(event site.MatchEvent) string {
	name := ansi.Strip(trimEventMinute(event))
	minute := formatMatchMinute(event.MinuteText)
	if name == "" || minute == "" {
		return ""
	}
	if event.TeamSide == "away" {
		return strings.TrimSpace(minute) + " " + name
	}
	return name + " " + minute
}

func renderTimelineRow(row timelineRow, width int) string {
	if row.home == "—" && row.away == "—" {
		return renderCenteredPanelLine("—", width, colorTextMuted, false)
	}
	left := ""
	right := ""
	if row.home != "—" {
		left = row.home
	}
	if row.away != "—" {
		right = row.away
	}
	return renderFullLine(axisPlainLine(left, " "+row.marker+" ", right, width), width, colorBgPanel, row.color, false)
}

func renderLineupsLabel(width int) string {
	return renderSectionLabel("LINEUPS", width)
}

func renderSectionLabel(label string, width int) string {
	label = strings.ToUpper(label)
	line := []rune(strings.Repeat("─", max(0, width)))
	if len(line) == 0 {
		return ""
	}
	center := width / 2
	labelStart := clamp(center-3, 0, max(0, width-len([]rune(label))))
	for i, r := range label {
		if labelStart+i >= len(line) {
			break
		}
		line[labelStart+i] = r
	}
	return renderFullLine(string(line), width, colorBgPanel, colorTextMuted, false)
}

func renderLineupTwoColumnRow(home, away string, width int, bg lipgloss.Color, homeColor lipgloss.Color, awayColor lipgloss.Color, bold bool) string {
	fg := homeColor
	if homeColor != awayColor {
		fg = colorTextSecondary
	}
	return renderFullLine(axisPlainLine(home, "│", away, width), width, bg, fg, bold)
}

func renderLineupHeaderRow(home, away string, width int) string {
	return renderFullLine(axisPlainLine(home, "   ", away, width), width, colorBgHeader, colorAccent, true)
}

type lineupCardMarker struct {
	color lipgloss.Color
	ok    bool
}

func lineupDisplayName(entry lineupEntry, side string, maxWidth int) string {
	return formatLineupPlayerWithCards(entry, side, maxWidth, true)
}

func lineupCardFor(player site.PlayerLine, idx map[string][]site.MatchEvent) lineupCardMarker {
	return cardMarkerAnnotationName(player.Name, idx)
}

func cardMarkerAnnotationName(name string, idx map[string][]site.MatchEvent) lineupCardMarker {
	return cardMarkerFromEvents(matchingPlayerEvents(name, idx))
}

func substituteCardMarkerAnnotationName(name string, idx map[string][]site.MatchEvent) lineupCardMarker {
	return cardMarkerFromEvents(matchingSubstituteCardEvents(name, idx))
}

func substituteCardMarkerAnnotationNameInRoster(name string, idx map[string][]site.MatchEvent, players []site.PlayerLine) lineupCardMarker {
	return cardMarkerFromEvents(matchingSubstituteCardEventsInRoster(name, idx, players))
}

func cardMarkerFromEvents(matched []site.MatchEvent) lineupCardMarker {
	if len(matched) == 0 {
		return lineupCardMarker{}
	}

	hasYellow := false
	for _, event := range matched {
		switch event.Kind {
		case "RC":
			return lineupCardMarker{color: colorRed, ok: true}
		case "YC":
			hasYellow = true
		}
	}
	if hasYellow {
		return lineupCardMarker{color: colorYellow, ok: true}
	}
	return lineupCardMarker{}
}

func lineupCardForEntry(entry lineupEntry, idx map[string][]site.MatchEvent) lineupCardMarker {
	return lineupCardFor(entry.player, idx)
}

func renderLineupPlayerRow(home string, homeCard lineupCardMarker, away string, awayCard lineupCardMarker, width int) string {
	leftWidth := max(1, width/2)
	rightWidth := max(1, width-leftWidth-1)
	defaultStyle := lipgloss.NewStyle().Foreground(colorTextSecondary).Background(colorBgPanel)
	divider := defaultStyle.Render("│")

	left := lineupSideCell(home, homeCard, leftWidth, "home", defaultStyle)
	right := lineupSideCell(away, awayCard, rightWidth, "away", defaultStyle)
	return left + divider + right
}

func lineupPlayerNameWidth(width int) int {
	leftWidth := max(1, width/2)
	rightWidth := max(1, width-leftWidth-1)
	return max(1, min(leftWidth, rightWidth)-2)
}

func lineupSideCell(name string, card lineupCardMarker, width int, side string, defaultStyle lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	cardWidth := 2
	nameWidth := max(0, width-cardWidth)
	cardStyle := lipgloss.NewStyle().Foreground(card.color).Background(colorBgPanel).Bold(true)

	switch side {
	case "home":
		text := renderLineupLabel(name, nameWidth, "home", defaultStyle)
		if !card.ok {
			return text + defaultStyle.Render("  ")
		}
		return text + cardStyle.Render("■") + defaultStyle.Render(" ")
	default:
		text := renderLineupLabel(name, nameWidth, "away", defaultStyle)
		if !card.ok {
			return defaultStyle.Render("  ") + text
		}
		return defaultStyle.Render(" ") + cardStyle.Render("■") + text
	}
}

func renderLineupLabel(label string, width int, side string, defaultStyle lipgloss.Style) string {
	text := truncate(label, width)
	if side == "home" {
		text = padLeft(text, width)
	} else {
		text = padRight(text, width)
	}

	noteStyle := lipgloss.NewStyle().Foreground(colorTextMuted).Background(colorBgPanel)
	yellowCardStyle := lipgloss.NewStyle().Foreground(colorYellow).Background(colorBgPanel).Bold(true)
	redCardStyle := lipgloss.NewStyle().Foreground(colorRed).Background(colorBgPanel).Bold(true)
	var b strings.Builder
	inNote := false
	for _, r := range text {
		switch string(r) {
		case lineupYellowCardToken:
			b.WriteString(yellowCardStyle.Render("■"))
			continue
		case lineupRedCardToken:
			b.WriteString(redCardStyle.Render("■"))
			continue
		}
		if r == '(' {
			inNote = true
		}
		style := defaultStyle
		if inNote {
			style = noteStyle
		}
		b.WriteString(style.Render(string(r)))
		if r == ')' {
			inNote = false
		}
	}
	return b.String()
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
