package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading terminal size..."
	}

	topBar := m.topContextBarView()
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

func (m Model) selectorOverlayView(body string) string {
	popup := m.selectorPopupView(selectorPopupWidth(m.width))
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
	title := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	panel := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	content := lipgloss.NewStyle().Width(max(24, width-4))
	innerWidth := max(24, width-4)
	seasonLines := renderSeasonsWindow(m.seasons, m.seasonCursor)
	leftWidth, rightWidth := selectorPaneWidths(innerWidth, seasonLines)

	var b strings.Builder
	b.WriteString(title.Render("Season + league"))
	b.WriteString("\n\n")
	left := selectorPaneView(leftWidth, "Season", m.focus == focusSeasons, seasonLines, title, focusStyle)
	right := selectorPaneView(rightWidth, "Leagues", m.focus == focusCompetitions, renderCompetitionWindow(m.competitions, m.competitionCursor), title, focusStyle)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right))

	if m.loading {
		b.WriteString("\nLoading...")
	}
	if m.err != "" {
		b.WriteString("\nError: ")
		b.WriteString(m.err)
	}

	return panel.Render(content.Render(strings.TrimRight(b.String(), "\n")))
}

func selectorPaneView(width int, heading string, focused bool, lines []string, title, focusStyle lipgloss.Style) string {
	base := lipgloss.NewStyle().Width(width)

	var b strings.Builder
	b.WriteString(title.Render(truncate(heading, width)))
	if focused {
		b.WriteString(" ")
		b.WriteString(focusStyle.Render("[focus]"))
	}
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
	rightWidth := total - leftWidth - 2
	if rightWidth < 16 {
		rightWidth = 16
		leftWidth = max(14, total-rightWidth-2)
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
	return lipgloss.JoinHorizontal(lipgloss.Top, standings, fixtures)
}

func (m Model) standingsPaneView(width int) string {
	return m.standingsPaneViewBounded(width)
}

func (m Model) standingsPaneViewBounded(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.MaxHeight(limit)
	}
	title := lipgloss.NewStyle().Bold(true)

	var b strings.Builder
	b.WriteString(title.Render("Standings"))
	b.WriteString("\n")

	if m.league == nil || len(m.league.Standings) == 0 {
		b.WriteString("Standings unavailable")
		return base.Render(b.String())
	}

	b.WriteString(truncate("   # Team                P  W  D  L Pts", width-2))
	b.WriteString("\n")
	for _, line := range renderStandingsWindow(m.league.Standings, m.currentFixture(), width-2, m.standingsRowLimit()) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) leagueFixturesPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if limit := m.bodyHeightLimit(); limit > 0 {
		base = base.MaxHeight(limit)
	}
	title := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	round := m.currentRound()
	if m.league == nil || round == nil {
		return base.Render("No league loaded yet")
	}

	var b strings.Builder
	b.WriteString(title.Render("Fixtures"))
	if m.focus == focusFixtures {
		b.WriteString(" ")
		b.WriteString(focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")
	b.WriteString(displayRoundLabel(round.Name, m.roundCursor+1))
	b.WriteString("\n\n")

	for _, line := range renderFixtureWindow(round.Fixtures, m.fixtureCursor, m.fixtureRowLimit(), width-2, false) {
		b.WriteString(truncate(line, width-2))
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString("\nLoading...")
	}
	if m.err != "" {
		b.WriteString("\nError: ")
		b.WriteString(m.err)
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) matchSketchView() string {
	leftWidth, centerWidth, _ := matchLayoutWidths(m.width)
	if leftWidth == 0 {
		return m.matchDetailPaneView(centerWidth)
	}

	sidebar := m.matchSidebarView(leftWidth)
	detail := m.matchDetailPaneView(centerWidth)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, detail)
}

func (m Model) bodyHeightLimit() int {
	reserved := 1
	if m.topContextBarView() != "" {
		reserved++
	}
	if m.height <= reserved {
		return 0
	}

	return m.height - reserved
}

func (m Model) topContextBarView() string {
	if m.suppressTopBar {
		return ""
	}

	if m.league != nil {
		if label := displayCompetitionLabel(m.league.Title); label != "" {
			style := lipgloss.NewStyle().Width(m.width).Padding(0, 1).Bold(true)
			return style.Render(truncate(label, m.width-2))
		}
	}

	return ""
}

func (m Model) standingsRowLimit() int {
	limit := m.bodyHeightLimit()
	if limit == 0 {
		return len(m.league.Standings)
	}

	reserved := 2
	return max(0, limit-reserved)
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

	reserved := 3
	if m.loading {
		reserved += 2
	}
	if m.err != "" {
		reserved += 2
	}

	return max(0, limit-reserved)
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
	title := lipgloss.NewStyle().Bold(true)

	round := m.currentRound()
	if round == nil {
		return base.Render(title.Render("Fixtures") + "\nNo fixtures in selected round")
	}

	var b strings.Builder
	b.WriteString(title.Render("Fixtures"))
	b.WriteString("\n")
	b.WriteString(truncate(displayRoundLabel(round.Name, m.roundCursor+1), width-2))
	b.WriteString("\n")

	for _, line := range renderFixtureWindow(round.Fixtures, m.fixtureCursor, m.matchFixtureRowLimit(), width-2, true) {
		b.WriteString(truncate(line, width-2))
		b.WriteString("\n")
	}

	if m.loading && m.match == nil {
		b.WriteString("\nLoading...")
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

	reserved := 2
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

	return 2 + len(m.league.Standings)
}

func (m Model) matchDetailPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if limit := m.matchViewportHeight(); limit > 0 {
		base = base.MaxHeight(limit)
	}
	title := lipgloss.NewStyle().Bold(true)

	if m.loading && m.match == nil {
		return base.Render(title.Render("Match") + "\nLoading match details...")
	}

	if m.err != "" && m.match == nil {
		return base.Render(title.Render("Match") + "\nError: " + m.err)
	}

	if m.match == nil {
		return base.Render("No match loaded")
	}

	content := m.matchDetailContent(width)
	return base.Render(clipLines(content, m.matchScroll, m.matchViewportHeight()))
}

func (m Model) statusBarView() string {
	parts := []string{"j/k: move", "left/right: round", "enter: open", "esc: selector", "q: quit"}
	if m.matchView {
		parts = []string{"h/l: round", "j/k: fixture", "pgup/pgdn: scroll", "ctrl+u/d: scroll", "esc: league", "r: reload", "q: quit"}
	}
	if m.selectorActive() {
		parts = []string{"tab: focus", "j/k: move", "enter: load", "q: quit"}
		if m.league != nil {
			parts = []string{"tab: focus", "j/k: move", "enter: load", "esc: close", "q: quit"}
		}
	}

	status := strings.Join(parts, "  ") + "  |  fetched: " + formatFetchTime(m.lastFetchAt)
	if m.loading {
		status += "  |  loading"
	}
	if m.err != "" {
		status += "  |  error"
	}

	return lipgloss.NewStyle().Width(m.width).Padding(0, 1).Reverse(true).Render(truncate(status, m.width-2))
}

func selectorPopupWidth(total int) int {
	if total <= 0 {
		return 36
	}

	return clamp(total/2+4, 40, 68)
}

func (m Model) matchDetailContent(width int) string {
	title := lipgloss.NewStyle().Bold(true)
	var b strings.Builder
	b.WriteString(title.Render(renderMatchDetailRow(m.match.HomeTeam, normalizeScore(m.match.Score), m.match.AwayTeam, width-4)))
	b.WriteString("\n")

	status := matchStatus(m.match)
	headerEvents := headerEventRows(m.match.Events)
	if len(headerEvents) > 0 || status != "" {
		if status != "" {
			b.WriteString(renderMatchDetailRow("", status, "", width-4))
			b.WriteString("\n")
		}
		for _, row := range headerEvents {
			if row.isDivider {
				b.WriteString(renderMatchDividerRow(row.label, row.minute, width-4))
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
		if len(headerEvents) > 0 {
			if ftScore := finalScoreLine(m.match); ftScore != "" {
				b.WriteString(renderMatchDividerRow("FT", ftScore, width-4))
				b.WriteString("\n")
			}
		}
	}

	if len(m.match.HomeLineup) > 0 || len(m.match.AwayLineup) > 0 {
		b.WriteString("\n")
		b.WriteString(title.Render(renderCenteredText("Lineups", width-4)))
		b.WriteString("\n")
		b.WriteString(renderLineupRowWithMarker(
			title.Render(m.match.HomeTeam),
			title.Render(m.match.AwayTeam),
			" ",
			width-4,
		))
		b.WriteString("\n")

		homeIdx := playerEventIndex(m.match.Events, "home")
		awayIdx := playerEventIndex(m.match.Events, "away")
		homeEntries := reorderedLineup(m.match.HomeLineup, homeIdx)
		awayEntries := reorderedLineup(m.match.AwayLineup, awayIdx)
		maxPlayers := max(len(homeEntries), len(awayEntries))

		for i := 0; i < maxPlayers; i++ {
			var hEntry, aEntry lineupEntry
			if i < len(homeEntries) {
				hEntry = homeEntries[i]
			}
			if i < len(awayEntries) {
				aEntry = awayEntries[i]
			}

			// Sub-on players get their substitution minute at the outer edge of the name.
			homeName := formatPlayerLabel(hEntry.player.Name)
			if hEntry.subMinute != "" {
				homeName = hEntry.subMinute + " " + homeName
			}
			awayName := formatPlayerLabel(aEntry.player.Name)
			if aEntry.subMinute != "" {
				awayName = awayName + " " + aEntry.subMinute
			}

			b.WriteString(renderAnnotatedLineupRow(
				homeName, cardAnnotation(hEntry.player, homeIdx),
				awayName, cardAnnotation(aEntry.player, awayIdx),
				width-4,
			))
			b.WriteString("\n")
		}
	}

	if metaLine := displayMatchMeta(m.match.Meta, m.match.Weather); metaLine != "" {
		b.WriteString("\n")
		b.WriteString(title.Render(renderCenteredText("Details", width-4)))
		b.WriteString("\n")
		b.WriteString(metaLine)
	}

	return strings.TrimRight(b.String(), "\n")
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
