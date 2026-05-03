package ui

import (
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
)

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
