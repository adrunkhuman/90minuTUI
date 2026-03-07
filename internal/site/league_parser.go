package site

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var spaceRe = regexp.MustCompile(`\s+`)

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

	page.LatestRound = rounds[len(rounds)-1]
	return page
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
