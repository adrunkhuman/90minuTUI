package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading terminal size..."
	}

	body := m.leagueSketchView()
	if m.matchView {
		body = m.matchSketchView()
	}
	if m.selectorActive() {
		body = m.selectorOverlayView(body)
	}

	return body + "\n" + m.statusBarView()
}

func (m Model) selectorOverlayView(body string) string {
	popup := m.selectorPopupView(selectorPopupWidth(m.width))
	bodyLines := strings.Split(body, "\n")
	totalHeight := max(len(bodyLines), max(1, m.height-1))
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
	if m.league != nil {
		b.WriteString(truncate(m.league.Title, width-2))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.league == nil || len(m.league.Standings) == 0 {
		b.WriteString("Standings unavailable")
		return base.Render(b.String())
	}

	b.WriteString("   # Team                P  W  D  L Pts\n")
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
	b.WriteString(title.Render(m.league.Title))
	b.WriteString("\n")
	if season := m.currentSeason(); season != nil {
		b.WriteString(season.Label)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(title.Render(fmt.Sprintf("Round %s", parseRoundNumber(round.Name, m.roundCursor+1))))
	if m.focus == focusFixtures {
		b.WriteString(" ")
		b.WriteString(focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")
	b.WriteString(round.Name)
	b.WriteString("\n\n")

	for _, line := range renderFixtureWindow(round.Fixtures, m.fixtureCursor, m.fixtureRowLimit()) {
		b.WriteString(line)
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
	return m.matchDetailPaneView(m.width)
}

func (m Model) bodyHeightLimit() int {
	if m.height <= 1 {
		return 0
	}

	return m.height - 1
}

func (m Model) standingsRowLimit() int {
	limit := m.bodyHeightLimit()
	if limit == 0 {
		return len(m.league.Standings)
	}

	reserved := 4
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

	reserved := 6
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
		parts = []string{"j/k: scroll", "esc: league", "r: reload", "q: quit"}
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
	heading := m.match.Title
	if m.match.HomeTeam != "" && m.match.AwayTeam != "" && m.match.Score != "" {
		heading = fmt.Sprintf("%s %s %s", m.match.HomeTeam, m.match.Score, m.match.AwayTeam)
	}

	b.WriteString(title.Render(heading))
	b.WriteString("\n")
	if m.match.Competition != "" {
		b.WriteString(m.match.Competition)
		b.WriteString("\n")
	}
	if m.match.Meta != "" {
		b.WriteString(m.match.Meta)
		b.WriteString("\n")
	}
	if m.match.Weather != "" {
		b.WriteString("Pogoda: ")
		b.WriteString(m.match.Weather)
		b.WriteString("\n")
	}

	if len(m.match.Events) > 0 {
		b.WriteString("\n")
		b.WriteString(title.Render("Timeline"))
		b.WriteString("\n")
		for _, event := range sortedEvents(m.match.Events) {
			homeText, awayText := "", ""
			eventText := formatEventLabel(event)
			if event.TeamSide == "home" {
				homeText = eventText
			} else {
				awayText = eventText
			}
			b.WriteString(renderSideBySide(homeText, event.MinuteText, awayText, width-4))
			b.WriteString("\n")
		}
	}

	if len(m.match.HomeLineup) > 0 || len(m.match.AwayLineup) > 0 {
		b.WriteString("\n")
		b.WriteString(title.Render("Lineups"))
		b.WriteString("\n")
		b.WriteString(renderSideBySide(m.match.HomeTeam, "", m.match.AwayTeam, width-4))
		b.WriteString("\n")

		maxPlayers := len(m.match.HomeLineup)
		if len(m.match.AwayLineup) > maxPlayers {
			maxPlayers = len(m.match.AwayLineup)
		}

		for i := 0; i < maxPlayers; i++ {
			homeText, awayText := "", ""
			if i < len(m.match.HomeLineup) {
				homeText = renderPlayerLine(m.match.HomeLineup[i])
			}
			if i < len(m.match.AwayLineup) {
				awayText = renderPlayerLine(m.match.AwayLineup[i])
			}
			b.WriteString(renderSideBySide(homeText, "", awayText, width-4))
			b.WriteString("\n")
		}
	}

	if m.match.NewsTitle != "" {
		b.WriteString("\n")
		b.WriteString(title.Render("Related news"))
		b.WriteString("\n")
		b.WriteString(m.match.NewsTitle)
		if m.match.NewsURL != "" {
			b.WriteString(" | ")
			b.WriteString(m.match.NewsURL)
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func (m Model) matchViewportHeight() int {
	return m.bodyHeightLimit()
}

func (m Model) matchScrollLimit() int {
	if !m.matchView || m.match == nil {
		return 0
	}
	viewport := m.matchViewportHeight()
	if viewport <= 0 {
		return 0
	}
	lines := strings.Split(m.matchDetailContent(m.width), "\n")
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
