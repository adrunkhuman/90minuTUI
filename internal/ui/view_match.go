package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
