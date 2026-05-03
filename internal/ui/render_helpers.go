package ui

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/x/ansi"
)

var polishMonthReplacer = strings.NewReplacer(
	"stycznia", "January",
	"lutego", "February",
	"marca", "March",
	"kwietnia", "April",
	"maja", "May",
	"czerwca", "June",
	"lipca", "July",
	"sierpnia", "August",
	"wrzesnia", "September",
	"września", "September",
	"pazdziernika", "October",
	"października", "October",
	"listopada", "November",
	"grudnia", "December",
)

var roundSuffixRe = regexp.MustCompile(`(?i)\s*-\s*(?:kolejka|runda)\s+(\d+)\s*$`)
var matchDatePrefixRe = regexp.MustCompile(`^\d{1,2}\s+\p{L}+\s+\d{4},\s+\d{1,2}:\d{2}`)
var leadingAttendanceRe = regexp.MustCompile(`^(\d[\d ]*)\b`)
var fixtureWhenInfoRe = regexp.MustCompile(`(?i)^(\d{1,2})\s+([\p{L}]+)(?:\s+\d{4})?,\s*(\d{1,2}:\d{2})`)
var fixtureDateTimeRe = regexp.MustCompile(`(?i)(\d{1,2})\s+([\p{L}]+)(?:\s+(\d{4}))?,\s*(\d{1,2}:\d{2})`)
var leagueSeasonYearsRe = regexp.MustCompile(`(\d{4})\s*/\s*(\d{2,4})`)
var playerNumberPrefixRe = regexp.MustCompile(`^\(\d+\)\s*`)
var playerNumberSuffixRe = regexp.MustCompile(`\s+\(\d+\)$`)
var trailingParenRe = regexp.MustCompile(`^(.*?)(\s+\([^)]*\))$`)
var substitutionMinutePrefixRe = regexp.MustCompile(`^\d+'?\s*`)

func renderSeasonsWindow(seasons []site.Season, cursor int) []string {
	if len(seasons) == 0 {
		return []string{"(none)"}
	}

	start, end := windowBounds(len(seasons), cursor, 10)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, seasons[i].Label)
	}

	return lines
}

func standingSelectionIndices(rows []site.StandingRow, fixture *site.Fixture) []int {
	if fixture == nil {
		return nil
	}

	indices := make([]int, 0, 2)
	for i, row := range rows {
		if strings.EqualFold(row.Team, fixture.Home) || strings.EqualFold(row.Team, fixture.Away) {
			indices = append(indices, i)
		}
	}

	return indices
}

func windowBounds(total, cursor, maxItems int) (int, int) {
	if maxItems <= 0 {
		return 0, 0
	}
	if total <= maxItems {
		return 0, total
	}

	half := maxItems / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}

	end := start + maxItems
	if end > total {
		end = total
		start = end - maxItems
	}

	return start, end
}

func anchoredWindowBounds(total int, anchors []int, maxItems int) (int, int) {
	// Keep the highlighted home/away rows visible together when standings overflow.
	if maxItems <= 0 {
		return 0, 0
	}
	if total <= maxItems {
		return 0, total
	}
	if len(anchors) == 0 {
		return windowBounds(total, 0, maxItems)
	}

	minAnchor := anchors[0]
	maxAnchor := anchors[0]
	for _, anchor := range anchors[1:] {
		if anchor < minAnchor {
			minAnchor = anchor
		}
		if anchor > maxAnchor {
			maxAnchor = anchor
		}
	}

	span := maxAnchor - minAnchor + 1
	if span >= maxItems {
		return windowBounds(total, minAnchor, maxItems)
	}

	start := minAnchor - (maxItems-span)/2
	if start < 0 {
		start = 0
	}
	end := start + maxItems
	if end > total {
		end = total
		start = end - maxItems
	}

	return start, end
}

func clamp(v, minV, maxV int) int {
	if maxV < minV {
		return minV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func atoiOrNeg(s string) int {
	value := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &value); err != nil {
		return -1
	}
	return value
}

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

func halftimeScore(events []site.MatchEvent) string {
	homeGoals := 0
	awayGoals := 0
	hasSecondHalf := false

	for _, event := range sortedEvents(events) {
		if !event.HasMinute {
			continue
		}
		// minute encodes stoppage as MM*100+extra, so 45:59 is the first-half ceiling.
		minute := event.Minute*100 + event.Stoppage
		if minute <= 4599 {
			if event.Kind == "GOAL" {
				if event.TeamSide == "home" {
					homeGoals++
				} else if event.TeamSide == "away" {
					awayGoals++
				}
			}
			continue
		}
		hasSecondHalf = true
	}

	if !hasSecondHalf {
		return ""
	}

	return fmt.Sprintf("HT %d – %d", homeGoals, awayGoals)
}

func finalScoreLine(page *site.MatchPage) string {
	if page == nil {
		return ""
	}

	score := strings.TrimSpace(page.Score)
	if score == "" {
		return ""
	}

	return "FT " + dividerScore(score)
}

func dividerScore(score string) string {
	trimmed := strings.TrimSpace(score)
	if trimmed == "" {
		return "?-?"
	}

	parts := strings.SplitN(trimmed, "-", 2)
	if len(parts) != 2 {
		return normalizeScore(trimmed)
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if left == "" || right == "" {
		return normalizeScore(trimmed)
	}

	return left + " – " + right
}

func matchMetaParts(meta, weather string) []string {
	parts := make([]string, 0, 4)
	cleanedMeta := normalizeDisplayText(meta)
	if cleanedMeta != "" {
		datePart := matchDatePrefixRe.FindString(cleanedMeta)
		remainder := strings.TrimSpace(strings.TrimPrefix(cleanedMeta, datePart))
		if datePart != "" {
			parts = append(parts, translatePolishDateText(datePart))
		}
		if matches := leadingAttendanceRe.FindStringSubmatch(remainder); len(matches) == 2 {
			parts = append(parts, "Attendance "+strings.TrimSpace(matches[1]))
			remainder = strings.TrimSpace(strings.TrimPrefix(remainder, matches[0]))
		}
		if remainder != "" {
			translated := translatePolishDateText(remainder)
			// Drop trailing "(City)" — only venue/city name, not part of the ref's identity.
			refName := strings.TrimSpace(trailingParenRe.ReplaceAllString(translated, "$1"))
			parts = append(parts, "Ref. "+refName)
		}
		if len(parts) == 0 {
			parts = append(parts, translatePolishDateText(cleanedMeta))
		}
	}

	cleanedWeather := normalizeDisplayText(weather)
	if cleanedWeather != "" {
		parts = append(parts, "Weather "+cleanedWeather)
	}

	return parts
}

func truncate(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= maxLen {
		return value
	}
	if maxLen == 1 {
		return "…"
	}

	return ansi.Cut(value, 0, maxLen-1) + "…"
}

func padRight(value string, width int) string {
	pad := width - ansi.StringWidth(value)
	if pad <= 0 {
		return value
	}

	return value + strings.Repeat(" ", pad)
}

func padLeft(value string, width int) string {
	pad := width - ansi.StringWidth(value)
	if pad <= 0 {
		return value
	}

	return strings.Repeat(" ", pad) + value
}

func padCenter(value string, width int) string {
	pad := width - ansi.StringWidth(value)
	if pad <= 0 {
		return value
	}

	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
}

const leftPaneWidth = 54

func leagueLayoutWidths(total int) (int, int) {
	if total < 88 {
		return 0, total
	}

	leftWidth := leftPaneWidth
	rightWidth := total - leftWidth - 1
	if rightWidth < 36 {
		rightWidth = 36
		leftWidth = max(0, total-rightWidth-1)
	}

	return leftWidth, rightWidth
}

func matchLayoutWidths(total int) (int, int, int) {
	if total < 72 {
		return 0, total, 0
	}

	leftWidth := leftPaneWidth
	centerWidth := total - leftWidth - 1
	if centerWidth >= 40 {
		return leftWidth, centerWidth, 0
	}

	return 0, total, 0
}

func abbreviateTeamName(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "---"
	}

	var letters []rune
	for _, r := range []rune(clean) {
		if unicode.IsLetter(r) {
			letters = append(letters, unicode.ToUpper(r))
		}
		if len(letters) == 3 {
			break
		}
	}

	if len(letters) == 0 {
		letters = []rune(strings.ToUpper(clean))
	}
	if len(letters) >= 3 {
		return string(letters[:3])
	}

	return padRight(string(letters), 3)
}

func abbreviatedFixtureLine(fixture *site.Fixture, scoreWidth int) string {
	if fixture == nil {
		return "--- " + padCenter("?-?", scoreWidth) + " ---"
	}

	score := padCenter(normalizeScore(fixture.Score), scoreWidth)
	return fmt.Sprintf("%s %s %s", abbreviateTeamName(fixture.Home), score, abbreviateTeamName(fixture.Away))
}

func fixtureAvailabilitySuffix(fixture *site.Fixture, width int, compact bool) string {
	if fixture == nil || strings.TrimSpace(fixture.MatchURL) != "" {
		return ""
	}

	suffix := " [no details]"
	minWidth := 64
	if compact {
		minWidth = 40
	}
	if width < minWidth+ansi.StringWidth(suffix) {
		return ""
	}

	return suffix
}

func normalizeScore(score string) string {
	trimmed := strings.TrimSpace(score)
	if trimmed == "" {
		return "?-?"
	}
	return strings.ReplaceAll(trimmed, "-", "-")
}

func formatFetchTime(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}
	return ts.Format("15:04:05")
}

func standingsTeamWidth(rows []site.StandingRow, width int) int {
	const reserved = 21
	minWidth := ansi.StringWidth("Team")
	maxWidth := max(minWidth, width-reserved)
	if maxWidth <= minWidth {
		return minWidth
	}

	teamWidth := minWidth
	for _, row := range rows {
		teamWidth = max(teamWidth, ansi.StringWidth(row.Team))
	}
	if teamWidth > maxWidth {
		return maxWidth
	}
	return teamWidth
}

func parseRoundNumber(name string, fallback int) string {
	for _, field := range strings.Fields(name) {
		if _, err := strconv.Atoi(field); err == nil {
			return field
		}
		trimmed := strings.TrimRight(field, ".")
		if _, err := strconv.Atoi(trimmed); err == nil {
			return trimmed
		}
	}

	if fallback <= 0 {
		return "?"
	}
	return strconv.Itoa(fallback)
}

func displayCompetitionLabel(label string) string {
	cleaned := normalizeDisplayText(label)
	if cleaned == "" {
		return ""
	}

	if matches := roundSuffixRe.FindStringSubmatch(cleaned); len(matches) == 2 {
		base := strings.TrimSpace(roundSuffixRe.ReplaceAllString(cleaned, ""))
		if base != "" {
			return base + " - Round " + matches[1]
		}
	}

	return translatePolishDateText(cleaned)
}

func displayRoundLabel(name string, fallback int) string {
	cleaned := normalizeDisplayText(name)
	if cleaned == "" {
		return "Round " + parseRoundNumber("", fallback)
	}

	lower := strings.ToLower(cleaned)
	if lower == "wyniki" {
		return "Results"
	}
	if strings.Contains(lower, "fina") {
		return translatePolishStageLabel(cleaned)
	}
	if strings.Contains(lower, "bara") || strings.Contains(lower, "play-off") || strings.Contains(lower, "playoff") {
		return translatePolishStageLabel(cleaned)
	}

	if strings.Contains(lower, "kolejka") || strings.Contains(lower, "runda") {
		number := parseRoundNumber(cleaned, fallback)
		if idx := strings.Index(cleaned, "-"); idx >= 0 {
			suffix := translatePolishDateText(strings.TrimSpace(cleaned[idx+1:]))
			if suffix != "" {
				return "Round " + number + " - " + suffix
			}
		}
		return "Round " + number
	}

	return translatePolishDateText(cleaned)
}

func displayRoundLabelWithFixtures(name string, fallback int, fixtures []site.Fixture, leagueTitle string) string {
	label := displayRoundLabel(name, fallback)
	span := roundFixtureDateSpan(fixtures, leagueTitle)
	if span == "" || !strings.HasPrefix(label, "Round ") {
		return label
	}

	roundPart, _, ok := strings.Cut(label, " - ")
	if !ok {
		roundPart = label
	}
	return roundPart + " - " + span
}

func roundFixtureDateSpan(fixtures []site.Fixture, leagueTitle string) string {
	var first time.Time
	var last time.Time
	for _, fixture := range fixtures {
		date, ok := fixtureDisplayDate(fixture.WhenInfo, leagueTitle)
		if !ok {
			continue
		}
		if first.IsZero() || date.Before(first) {
			first = date
		}
		if last.IsZero() || date.After(last) {
			last = date
		}
	}

	if first.IsZero() {
		return ""
	}
	if sameCalendarDate(first, last) {
		return first.Format("Jan 2")
	}
	if first.Year() == last.Year() && first.Month() == last.Month() {
		return fmt.Sprintf("%s-%d", first.Format("Jan 2"), last.Day())
	}
	return fmt.Sprintf("%s-%s", first.Format("Jan 2"), last.Format("Jan 2"))
}

func fixtureDisplayDate(value, leagueTitle string) (time.Time, bool) {
	cleaned := normalizeDisplayText(value)
	matches := fixtureDateTimeRe.FindStringSubmatch(cleaned)
	if len(matches) != 5 {
		return time.Time{}, false
	}

	month := polishMonthNumber(matches[2])
	if month == "" {
		return time.Time{}, false
	}

	day := atoiOrNeg(matches[1])
	monthNum := atoiOrNeg(month)
	year := atoiOrNeg(matches[3])
	if year < 0 {
		year = inferFixtureYear(monthNum, leagueTitle)
	}
	if day <= 0 || monthNum <= 0 || year <= 0 {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(monthNum), day, 0, 0, 0, 0, time.Local), true
}

func sameCalendarDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func trimEventMinute(event site.MatchEvent) string {
	text := eventPlayerText(event)
	return formatPlayerLabel(text)
}

func eventPlayerText(event site.MatchEvent) string {
	text := normalizeDisplayText(event.Text)
	if text == "" || event.MinuteText == "" {
		return text
	}

	minute := regexp.QuoteMeta(event.MinuteText)
	re := regexp.MustCompile(`^` + minute + `'?\s*(?:->\s*)?`)
	text = strings.TrimSpace(re.ReplaceAllString(text, ""))
	re = regexp.MustCompile(`\s+` + minute + `(\s*(?:\([^)]*\))?)$`)
	text = strings.TrimSpace(re.ReplaceAllString(text, `$1`))

	if event.Kind == "SUB" {
		text = strings.TrimSpace(strings.TrimPrefix(text, "->"))
		text = normalizeSubstitutionText(text)
	}
	if event.Kind == "GOAL" {
		text = strings.ReplaceAll(text, "(k)", "(pen)")
	}
	if event.Kind == "MISS" {
		text = strings.ReplaceAll(text, "(nk)", "(pen)")
	}

	return canonicalPlayerName(text)
}

func normalizeSubstitutionText(text string) string {
	cleaned := normalizeDisplayText(text)
	if cleaned == "" {
		return ""
	}
	return playerNumberSuffixRe.ReplaceAllString(cleaned, "")
}

func normalizeDisplayText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func translatePolishDateText(value string) string {
	return polishMonthReplacer.Replace(normalizeDisplayText(value))
}

func formatFixtureWhenInfo(value string) string {
	cleaned := normalizeDisplayText(value)
	if cleaned == "" {
		return ""
	}

	if matches := fixtureWhenInfoRe.FindStringSubmatch(cleaned); len(matches) == 4 {
		if month := polishMonthNumber(matches[2]); month != "" {
			return fmt.Sprintf("%02s/%s %s", matches[1], month, matches[3])
		}
	}

	if idx := strings.Index(cleaned, "("); idx > 0 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}

	return translatePolishDateText(cleaned)
}

func formatFixtureDateTime(value, leagueTitle string) string {
	cleaned := normalizeDisplayText(value)
	if cleaned == "" {
		return ""
	}

	matches := fixtureDateTimeRe.FindStringSubmatch(cleaned)
	if len(matches) != 5 {
		return formatFixtureWhenInfo(cleaned)
	}

	month := polishMonthNumber(matches[2])
	if month == "" {
		return formatFixtureWhenInfo(cleaned)
	}

	day := atoiOrNeg(matches[1])
	monthNum := atoiOrNeg(month)
	year := atoiOrNeg(matches[3])
	if year < 0 {
		year = inferFixtureYear(monthNum, leagueTitle)
	}
	if day <= 0 || monthNum <= 0 || year <= 0 {
		return fmt.Sprintf("%02d/%02d %s", max(0, day), max(0, monthNum), matches[4])
	}

	clock, err := time.Parse("15:04", matches[4])
	if err != nil {
		return fmt.Sprintf("%02d/%02d %s", day, monthNum, matches[4])
	}
	when := time.Date(year, time.Month(monthNum), day, clock.Hour(), clock.Minute(), 0, 0, time.Local)
	return when.Format("Mon 02/01 15:04")
}

func inferFixtureYear(month int, leagueTitle string) int {
	matches := leagueSeasonYearsRe.FindStringSubmatch(leagueTitle)
	if len(matches) != 3 {
		return 0
	}
	startYear := atoiOrNeg(matches[1])
	endYear := atoiOrNeg(matches[2])
	if endYear >= 0 && endYear < 100 {
		endYear += (startYear / 100) * 100
		if endYear < startYear {
			endYear += 100
		}
	}
	if month >= 7 {
		return startYear
	}
	return endYear
}

func polishMonthNumber(value string) string {
	switch strings.ToLower(normalizeDisplayText(value)) {
	case "stycznia":
		return "01"
	case "lutego":
		return "02"
	case "marca":
		return "03"
	case "kwietnia":
		return "04"
	case "maja":
		return "05"
	case "czerwca":
		return "06"
	case "lipca":
		return "07"
	case "sierpnia":
		return "08"
	case "wrzesnia", "września":
		return "09"
	case "pazdziernika", "października":
		return "10"
	case "listopada":
		return "11"
	case "grudnia":
		return "12"
	default:
		return ""
	}
}

func translatePolishStageLabel(value string) string {
	cleaned := translatePolishDateText(value)
	replacements := []struct{ old, new string }{
		{"1/16 finału", "Round of 32"},
		{"1/8 finału", "Round of 16"},
		{"1/4 finału", "Quarter-finals"},
		{"1/2 finału", "Semi-finals"},
		{"finał", "Final"},
		{"baraże", "Play-offs"},
		{"baraż", "Play-off"},
	}
	for _, replacement := range replacements {
		cleaned = strings.ReplaceAll(cleaned, replacement.old, replacement.new)
	}

	return cleaned
}
