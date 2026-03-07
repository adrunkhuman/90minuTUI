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

	emphasizeRight := m.matchView || m.focus == focusFixtures
	leftWidth, rightWidth := layoutWidths(m.width, m.sidebarCollapsed, emphasizeRight)
	right := m.rightPaneView(rightWidth)
	left := ""
	if leftWidth > 0 {
		left = m.leftPaneView(leftWidth)
	}

	if m.loading {
		right += "\n\nLoading..."
	}
	if m.err != "" {
		right += "\n\nError: " + m.err
	}

	help := "tab: focus  enter: load/open  j/k: move  h/l: round  s: sidebar  esc: back  r: reload  q: quit"
	if left == "" {
		return right + "\n\n" + help
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n\n" + help
}

func (m Model) leftPaneView(width int) string {
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

func (m Model) rightPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	if m.matchView && m.match != nil {
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

			events := sortedEvents(m.match.Events)
			for _, event := range events {
				homeText, awayText := "", ""
				eventText := formatEventLabel(event)
				if event.TeamSide == "home" {
					homeText = eventText
				} else {
					awayText = eventText
				}

				line := renderSideBySide(homeText, event.MinuteText, awayText, width-4)
				b.WriteString(line)
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
		b.WriteString("\n(esc to go back)")
		return base.Render(strings.TrimRight(b.String(), "\n"))
	}

	if m.league == nil || len(m.league.Rounds) == 0 {
		return base.Render("No league loaded yet")
	}

	round := m.currentRound()
	if round == nil {
		return base.Render("No fixtures in selected round")
	}

	var b strings.Builder
	b.WriteString(title.Render(m.league.Title))
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(m.league.URL))
	b.WriteString("\n\n")
	b.WriteString(title.Render(fmt.Sprintf("Round %d/%d: %s", m.roundCursor+1, len(m.league.Rounds), round.Name)))
	if m.focus == focusFixtures {
		b.WriteString(" " + focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")

	for i, f := range round.Fixtures {
		prefix := "  "
		if i == m.fixtureCursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%s %s %s", prefix, f.Home, f.Score, f.Away)
		if f.WhenInfo != "" {
			line += " | " + f.WhenInfo
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}
