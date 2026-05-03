package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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
		if !event.HasMinute || event.Minute*100+event.Stoppage > firstHalfMinuteCeiling || event.Kind != "GOAL" {
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
