package ui

import (
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
)

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

	reserved := 1
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

func (m Model) standingsRowLimit() int {
	limit := m.bodyHeightLimit()
	if limit == 0 {
		return len(m.league.Standings)
	}
	return max(0, limit-2)
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
