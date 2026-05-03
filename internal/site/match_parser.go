package site

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var matchMetaLineRe = regexp.MustCompile(`\d{1,2}\s+\p{L}+.*\d{1,2}:\d{2}`)

// It identifies the match table by content signatures instead of a fixed width.
func parseMatchPage(doc *goquery.Document, url string) *MatchPage {
	title := normalizeWhitespace(doc.Find("title").First().Text())
	if title == "" {
		return nil
	}

	table := findMatchMainTable(doc)
	if table.Length() == 0 {
		return &MatchPage{Title: title, URL: url, MatchID: extractMatchID(url)}
	}

	page := &MatchPage{Title: title, URL: url, MatchID: extractMatchID(url)}

	page.Competition = normalizeWhitespace(table.Find("tr").First().Find("b").First().Text())
	page.Meta = firstMetaLine(table)
	page.Weather = normalizeWhitespace(table.Find("img[src*='pog_termo']").Parent().Text())

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() != 3 {
			return
		}

		left := normalizeWhitespace(tds.Eq(0).Text())
		middle := normalizeWhitespace(tds.Eq(1).Text())
		right := normalizeWhitespace(tds.Eq(2).Text())

		if page.Score == "" && left != "" && right != "" && strings.Contains(middle, "-") {
			page.HomeTeam = left
			page.AwayTeam = right
			page.Score = middle
			return
		}

		if kind, ok := scoreRowEventKind(row); ok {
			// Incident rows keep one side empty; non-empty side maps event ownership.
			if left != "" && right == "" {
				minuteText := extractMinute(left)
				m, s, ok := parseMinute(minuteText)
				page.Events = append(page.Events, MatchEvent{
					MinuteText: minuteText,
					Minute:     m,
					Stoppage:   s,
					HasMinute:  ok,
					Kind:       kind,
					TeamSide:   "home",
					Text:       left,
				})
			}

			if right != "" && left == "" {
				minuteText := extractMinute(right)
				m, s, ok := parseMinute(minuteText)
				page.Events = append(page.Events, MatchEvent{
					MinuteText: minuteText,
					Minute:     m,
					Stoppage:   s,
					HasMinute:  ok,
					Kind:       kind,
					TeamSide:   "away",
					Text:       right,
				})
			}
		}
	})

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		if !isLineupRow(row) {
			return
		}

		tds := row.Find("td")
		if tds.Length() != 3 {
			return
		}

		if parsed := parsePlayerCell(tds.Eq(0), "home"); parsed != nil {
			page.HomeLineup = append(page.HomeLineup, *parsed.Player)
			for _, event := range playerTimelineEvents(*parsed.Player, "home") {
				page.Events = append(page.Events, event)
			}
			page.Events = append(page.Events, parsed.ExtraEvents...)
		}
		if parsed := parsePlayerCell(tds.Eq(2), "away"); parsed != nil {
			page.AwayLineup = append(page.AwayLineup, *parsed.Player)
			for _, event := range playerTimelineEvents(*parsed.Player, "away") {
				page.Events = append(page.Events, event)
			}
			page.Events = append(page.Events, parsed.ExtraEvents...)
		}
	})

	table.Find("tr td[colspan='3']").Each(func(_ int, td *goquery.Selection) {
		text := normalizeWhitespace(td.Text())
		if !strings.Contains(strings.ToLower(text), "przeczytaj news") {
			return
		}

		page.NewsTitle = normalizeWhitespace(td.Find("a").First().Text())
		page.NewsURL = strings.TrimSpace(td.Find("a").First().AttrOr("href", ""))
	})

	return page
}

func scoreRowEventKind(row *goquery.Selection) (string, bool) {
	tds := row.Find("td")
	if tds.Length() != 3 {
		return "", false
	}
	middle := normalizeWhitespace(tds.Eq(1).Text())
	if middle == "-" || isScoreLikeText(middle) {
		return "GOAL", true
	}

	leftKind := scoreCellEventKind(tds.Eq(0))
	rightKind := scoreCellEventKind(tds.Eq(2))
	if leftKind != "" && leftKind == rightKind {
		return leftKind, true
	}
	if leftKind != "" && normalizeWhitespace(tds.Eq(2).Text()) == "" {
		return leftKind, true
	}
	if rightKind != "" && normalizeWhitespace(tds.Eq(0).Text()) == "" {
		return rightKind, true
	}

	return "", false
}

func scoreCellEventKind(cell *goquery.Selection) string {
	switch {
	case cell.Find("img[src*='goal.gif']").Length() > 0:
		return "GOAL"
	case cell.Find("img[src*='missed.gif']").Length() > 0:
		return "MISS"
	default:
		return ""
	}
}

func findMatchMainTable(doc *goquery.Document) *goquery.Selection {
	bestScore := -1
	// Preserve legacy fallback for older 90minut pages that still use width=480.
	best := doc.Find("table.main[width='480']").First()

	doc.Find("table").Each(func(_ int, table *goquery.Selection) {
		score := 0
		if hasMatchScoreRow(table) {
			score += 3
		}
		if table.Find("img[src*='pog_termo']").Length() > 0 {
			score++
		}
		if table.Find("img[src*='yel.gif'], img[src*='red.gif'], img[src*='red2.gif'], img[src*='sub.gif']").Length() > 0 {
			score += 2
		}
		if table.Find("tr[bgcolor]").Length() > 0 {
			score++
		}

		if score > bestScore {
			bestScore = score
			best = table
		}
	})

	if best.Length() == 0 || bestScore <= 0 {
		return &goquery.Selection{}
	}

	return best
}

func hasMatchScoreRow(table *goquery.Selection) bool {
	found := false
	table.Find("tr").EachWithBreak(func(_ int, row *goquery.Selection) bool {
		tds := row.Find("td")
		if tds.Length() != 3 {
			return true
		}

		left := normalizeWhitespace(tds.Eq(0).Text())
		middle := normalizeWhitespace(tds.Eq(1).Text())
		right := normalizeWhitespace(tds.Eq(2).Text())
		if left != "" && right != "" && strings.Contains(middle, "-") {
			found = true
			return false
		}

		return true
	})

	return found
}

func firstMetaLine(table *goquery.Selection) string {
	meta := ""
	table.Find("td[colspan='3']").EachWithBreak(func(_ int, td *goquery.Selection) bool {
		text := normalizeWhitespace(td.Text())
		if text == "" {
			return true
		}

		lower := strings.ToLower(text)
		if strings.Contains(lower, "przeczytaj news") {
			return true
		}

		if td.Find("img[src*='pog_termo']").Length() > 0 {
			return true
		}

		if normalizeWhitespace(td.Find("b").First().Text()) == text {
			return true
		}

		if !looksLikeMatchMeta(text) {
			return true
		}

		meta = text
		return false
	})

	return meta
}

func looksLikeMatchMeta(text string) bool {
	cleaned := normalizeWhitespace(text)
	lower := strings.ToLower(cleaned)
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "strona główna") || strings.Contains(lower, "strona g") {
		return false
	}

	return matchMetaLineRe.MatchString(cleaned)
}

func isLineupRow(row *goquery.Selection) bool {
	tds := row.Find("td")
	if tds.Length() != 3 {
		return false
	}

	if row.Find("img[src*='yel.gif'], img[src*='red.gif'], img[src*='red2.gif'], img[src*='sub.gif']").Length() > 0 {
		return true
	}

	if _, ok := row.Attr("bgcolor"); !ok {
		return false
	}

	return tds.Eq(0).Find("a").Length() > 0 || tds.Eq(2).Find("a").Length() > 0
}

func playerTimelineEvents(player PlayerLine, side string) []MatchEvent {
	events := make([]MatchEvent, 0, 2)

	for _, marker := range player.Events {
		event := MatchEvent{TeamSide: side, Text: player.Name}

		switch {
		case marker == "YC":
			event.Kind = "YC"
		case marker == "RC":
			event.Kind = "RC"
		case strings.Contains(marker, "->"):
			event.Kind = "SUB"
			event.MinuteText = extractMinute(marker)
			event.Minute, event.Stoppage, event.HasMinute = parseMinute(event.MinuteText)
			event.Text = substitutionEventText(player.Name, marker)
		default:
			event.Kind = "EVENT"
			event.Text = marker
		}

		events = append(events, event)
	}

	return events
}

func substitutionEventText(outgoing, marker string) string {
	parts := strings.SplitN(marker, "->", 2)
	if len(parts) != 2 {
		return normalizeWhitespace(marker)
	}

	incoming := normalizeWhitespace(parts[1])
	if incoming == "" {
		return normalizeWhitespace(marker)
	}

	outgoing = normalizeWhitespace(outgoing)
	if outgoing == "" {
		return incoming
	}

	return outgoing + " -> " + incoming
}

type parsedPlayerCell struct {
	Player      *PlayerLine
	ExtraEvents []MatchEvent
}

func parsePlayerCell(cell *goquery.Selection, side string) *parsedPlayerCell {
	raw := normalizeWhitespace(cell.Text())
	if raw == "" {
		return nil
	}

	name := raw
	anchors := make([]string, 0, 3)
	markersByPlayer := make(map[string][]string, 2)
	currentPlayer := ""

	for _, node := range cell.Contents().Nodes {
		switch node.Type {
		case html.ElementNode:
			switch strings.ToLower(node.Data) {
			case "a":
				playerName := normalizeWhitespace(goquery.NewDocumentFromNode(node).Text())
				if playerName == "" {
					continue
				}
				anchors = append(anchors, playerName)
				if len(anchors) == 1 {
					name = playerName
				}
				currentPlayer = playerName
			case "img":
				src := ""
				for _, attr := range node.Attr {
					if strings.EqualFold(attr.Key, "src") {
						src = strings.ToLower(strings.TrimSpace(attr.Val))
						break
					}
				}
				switch {
				case strings.Contains(src, "sub.gif"):
					currentPlayer = ""
				case currentPlayer == "":
					continue
				case strings.Contains(src, "yel.gif"):
					markersByPlayer[currentPlayer] = append(markersByPlayer[currentPlayer], "YC")
				case strings.Contains(src, "red.gif"), strings.Contains(src, "red2.gif"):
					markersByPlayer[currentPlayer] = append(markersByPlayer[currentPlayer], "RC")
				}
			}
		}
	}

	events := append([]string(nil), markersByPlayer[name]...)

	if cell.Find("img[src*='sub.gif']").Length() > 0 && len(anchors) > 1 {
		// In 90minut substitution cells, the last linked player is the replacement.
		replacement := anchors[len(anchors)-1]
		minute := substitutionMinute(raw, replacement)
		if minute != "" {
			events = append(events, fmt.Sprintf("%s' -> %s", minute, replacement))
		} else {
			events = append(events, fmt.Sprintf("sub -> %s", replacement))
		}
	}

	extraEvents := make([]MatchEvent, 0, 2)
	for i := 1; i < len(anchors); i++ {
		for _, marker := range markersByPlayer[anchors[i]] {
			extraEvents = append(extraEvents, MatchEvent{
				Kind:     marker,
				TeamSide: side,
				Text:     anchors[i],
			})
		}
	}

	return &parsedPlayerCell{
		Player:      &PlayerLine{Name: name, Events: events, RawText: raw},
		ExtraEvents: extraEvents,
	}
}

func substitutionMinute(raw, replacement string) string {
	idx := strings.Index(raw, replacement)
	if idx <= 0 {
		return ""
	}

	prefix := normalizeWhitespace(raw[:idx])
	matches := minuteAtEndRe.FindStringSubmatch(prefix)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
