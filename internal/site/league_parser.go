package site

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// parseLeaguePage parses a competition page into rounds and fixtures.
// It prefers structural markers and match-link semantics over fixed table widths.
func parseLeaguePage(doc *goquery.Document, url string) *LeaguePage {
	page := &LeaguePage{URL: url, LeagueKey: extractLeagueKey(url)}

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

func parseRounds(doc *goquery.Document) []Round {
	rounds := make([]Round, 0, 16)
	currentName := ""

	doc.Find("table").Each(func(_ int, table *goquery.Selection) {
		if name, ok := roundNameFromTable(table); ok {
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

		if len(rounds) > 0 && rounds[len(rounds)-1].Name == roundName {
			rounds[len(rounds)-1].Fixtures = append(rounds[len(rounds)-1].Fixtures, fixtures...)
			return
		}

		rounds = append(rounds, Round{Name: roundName, Fixtures: fixtures})
	})

	return rounds
}

func roundNameFromTable(table *goquery.Selection) (string, bool) {
	if table.Find("a[href*='mecz.php']").Length() > 0 {
		return "", false
	}

	heading := normalizeWhitespace(table.Find("u").First().Text())
	if looksLikeRoundHeading(heading) {
		return heading, true
	}

	text := normalizeWhitespace(table.Text())
	if looksLikeRoundHeading(text) {
		return text, true
	}

	return "", false
}

func looksLikeRoundHeading(text string) bool {
	if text == "" {
		return false
	}

	lower := strings.ToLower(text)
	return strings.Contains(lower, "kolejka") || strings.Contains(lower, "runda")
}

func parseFixturesTable(table *goquery.Selection) []Fixture {
	fixtures := make([]Fixture, 0, 16)

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		matchLinks := row.Find("a[href*='mecz.php']")
		if matchLinks.Length() != 1 {
			return
		}

		scoreCell := matchLinks.First().Closest("td")
		if scoreCell.Length() == 0 {
			return
		}

		tds := row.Find("td")
		scoreIdx := scoreCell.Index()
		home, _ := nearestTeamCellText(tds, scoreIdx-1, -1)
		away, awayIdx := nearestTeamCellText(tds, scoreIdx+1, 1)
		if home == "" || away == "" || isScoreLikeText(home) || isScoreLikeText(away) {
			return
		}

		matchLink := strings.TrimSpace(scoreCell.Find("a[href*='mecz.php']").First().AttrOr("href", ""))
		score := normalizeWhitespace(scoreCell.Text())
		whenInfo := joinNonEmptyCells(tds, awayIdx+1)

		if score == "" || matchLink == "" {
			return
		}

		fixtures = append(fixtures, Fixture{
			Home:     home,
			Away:     away,
			Score:    score,
			WhenInfo: whenInfo,
			MatchURL: matchLink,
			MatchID:  extractMatchID(matchLink),
		})
	})

	return fixtures
}

func nearestTeamCellText(tds *goquery.Selection, start, step int) (string, int) {
	for idx := start; idx >= 0 && idx < tds.Length(); idx += step {
		text := normalizeWhitespace(tds.Eq(idx).Text())
		if text != "" {
			return text, idx
		}
	}

	return "", -1
}

func joinNonEmptyCells(tds *goquery.Selection, start int) string {
	if start < 0 || start >= tds.Length() {
		return ""
	}

	parts := make([]string, 0, 2)
	for idx := start; idx < tds.Length(); idx++ {
		value := normalizeWhitespace(tds.Eq(idx).Text())
		if value != "" {
			parts = append(parts, value)
		}
	}

	return strings.Join(parts, " ")
}
