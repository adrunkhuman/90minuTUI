package ui

import (
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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
