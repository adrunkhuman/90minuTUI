package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading terminal size..."
	}

	body := m.archiveSelectionView()
	if m.league != nil {
		if m.matchView {
			body = m.matchSketchView()
		} else {
			body = m.leagueSketchView()
		}
	}

	return body + "\n" + m.statusBarView()
}

func (m Model) archiveSelectionView() string {
	emphasizeRight := m.focus == focusFixtures
	leftWidth, rightWidth := layoutWidths(m.width, m.sidebarCollapsed, emphasizeRight)
	right := m.archiveRightPaneView(rightWidth)
	left := ""
	if leftWidth > 0 {
		left = m.archiveLeftPaneView(leftWidth)
	}

	if left == "" {
		return right
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) archiveLeftPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	var b strings.Builder
	b.WriteString(title.Render("Season"))
	if m.focus == focusSeasons {
		b.WriteString(" " + focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")
	for _, line := range renderSeasonsWindow(m.seasons, m.seasonCursor) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(title.Render("Leagues"))
	if m.focus == focusCompetitions {
		b.WriteString(" " + focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")
	for _, line := range renderCompetitionWindow(m.competitions, m.competitionCursor) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) archiveRightPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true)

	if m.league == nil || len(m.league.Rounds) == 0 {
		body := "Select a season and league to load fixtures"
		if m.loading {
			body += "\n\nLoading..."
		}
		if m.err != "" {
			body += "\n\nError: " + m.err
		}
		return base.Render(body)
	}

	round := m.currentRound()
	if round == nil {
		return base.Render("No fixtures in selected round")
	}

	var b strings.Builder
	b.WriteString(title.Render(m.league.Title))
	b.WriteString("\n")
	b.WriteString(round.Name)
	b.WriteString("\n\n")
	for i, f := range round.Fixtures {
		prefix := "  "
		if i == m.fixtureCursor {
			prefix = "> "
		}
		b.WriteString(prefix)
		b.WriteString(abbreviatedFixtureLine(&f))
		if f.WhenInfo != "" {
			b.WriteString(" | ")
			b.WriteString(f.WhenInfo)
		}
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
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
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
	for _, row := range m.league.Standings {
		selected := false
		if fixture := m.currentFixture(); fixture != nil {
			selected = strings.EqualFold(row.Team, fixture.Home) || strings.EqualFold(row.Team, fixture.Away)
		}
		b.WriteString(formatStandingRow(row, selected, width-2))
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) leagueFixturesPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
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

	for i, fixture := range round.Fixtures {
		prefix := "  "
		if i == m.fixtureCursor {
			prefix = "> "
		}
		b.WriteString(prefix)
		b.WriteString(abbreviatedFixtureLine(&fixture))
		if fixture.WhenInfo != "" {
			b.WriteString(" | ")
			b.WriteString(fixture.WhenInfo)
		}
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
	parts := make([]string, 0, 2)
	if leftWidth > 0 {
		parts = append(parts, m.matchSidebarView(leftWidth))
	}
	parts = append(parts, m.matchDetailPaneView(centerWidth))

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m Model) matchSidebarView(width int) string {
	context := m.matchContextStripView(width)
	fixture := m.matchFixtureStripView(width)
	return lipgloss.JoinVertical(lipgloss.Left, context, fixture)
}

func (m Model) matchContextStripView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true)

	round := m.currentRound()
	lines := []string{title.Render("Context")}
	if season := m.currentSeason(); season != nil {
		lines = append(lines, truncate(season.Label, width-2))
	}
	if m.league != nil {
		lines = append(lines, truncate(m.league.Title, width-2))
	}
	if round != nil {
		lines = append(lines, fmt.Sprintf("Round %s", parseRoundNumber(round.Name, m.roundCursor+1)))
	}

	return base.Render(strings.Join(lines, "\n"))
}

func (m Model) matchFixtureStripView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true)

	fixture := m.currentFixture()
	lines := []string{title.Render("Fixture"), truncate(abbreviatedFixtureLine(fixture), width-2)}
	if fixture != nil && fixture.WhenInfo != "" {
		lines = append(lines, truncate(fixture.WhenInfo, width-2))
	}

	return base.Render(strings.Join(lines, "\n"))
}

func (m Model) matchDetailPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
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

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) statusBarView() string {
	parts := []string{"j/k: move", "left/right: round", "enter: open", "esc: selector", "q: quit"}
	if m.matchView {
		parts = []string{"j/k: move", "esc: league", "q: quit"}
	}
	if m.league == nil {
		parts = []string{"tab: focus", "j/k: move", "enter: load", "q: quit"}
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
