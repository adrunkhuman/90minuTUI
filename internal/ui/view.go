package ui

import (
	"fmt"
	"strings"

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
		body = m.selectorOverlayView(body)
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

// topBarView renders a two-line header: competition + round on line 1, a dim
// separator on line 2. Returns "" when the bar should be suppressed.
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
		roundLabel := displayRoundLabel(round.Name, m.roundCursor+1)
		right = fmt.Sprintf("%s / %d", roundLabel, len(m.league.Rounds))
	}

	titleLine := barLine(label, right, m.width)
	sep := styleDim.Render(strings.Repeat("─", m.width))
	return titleLine + "\n" + sep
}

// barLine builds a fixed-width line: bold left label + subtle right label, space-padded.
func barLine(left, right string, width int) string {
	inner := width - 2 // 1 leading + 1 trailing space
	if inner <= 0 {
		return ""
	}
	if right == "" {
		text := truncate(left, inner)
		pad := max(0, inner-ansi.StringWidth(text))
		return " " + styleBold.Render(text) + strings.Repeat(" ", pad) + " "
	}
	rightW := ansi.StringWidth(right)
	leftMax := max(0, inner-rightW-1)
	leftText := truncate(left, leftMax)
	leftW := ansi.StringWidth(leftText)
	gap := max(1, inner-leftW-rightW)
	return " " + styleBold.Render(leftText) + strings.Repeat(" ", gap) + styleSubtle.Render(right) + " "
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
	innerWidth := max(24, width-4)
	seasonLines := renderSeasonsWindow(m.seasons, m.seasonCursor)
	competitionLines := renderCompetitionWindow(m.competitions, m.competitionCursor)
	leftWidth, rightWidth := selectorPaneWidths(innerWidth, seasonLines)

	rightHeading := "Leagues"
	if strings.TrimSpace(m.competitionTitle) != "" {
		rightHeading = m.competitionTitle
	}

	left := selectorPaneView(leftWidth, "Season", m.focus == focusSeasons, seasonLines)
	right := selectorPaneView(rightWidth, rightHeading, m.focus == focusCompetitions, competitionLines)

	// Vertical divider — height matches the taller pane.
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	divH := max(len(leftLines), len(rightLines))
	divParts := make([]string, divH)
	for i := range divParts {
		divParts[i] = styleDim.Render("│")
	}
	divider := strings.Join(divParts, "\n")

	var b strings.Builder
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right))

	if m.loading {
		b.WriteString("\n")
		b.WriteString(styleSubtle.Render("Loading…"))
	}
	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(styleSubtle.Render("Error: " + m.err))
	}

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Padding(0, 1)
	content := lipgloss.NewStyle().Width(innerWidth)

	return panel.Render(content.Render(strings.TrimRight(b.String(), "\n")))
}

func selectorPaneView(width int, heading string, focused bool, lines []string) string {
	base := lipgloss.NewStyle().Width(width)

	var headingStyle lipgloss.Style
	if focused {
		headingStyle = styleBold.Copy().Foreground(colorAccent)
	} else {
		headingStyle = styleSubtle
	}

	var b strings.Builder
	b.WriteString(headingStyle.Render(truncate(heading, width)))
	b.WriteString("\n")
	for _, line := range lines {
		b.WriteString(truncate(line, width))
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func selectorPaneWidths(total int, seasonLines []string) (int, int) {
	seasonWidth := lipgloss.Width("Season")
	for _, line := range seasonLines {
		seasonWidth = max(seasonWidth, lipgloss.Width(line))
	}

	leftWidth := clamp(seasonWidth+1, 14, 18)
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
		return styleDim.Render("│")
	}
	parts := make([]string, h)
	for i := range parts {
		parts[i] = styleDim.Render("│")
	}
	return strings.Join(parts, "\n")
}

// standingsPaneView renders the league table without a title header —
// the column header acts as the first row.
func (m Model) standingsPaneView(width int) string {
	return m.standingsPaneViewBounded(width)
}

func (m Model) standingsPaneViewBounded(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.MaxHeight(limit)
	}

	var b strings.Builder

	if m.league == nil || len(m.league.Standings) == 0 {
		b.WriteString(styleSubtle.Render("no standings"))
		return base.Render(b.String())
	}

	teamWidth := standingsTeamWidth(m.league.Standings, width-2)
	colHeader := styleSubtle.Render(truncate(
		fmt.Sprintf("  %2s %-*s %2s %2s %2s %2s %3s", "#", teamWidth, "Team", "P", "W", "D", "L", "Pts"),
		width-2,
	))
	b.WriteString(colHeader)
	b.WriteString("\n")
	for _, line := range renderStandingsWindow(m.league.Standings, m.currentFixture(), width-2, m.standingsRowLimit()) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

// leagueFixturesPaneView renders the fixture list. The round label is the only
// header line — no "Fixtures" title, no focus indicator.
func (m Model) leagueFixturesPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.MaxHeight(limit)
	}

	round := m.currentRound()
	if m.league == nil || round == nil {
		return base.Render(styleSubtle.Render("no league loaded"))
	}

	var b strings.Builder

	roundLabel := displayRoundLabel(round.Name, m.roundCursor+1)
	total := len(m.league.Rounds)
	b.WriteString(styleSubtle.Render(truncate(fmt.Sprintf("%s / %d", roundLabel, total), width-2)))
	b.WriteString("\n")

	for _, line := range renderFixtureWindow(round.Fixtures, m.fixtureCursor, m.fixtureRowLimit(), width-2, false) {
		b.WriteString(truncate(line, width-2))
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString("\n")
		b.WriteString(styleSubtle.Render("Loading…"))
	}
	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(styleSubtle.Render(m.err))
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
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

	standings := clamp(total/2, 8, 14)
	fixtures := total - standings
	if fixtures < minFixtures {
		fixtures = minFixtures
		standings = max(4, total-fixtures)
	}

	return standings, fixtures
}

func (m Model) matchFixtureRailView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.Height(limit).MaxHeight(limit)
	}

	round := m.currentRound()
	if round == nil {
		return base.Render(styleSubtle.Render("no fixtures"))
	}

	var b strings.Builder
	b.WriteString(styleSubtle.Render(truncate(displayRoundLabel(round.Name, m.roundCursor+1), width-2)))
	b.WriteString("\n")

	for _, line := range renderFixtureWindow(round.Fixtures, m.fixtureCursor, m.matchFixtureRowLimit(), width-2, true) {
		b.WriteString(truncate(line, width-2))
		b.WriteString("\n")
	}

	if m.loading && m.match == nil {
		b.WriteString("\n")
		b.WriteString(styleSubtle.Render("Loading…"))
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
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

	reserved := 1 // round label
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
	return 1 + len(m.league.Standings) // col header + rows
}

func (m Model) standingsRowLimit() int {
	limit := m.bodyHeightLimit()
	if limit == 0 {
		return len(m.league.Standings)
	}
	return max(0, limit-1) // 1 for col header
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

	reserved := 1 // round label
	if m.loading {
		reserved += 2
	}
	if m.err != "" {
		reserved += 2
	}

	return max(0, limit-reserved)
}

func (m Model) matchDetailPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if limit := m.matchViewportHeight(); limit > 0 {
		base = base.MaxHeight(limit)
	}

	if m.loading && m.match == nil {
		return base.Render(styleSubtle.Render("Loading…"))
	}

	if m.err != "" && m.match == nil {
		return base.Render(styleSubtle.Render("Error: " + m.err))
	}

	if m.match == nil {
		return base.Render(styleSubtle.Render("no match loaded"))
	}

	content := m.matchDetailContent(width)
	return base.Render(clipLines(content, m.matchScroll, m.matchViewportHeight()))
}

func (m Model) statusBarView() string {
	var parts []string

	switch {
	case m.selectorActive():
		parts = []string{"j/k: move", "tab: focus", "enter: load"}
		if len(m.competitionStack) > 0 {
			parts[2] = "enter: open"
			parts = append(parts, "esc: back")
		} else if m.league != nil {
			parts = append(parts, "esc: close")
		}
		parts = append(parts, "q: quit")

	case m.matchView:
		parts = []string{"j/k: fixture", "h/l: round", "pgup/pgdn: scroll", "esc: back", "r: reload", "q: quit"}

	default:
		enterHint := "enter: details"
		if !m.currentFixtureDrillable() {
			enterHint = "enter: unavail"
		}
		parts = []string{"j/k: move", "h/l: round", enterHint, "esc: selector", "q: quit"}
	}

	left := strings.Join(parts, "  ·  ")

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
		left = truncate(left, inner-rightW-2)
		leftW = ansi.StringWidth(left)
		gap = max(1, inner-leftW-rightW)
	}

	status := left + strings.Repeat(" ", gap) + right
	return lipgloss.NewStyle().Width(m.width).Padding(0, 1).Reverse(true).Render(truncate(status, m.width-2))
}

func selectorPopupWidth(total int, seasonLines []string, rightHeading string, competitionLines []string) int {
	if total <= 0 {
		return 36
	}

	leftWidth, rightWidth := selectorContentWidths(seasonLines, rightHeading, competitionLines)
	desired := leftWidth + 1 + rightWidth + 4 // divider + panel border/padding
	return clamp(desired, 40, max(40, total-2))
}

func selectorContentWidths(seasonLines []string, rightHeading string, competitionLines []string) (int, int) {
	seasonWidth := lipgloss.Width("Season")
	for _, line := range seasonLines {
		seasonWidth = max(seasonWidth, lipgloss.Width(line))
	}
	leftWidth := clamp(seasonWidth+1, 14, 18)

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
		lines = append(lines, "  "+item.Name)
	}
	return lines
}

func (m Model) matchDetailContent(width int) string {
	var b strings.Builder

	// Score header — one blank line of breathing room above.
	b.WriteString("\n")
	scoreText := matchDetailScore(m.match.Score)
	b.WriteString(styleBold.Render(renderMatchDetailRow(m.match.HomeTeam, scoreText, m.match.AwayTeam, width-4)))
	b.WriteString("\n")

	// Date and match details sit directly under the score.
	metaDate, metaDetails := matchMetaDisplay(m.match.Meta, m.match.Weather)
	if metaDate != "" {
		b.WriteString(styleSubtle.Render(padCenter(truncate(metaDate, width-4), width-4)))
		b.WriteString("\n")
	}
	if metaDetails != "" {
		b.WriteString(styleDim.Render(padCenter(truncate(metaDetails, width-4), width-4)))
		b.WriteString("\n")
	}

	status := matchStatus(m.match)
	headerEvents := headerEventRows(m.match.Events)
	ftDivider := finalScoreLine(m.match)
	if len(headerEvents) > 0 || status != "" || ftDivider != "" {
		b.WriteString(renderMatchDetailRow("", "", "", width-4))
		b.WriteString("\n")
		if status != "" {
			b.WriteString(renderMatchDetailRow("", status, "", width-4))
			b.WriteString("\n")
		}
		for _, row := range headerEvents {
			if row.isDivider {
				b.WriteString(styleDim.Render(renderMatchDividerRow(row.label, width-4)))
				b.WriteString("\n")
				continue
			}
			homeText, awayText := "", ""
			if row.side == "home" {
				homeText = row.label
			} else {
				awayText = row.label
			}
			b.WriteString(renderMatchDetailRow(homeText, row.minute, awayText, width-4))
			b.WriteString("\n")
		}
		if ftDivider != "" {
			b.WriteString(styleDim.Render(renderMatchDividerRow(ftDivider, width-4)))
			b.WriteString("\n")
		}
	}

	if len(m.match.HomeLineup) > 0 || len(m.match.AwayLineup) > 0 {
		b.WriteString("\n")
		b.WriteString(styleSubtle.Render(renderDividerLabel("Lineups", width-4)))
		b.WriteString("\n")
		b.WriteString(renderLineupRowWithMarker(
			styleBold.Render(m.match.HomeTeam),
			styleBold.Render(m.match.AwayTeam),
			" ",
			width-4,
		))
		b.WriteString("\n")

		homeIdx := playerEventIndex(m.match.Events, "home")
		awayIdx := playerEventIndex(m.match.Events, "away")
		homeEntries := annotatedLineup(m.match.HomeLineup, homeIdx)
		awayEntries := annotatedLineup(m.match.AwayLineup, awayIdx)
		maxPlayers := max(len(homeEntries), len(awayEntries))
		playerWidth := lineupPlayerWidth(width - 4)

		for i := 0; i < maxPlayers; i++ {
			var hEntry, aEntry lineupEntry
			if i < len(homeEntries) {
				hEntry = homeEntries[i]
			}
			if i < len(awayEntries) {
				aEntry = awayEntries[i]
			}

			homeName := formatLineupPlayer(hEntry, "home", playerWidth)
			awayName := formatLineupPlayer(aEntry, "away", playerWidth)

			b.WriteString(renderAnnotatedLineupRow(
				homeName, cardAnnotation(hEntry.player, homeIdx),
				awayName, cardAnnotation(aEntry.player, awayIdx),
				width-4,
			))
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// matchDetailScore formats the score for the match header with an en-dash,
// giving it more visual weight than the compact fixture list format.
func matchDetailScore(score string) string {
	trimmed := strings.TrimSpace(score)
	if trimmed == "" {
		return "? – ?"
	}
	parts := strings.SplitN(trimmed, "-", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]) + " – " + strings.TrimSpace(parts[1])
	}
	return trimmed
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
