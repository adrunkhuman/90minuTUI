package ui

import (
	"image/color"
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/x/ansi"
)

type timelineRow struct {
	home   string
	away   string
	marker string
	color  color.Color
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
		case site.TeamSideHome:
			row.home = label
		case site.TeamSideAway:
			row.away = label
		default:
			continue
		}

		if event.HasMinute && event.Minute*100+event.Stoppage <= firstHalfMinuteCeiling {
			firstHalf = append(firstHalf, row)
		} else {
			secondHalf = append(secondHalf, row)
		}
	}

	return firstHalf, secondHalf
}

func timelineMarker(kind site.MatchEventKind) (string, color.Color, bool) {
	switch kind {
	case site.EventKindGoal:
		return "⚽", colorAccent, true
	case site.EventKindMiss:
		return "❌", colorLoss, true
	case site.EventKindRedCard:
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
	if event.TeamSide == site.TeamSideAway {
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
