package site

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var spaceRe = regexp.MustCompile(`\s+`)
var minuteAtEndRe = regexp.MustCompile(`(\d{1,3})\s*$`)
var minuteAnywhereRe = regexp.MustCompile(`(\d{1,3}(?:\+\d{1,2})?)`)

func parseLeaguePage(doc *goquery.Document, url string) *LeaguePage {
	page := &LeaguePage{URL: url}

	page.Title = strings.TrimSpace(doc.Find("title").First().Text())
	if page.Title == "" {
		page.Title = strings.TrimSpace(doc.Find("td.main b").First().Text())
	}

	rounds := parseRounds(doc)
	if len(rounds) == 0 {
		return nil
	}

	page.Rounds = rounds
	return page
}

func parseMatchPage(doc *goquery.Document, url string) *MatchPage {
	title := normalizeWhitespace(doc.Find("title").First().Text())
	if title == "" {
		return nil
	}

	table := doc.Find("table.main[width='480']").First()
	if table.Length() == 0 {
		return &MatchPage{Title: title, URL: url}
	}

	page := &MatchPage{Title: title, URL: url}

	page.Competition = normalizeWhitespace(table.Find("tr").First().Find("b").First().Text())
	metaCell := table.Find("tr").Eq(1).Find("td[colspan='3']").First()
	page.Meta = normalizeWhitespace(metaCell.Text())

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

	table.Find("tr[bgcolor]").Each(func(_ int, row *goquery.Selection) {
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

func extractMinute(text string) string {
	matches := minuteAnywhereRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func parseRounds(doc *goquery.Document) []Round {
	rounds := make([]Round, 0, 16)
	currentName := ""

	doc.Find("table.main[width='600']").Each(func(_ int, table *goquery.Selection) {
		text := normalizeWhitespace(table.Text())
		if strings.Contains(strings.ToLower(text), "kolejka") {
			name := strings.TrimSpace(table.Find("u").First().Text())
			if name == "" {
				name = text
			}
			currentName = name
			return
		}

		fixtures := parseFixturesTable(table)
		if len(fixtures) == 0 {
			return
		}

		roundName := currentName
		if roundName == "" {
			roundName = "Wyniki"
		}

		rounds = append(rounds, Round{Name: roundName, Fixtures: fixtures})
	})

	return rounds
}

func parseFixturesTable(table *goquery.Selection) []Fixture {
	fixtures := make([]Fixture, 0, 16)

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() < 3 {
			return
		}

		scoreCell := tds.Eq(1)
		matchLink := strings.TrimSpace(scoreCell.Find("a").AttrOr("href", ""))
		score := normalizeWhitespace(scoreCell.Text())
		home := normalizeWhitespace(tds.Eq(0).Text())
		away := normalizeWhitespace(tds.Eq(2).Text())
		whenInfo := ""
		if tds.Length() > 3 {
			whenInfo = normalizeWhitespace(tds.Eq(3).Text())
		}

		if home == "" || away == "" || score == "" || !strings.Contains(matchLink, "mecz.php") {
			return
		}

		fixtures = append(fixtures, Fixture{
			Home:     home,
			Away:     away,
			Score:    score,
			WhenInfo: whenInfo,
			MatchURL: matchLink,
		})
	})

	return fixtures
}

func normalizeWhitespace(v string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(v, " "))
}
