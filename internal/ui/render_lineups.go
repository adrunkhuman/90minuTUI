package ui

import (
	"image/color"
	"strings"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/charmbracelet/x/ansi"
)

type lineupCardMarker struct {
	color color.Color
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

// A substitute can enter and later leave; keep both notes on one rendered row.
func formatLineupPlayer(entry lineupEntry, side site.TeamSide, maxWidth int) string {
	return formatLineupPlayerWithCards(entry, side, maxWidth, false)
}

func formatLineupPlayerWithCards(entry lineupEntry, side site.TeamSide, maxWidth int, tokens bool) string {
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

func lineupPlayerLabel(entry lineupEntry, side site.TeamSide, name string, shortenNotes, tokens bool) string {
	notes := lineupNotes(entry, side, shortenNotes, tokens)
	if len(notes) == 0 {
		return name
	}

	if side == site.TeamSideHome {
		parts := append(notes, name)
		return strings.Join(parts, " ")
	}

	parts := append([]string{name}, notes...)
	return strings.Join(parts, " ")
}

func lineupNotes(entry lineupEntry, side site.TeamSide, shortenNotes, tokens bool) []string {
	notes := make([]string, 0, 2)

	if note := entryNote(entry, side, shortenNotes, tokens); note != "" {
		notes = append(notes, note)
	}
	if note := exitNote(entry, side, shortenNotes, tokens); note != "" {
		notes = append(notes, note)
	}

	return notes
}

func entryNote(entry lineupEntry, side site.TeamSide, shortenNotes, tokens bool) string {
	if entry.enteredAt == "" {
		return ""
	}

	replaced := formatSubNoteName(entry.replaced, shortenNotes)
	card := lineupCardText(entry.replacedYC, tokens)
	if side == site.TeamSideHome {
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

func exitNote(entry lineupEntry, side site.TeamSide, shortenNotes, tokens bool) string {
	if entry.replacedBy == "" {
		return ""
	}

	replacement := formatSubNoteName(entry.replacedBy, shortenNotes)
	card := lineupCardText(entry.replacedByYC, tokens)
	text := "("
	if side == site.TeamSideHome && entry.leftAt != "" {
		text += entry.leftAt + " "
	}
	if side != site.TeamSideHome && card != "" {
		text += card + " "
	}
	text += replacement
	if side == site.TeamSideHome {
		text += card
	}
	if side != site.TeamSideHome && entry.leftAt != "" {
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

func annotateLineupPlayer(player site.PlayerLine, idx map[string][]site.MatchEvent) lineupEntry {
	return annotateLineupPlayerInRoster(player, idx, nil)
}

func annotateLineupPlayerInRoster(player site.PlayerLine, idx map[string][]site.MatchEvent, players []site.PlayerLine) lineupEntry {
	entry := lineupEntry{player: player}
	for _, event := range sortedEvents(matchingPlayerEventsInRoster(player.Name, idx, players)) {
		if event.Kind != site.EventKindSubstitution {
			continue
		}

		out, in := substitutionPlayers(event)
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
	// Abbreviated names are trusted only when the roster has one compact match; ambiguous initials stay unannotated.
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

// Only synthesize rows for substitutes who were later subbed off; avoid inventing one-off bench entrants absent from the source lineup.
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
