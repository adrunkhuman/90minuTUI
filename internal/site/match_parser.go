package site

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// parseMatchPage parses a single match page into score, timeline and lineups.
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

		if strings.Contains(middle, "-") {
			if left != "" && right == "" {
				page.Events = append(page.Events, MatchEvent{
					MinuteText: extractMinute(left),
					Kind:       "GOAL",
					TeamSide:   "home",
					Text:       left,
				})
			}

			if right != "" && left == "" {
				page.Events = append(page.Events, MatchEvent{
					MinuteText: extractMinute(right),
					Kind:       "GOAL",
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

		if player := parsePlayerCell(tds.Eq(0)); player != nil {
			page.HomeLineup = append(page.HomeLineup, *player)
			for _, event := range playerTimelineEvents(*player, "home") {
				page.Events = append(page.Events, event)
			}
		}
		if player := parsePlayerCell(tds.Eq(2)); player != nil {
			page.AwayLineup = append(page.AwayLineup, *player)
			for _, event := range playerTimelineEvents(*player, "away") {
				page.Events = append(page.Events, event)
			}
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

func findMatchMainTable(doc *goquery.Document) *goquery.Selection {
	bestScore := -1
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

		meta = text
		return false
	})

	return meta
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
			event.Text = marker
		default:
			event.Kind = "EVENT"
			event.Text = marker
		}

		events = append(events, event)
	}

	return events
}

func parsePlayerCell(cell *goquery.Selection) *PlayerLine {
	raw := normalizeWhitespace(cell.Text())
	if raw == "" {
		return nil
	}

	anchors := make([]string, 0, 3)
	cell.Find("a").Each(func(_ int, a *goquery.Selection) {
		name := normalizeWhitespace(a.Text())
		if name != "" {
			anchors = append(anchors, name)
		}
	})

	name := raw
	if len(anchors) > 0 {
		name = anchors[0]
	}

	events := make([]string, 0, 3)
	if cell.Find("img[src*='yel.gif']").Length() > 0 {
		events = append(events, "YC")
	}
	if cell.Find("img[src*='red.gif'], img[src*='red2.gif']").Length() > 0 {
		events = append(events, "RC")
	}

	if cell.Find("img[src*='sub.gif']").Length() > 0 && len(anchors) > 1 {
		replacement := anchors[len(anchors)-1]
		minute := substitutionMinute(raw, replacement)
		if minute != "" {
			events = append(events, fmt.Sprintf("%s' -> %s", minute, replacement))
		} else {
			events = append(events, fmt.Sprintf("sub -> %s", replacement))
		}
	}

	return &PlayerLine{Name: name, Events: events, RawText: raw}
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
