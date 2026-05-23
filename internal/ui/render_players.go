package ui

import (
	"sort"
	"strings"
	"unicode"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/x/ansi"
)

func renderPlayerLine(player site.PlayerLine) string {
	return formatPlayerLabel(player.Name)
}

func sortedEvents(events []site.MatchEvent) []site.MatchEvent {
	ordered := make([]site.MatchEvent, len(events))
	copy(ordered, events)

	sort.SliceStable(ordered, func(i, j int) bool {
		hi := ordered[i].HasMinute
		hj := ordered[j].HasMinute
		mi := ordered[i].Minute*100 + ordered[i].Stoppage
		mj := ordered[j].Minute*100 + ordered[j].Stoppage

		if hi != hj {
			return hi
		}
		if hi && mj != mi {
			return mi < mj
		}

		weightI := eventWeight(ordered[i].Kind)
		weightJ := eventWeight(ordered[j].Kind)
		if weightI != weightJ {
			return weightI < weightJ
		}

		return false
	})

	return ordered
}

func eventWeight(kind site.MatchEventKind) int {
	switch kind {
	case site.EventKindGoal:
		return 0
	case site.EventKindMiss:
		return 1
	case site.EventKindRedCard:
		return 2
	case site.EventKindYellowCard:
		return 3
	case site.EventKindSubstitution:
		return 4
	default:
		return 9
	}
}

func faintText(text string) string {
	if text == "" {
		return ""
	}
	return "\x1b[2m" + text + "\x1b[0m"
}

func faintPenaltySuffix(text string) string {
	if text == "" || !strings.Contains(text, "(pen)") {
		return text
	}
	return strings.ReplaceAll(text, "(pen)", faintText("(pen)"))
}

func eventPrefix(kind site.MatchEventKind) string {
	switch kind {
	case site.EventKindGoal:
		return "⚽"
	case site.EventKindMiss:
		return "❌"
	case site.EventKindSubstitution:
		return "↕"
	case site.EventKindYellowCard:
		return styleYellow.Render("■")
	case site.EventKindRedCard:
		return styleRed.Render("■")
	default:
		return "•"
	}
}

func formatPlayerLabel(value string) string {
	cleaned := canonicalPlayerName(value)
	if cleaned == "" {
		return ""
	}

	suffixes := make([]string, 0, 2)
	for {
		matches := trailingParenRe.FindStringSubmatch(cleaned)
		if len(matches) != 3 {
			break
		}
		suffixes = append([]string{strings.TrimSpace(matches[2])}, suffixes...)
		cleaned = normalizeDisplayText(matches[1])
	}

	words := strings.Fields(cleaned)
	if len(words) >= 2 {
		last := words[len(words)-1]
		initials := make([]string, 0, len(words)-1)
		for _, word := range words[:len(words)-1] {
			r := []rune(word)
			if len(r) == 0 {
				continue
			}
			initials = append(initials, string(unicode.ToUpper(r[0]))+".")
		}
		cleaned = strings.TrimSpace(strings.Join(append(initials, last), " "))
	}

	if len(suffixes) > 0 {
		cleaned += " " + strings.Join(suffixes, " ")
	}

	return faintPenaltySuffix(cleaned)
}

func canonicalPlayerName(value string) string {
	cleaned := normalizeDisplayText(value)
	if cleaned == "" {
		return ""
	}

	cleaned = playerNumberPrefixRe.ReplaceAllString(cleaned, "")
	cleaned = playerNumberSuffixRe.ReplaceAllString(cleaned, "")

	suffixes := make([]string, 0, 2)
	for {
		matches := trailingParenRe.FindStringSubmatch(cleaned)
		if len(matches) != 3 {
			break
		}
		inner := strings.TrimSpace(strings.Trim(matches[2], " ()"))
		cleaned = normalizeDisplayText(matches[1])
		if digitsOnly(inner) {
			continue
		}
		suffixes = append([]string{strings.TrimSpace(matches[2])}, suffixes...)
	}

	if len(suffixes) > 0 {
		cleaned += " " + strings.Join(suffixes, " ")
	}

	return cleaned
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func formatMatchMinute(minute string) string {
	cleaned := normalizeDisplayText(minute)
	if cleaned == "" {
		return ""
	}
	if strings.HasSuffix(cleaned, "'") {
		if ansi.StringWidth(cleaned) == 2 {
			return " " + cleaned
		}
		return cleaned
	}
	formatted := cleaned + "'"
	if ansi.StringWidth(formatted) == 2 {
		return " " + formatted
	}
	return formatted
}

func matchStatus(page *site.MatchPage) string {
	if page == nil {
		return ""
	}

	text := strings.ToLower(normalizeDisplayText(page.Meta + " " + page.Title))
	switch {
	case strings.Contains(text, "odwo"):
		return "OFF"
	case strings.Contains(text, "przelo") || strings.Contains(text, "przeło"):
		return "PPD"
	case strings.Contains(text, "dogr"):
		return "AET"
	default:
		return ""
	}
}

func playerMatchKey(label string) string {
	formatted := normalizeDisplayText(canonicalPlayerName(label))
	if formatted == "" {
		return ""
	}
	return strings.ToLower(formatted)
}

// Key substitutions by both players so lineup annotations can find entries and exits.
func playerEventIndex(events []site.MatchEvent, side site.TeamSide) map[string][]site.MatchEvent {
	idx := make(map[string][]site.MatchEvent)
	for _, e := range events {
		if e.TeamSide != side {
			continue
		}
		if e.Kind == site.EventKindSubstitution {
			out, in := substitutionPlayers(e)
			if key := playerMatchKey(out); key != "" {
				idx[key] = append(idx[key], e)
			}
			if key := playerMatchKey(in); key != "" {
				idx[key] = append(idx[key], e)
			}
			continue
		}
		name := eventPlayerText(e)
		if key := playerMatchKey(name); key != "" {
			idx[key] = append(idx[key], e)
		}
	}
	return idx
}

func substitutionPlayers(event site.MatchEvent) (string, string) {
	return canonicalPlayerName(event.SubstitutionOut), canonicalPlayerName(event.SubstitutionIn)
}

func matchingPlayerEvents(name string, idx map[string][]site.MatchEvent) []site.MatchEvent {
	key := playerMatchKey(name)
	if key == "" {
		return nil
	}
	matched := append([]site.MatchEvent(nil), idx[key]...)

	compact := playerCompactMatchKey(name)
	if compact == "" {
		return matched
	}
	if !isAbbreviatedPlayerName(name) {
		return matched
	}

	var compactMatched []site.MatchEvent
	for candidate, events := range idx {
		if candidate == key {
			continue
		}
		if playerCompactMatchKey(candidate) != compact {
			continue
		}
		if compactMatched != nil {
			return matched
		}
		compactMatched = events
	}
	return append(matched, compactMatched...)
}

func matchingPlayerEventsInRoster(name string, idx map[string][]site.MatchEvent, players []site.PlayerLine) []site.MatchEvent {
	matched := exactPlayerEvents(name, idx)
	if compactMatchCountForName(name, players) != 1 {
		return matched
	}

	compact := playerCompactMatchKey(name)
	var compactMatched []site.MatchEvent
	for candidate, events := range idx {
		if playerMatchKey(candidate) == playerMatchKey(name) || playerCompactMatchKey(candidate) != compact {
			continue
		}
		if compactMatched != nil {
			return matched
		}
		compactMatched = events
	}
	return append(matched, compactMatched...)
}

func exactPlayerEvents(name string, idx map[string][]site.MatchEvent) []site.MatchEvent {
	key := playerMatchKey(name)
	if key == "" {
		return nil
	}
	return append([]site.MatchEvent(nil), idx[key]...)
}

func playerCompactMatchKey(name string) string {
	return strings.ToLower(normalizeDisplayText(formatPlayerLabel(name)))
}

func matchingSubstituteCardEvents(name string, idx map[string][]site.MatchEvent) []site.MatchEvent {
	matched := append([]site.MatchEvent(nil), matchingPlayerEvents(name, idx)...)
	compact := playerCompactMatchKey(name)
	if compact == "" {
		return matched
	}

	var compactMatched []site.MatchEvent
	for candidate, events := range idx {
		if playerMatchKey(candidate) == playerMatchKey(name) || playerCompactMatchKey(candidate) != compact {
			continue
		}
		cardEvents := filterCardEvents(events)
		if len(cardEvents) == 0 {
			continue
		}
		if compactMatched != nil {
			return matched
		}
		compactMatched = cardEvents
	}
	return append(matched, compactMatched...)
}

func matchingSubstituteCardEventsInRoster(name string, idx map[string][]site.MatchEvent, players []site.PlayerLine) []site.MatchEvent {
	matched := exactPlayerEvents(name, idx)
	if compactMatchCountForName(name, players) != 1 {
		return matched
	}

	compact := playerCompactMatchKey(name)
	var compactMatched []site.MatchEvent
	for candidate, events := range idx {
		if playerMatchKey(candidate) == playerMatchKey(name) || playerCompactMatchKey(candidate) != compact {
			continue
		}
		cardEvents := filterCardEvents(events)
		if len(cardEvents) == 0 {
			continue
		}
		if compactMatched != nil {
			return matched
		}
		compactMatched = cardEvents
	}
	return append(matched, compactMatched...)
}

func filterCardEvents(events []site.MatchEvent) []site.MatchEvent {
	filtered := make([]site.MatchEvent, 0, len(events))
	for _, event := range events {
		if event.Kind == site.EventKindYellowCard || event.Kind == site.EventKindRedCard {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func isAbbreviatedPlayerName(name string) bool {
	for field := range strings.FieldsSeq(canonicalPlayerName(name)) {
		if strings.HasSuffix(field, ".") {
			return true
		}
	}
	return false
}
