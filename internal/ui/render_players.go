package ui

import (
	"sort"
	"strings"
	"unicode"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/lipgloss"
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

func eventWeight(kind string) int {
	switch kind {
	case "GOAL":
		return 0
	case "MISS":
		return 1
	case "RC":
		return 2
	case "YC":
		return 3
	case "SUB":
		return 4
	default:
		return 9
	}
}

func substitutionPlayers(text string) (string, string) {
	parts := strings.SplitN(normalizeDisplayText(text), "->", 2)
	if len(parts) != 2 {
		return "", ""
	}

	outgoing := normalizeDisplayText(substitutionMinutePrefixRe.ReplaceAllString(strings.TrimSpace(parts[0]), ""))
	incoming := normalizeDisplayText(parts[1])
	if strings.EqualFold(outgoing, "sub") {
		outgoing = ""
	}

	return canonicalPlayerName(outgoing), canonicalPlayerName(incoming)
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

func eventPrefix(kind string) string {
	switch kind {
	case "GOAL":
		return "⚽"
	case "MISS":
		return "❌"
	case "SUB":
		return "↕"
	case "YC":
		return styleYellow.Render("■")
	case "RC":
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

// playerEventIndex maps normalized compact player labels to events for the given side.
// SUB events are indexed under both outgoing and incoming player names.
func playerEventIndex(events []site.MatchEvent, side string) map[string][]site.MatchEvent {
	idx := make(map[string][]site.MatchEvent)
	for _, e := range events {
		if e.TeamSide != side {
			continue
		}
		if e.Kind == "SUB" {
			out, in := substitutionPlayers(e.Text)
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
		if event.Kind == "YC" || event.Kind == "RC" {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func isAbbreviatedPlayerName(name string) bool {
	for _, field := range strings.Fields(canonicalPlayerName(name)) {
		if strings.HasSuffix(field, ".") {
			return true
		}
	}
	return false
}

type lineupCardMarker struct {
	color lipgloss.Color
	ok    bool
}

type lineupEntry struct {
	player       site.PlayerLine
	enteredAt    string
	replaced     string
	replacedYC   lineupCardMarker
	leftAt       string
	replacedBy   string
	replacedByYC lineupCardMarker
}

const (
	lineupYellowCardToken = "\ue000"
	lineupRedCardToken    = "\ue001"
)

// A lineup row can carry both entry and exit notes for players who came on and were later replaced.
func formatLineupPlayer(entry lineupEntry, side string, maxWidth int) string {
	return formatLineupPlayerWithCards(entry, side, maxWidth, false)
}

func formatLineupPlayerWithCards(entry lineupEntry, side string, maxWidth int, tokens bool) string {
	name := formatPlayerLabel(entry.player.Name)
	if entry.enteredAt == "" && entry.replaced == "" && entry.leftAt == "" && entry.replacedBy == "" {
		return name
	}

	label := lineupPlayerLabel(entry, side, name, false, tokens)
	if maxWidth > 0 && ansi.StringWidth(label) > maxWidth {
		shortened := lineupPlayerLabel(entry, side, name, true, tokens)
		if ansi.StringWidth(shortened) < ansi.StringWidth(label) {
			return shortened
		}
	}

	return label
}

func lineupPlayerLabel(entry lineupEntry, side, name string, shortenNotes, tokens bool) string {
	notes := lineupNotes(entry, side, shortenNotes, tokens)
	if len(notes) == 0 {
		return name
	}

	if side == "home" {
		parts := append(notes, name)
		return strings.Join(parts, " ")
	}

	parts := append([]string{name}, notes...)
	return strings.Join(parts, " ")
}

func lineupNotes(entry lineupEntry, side string, shortenNotes, tokens bool) []string {
	notes := make([]string, 0, 2)

	if note := entryNote(entry, side, shortenNotes, tokens); note != "" {
		notes = append(notes, note)
	}
	if note := exitNote(entry, side, shortenNotes, tokens); note != "" {
		notes = append(notes, note)
	}

	return notes
}

func entryNote(entry lineupEntry, side string, shortenNotes, tokens bool) string {
	if entry.enteredAt == "" {
		return ""
	}

	replaced := formatSubNoteName(entry.replaced, shortenNotes)
	card := lineupCardText(entry.replacedYC, tokens)
	if side == "home" {
		text := "(" + entry.enteredAt
		if replaced != "" {
			text += " for " + replaced + card
		}
		return substitutionNoteText(text+")", tokens)
	}

	text := "(for "
	if card != "" {
		text += card + " "
	}
	if replaced != "" {
		text += replaced + " "
	}
	text += entry.enteredAt
	return substitutionNoteText(text+")", tokens)
}

func exitNote(entry lineupEntry, side string, shortenNotes, tokens bool) string {
	if entry.replacedBy == "" {
		return ""
	}

	replacement := formatSubNoteName(entry.replacedBy, shortenNotes)
	card := lineupCardText(entry.replacedByYC, tokens)
	text := "("
	if side == "home" && entry.leftAt != "" {
		text += entry.leftAt + " "
	}
	if side != "home" && card != "" {
		text += card + " "
	}
	text += replacement
	if side == "home" {
		text += card
	}
	if side != "home" && entry.leftAt != "" {
		text += " " + entry.leftAt
	}
	return substitutionNoteText(text+")", tokens)
}

func substitutionNoteText(text string, tokens bool) string {
	if tokens {
		return text
	}
	return faintText(text)
}

func lineupCardText(card lineupCardMarker, tokens bool) string {
	if !card.ok {
		return ""
	}
	if card.color == colorRed {
		if tokens {
			return lineupRedCardToken
		}
		return "■"
	}
	if tokens {
		return lineupYellowCardToken
	}
	return "■"
}

// Under width pressure, shorten only substitution-note names so the main player label stays stable.
func formatSubNoteName(name string, surnameOnly bool) string {
	formatted := formatPlayerLabel(name)
	if !surnameOnly {
		return formatted
	}

	cleaned := canonicalPlayerName(name)
	if cleaned == "" {
		return ""
	}

	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return ""
	}

	return faintPenaltySuffix(words[len(words)-1])
}

// Players can collect both entry and exit notes when they are substituted on and off in one match.
func annotateLineupPlayer(player site.PlayerLine, idx map[string][]site.MatchEvent) lineupEntry {
	return annotateLineupPlayerInRoster(player, idx, nil)
}

func annotateLineupPlayerInRoster(player site.PlayerLine, idx map[string][]site.MatchEvent, players []site.PlayerLine) lineupEntry {
	entry := lineupEntry{player: player}
	for _, event := range sortedEvents(matchingPlayerEventsInRoster(player.Name, idx, players)) {
		if event.Kind != "SUB" {
			continue
		}

		out, in := substitutionPlayers(event.Text)
		minute := strings.TrimSpace(formatMatchMinute(event.MinuteText))
		if playerNameMatchesInRoster(in, player.Name, players) {
			entry.enteredAt = minute
			entry.replaced = out
			entry.replacedYC = substituteCardMarkerAnnotationNameInRoster(out, idx, players)
		}
		if playerNameMatchesInRoster(out, player.Name, players) {
			entry.leftAt = minute
			entry.replacedBy = in
			entry.replacedByYC = substituteCardMarkerAnnotationNameInRoster(in, idx, players)
		}
	}

	return entry
}

func playerNameMatches(left, right string) bool {
	leftKey := playerMatchKey(left)
	rightKey := playerMatchKey(right)
	if leftKey == "" || rightKey == "" {
		return false
	}
	if leftKey == rightKey {
		return true
	}
	return isAbbreviatedPlayerName(right) && playerCompactMatchKey(left) == playerCompactMatchKey(right)
}

func playerNameMatchesInRoster(left, right string, players []site.PlayerLine) bool {
	leftKey := playerMatchKey(left)
	rightKey := playerMatchKey(right)
	if leftKey != "" && leftKey == rightKey {
		return true
	}
	if playerCompactMatchKey(left) == "" || playerCompactMatchKey(left) != playerCompactMatchKey(right) {
		return false
	}
	if !isAbbreviatedPlayerName(left) && !isAbbreviatedPlayerName(right) {
		return false
	}
	return compactMatchCountForName(right, players) == 1
}

func compactMatchCountForName(name string, players []site.PlayerLine) int {
	compact := playerCompactMatchKey(name)
	if compact == "" {
		return 0
	}

	count := 0
	foundQuery := false
	for _, player := range players {
		if playerCompactMatchKey(player.Name) != compact {
			continue
		}
		count++
		if playerNameMatches(name, player.Name) || playerNameMatches(player.Name, name) {
			foundQuery = true
		}
	}
	if !foundQuery {
		count++
	}
	return count
}

// Synthetic entrant rows are only added when a substitute was later replaced again.
func annotatedLineup(players []site.PlayerLine, idx map[string][]site.MatchEvent) []lineupEntry {
	if len(players) == 0 {
		return nil
	}

	byKey := make(map[string]site.PlayerLine, len(players))
	for _, player := range players {
		byKey[playerMatchKey(player.Name)] = player
	}

	entries := make([]lineupEntry, 0, len(players))
	addedSynthetic := make(map[string]bool)
	for _, player := range players {
		entry := annotateLineupPlayerInRoster(player, idx, players)
		entries = append(entries, entry)

		inKey := playerMatchKey(entry.replacedBy)
		syntheticKey := playerSyntheticKey(entry.replacedBy)
		if inKey == "" || addedSynthetic[syntheticKey] {
			continue
		}
		if _, exists := byKey[inKey]; exists || lineupContainsPlayer(players, entry.replacedBy) {
			continue
		}

		synthetic := annotateLineupPlayerInRoster(site.PlayerLine{Name: entry.replacedBy}, idx, players)
		if synthetic.replacedBy == "" {
			continue
		}

		entries = append(entries, synthetic)
		addedSynthetic[syntheticKey] = true
	}

	return entries
}

func lineupContainsPlayer(players []site.PlayerLine, name string) bool {
	for _, player := range players {
		if playerNameMatches(name, player.Name) || playerNameMatches(player.Name, name) {
			return true
		}
	}
	return false
}

func playerSyntheticKey(name string) string {
	if isAbbreviatedPlayerName(name) {
		return playerCompactMatchKey(name)
	}
	if key := playerMatchKey(name); key != "" {
		return key
	}
	if compact := playerCompactMatchKey(name); compact != "" {
		return compact
	}
	return ""
}
