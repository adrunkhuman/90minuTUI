package site

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// League pages drift across seasons, so round extraction keys off heading tables
// and `mecz.php` links instead of width or column positions.
func parseLeaguePage(doc *goquery.Document, url string) *LeaguePage {
	page := &LeaguePage{URL: url, LeagueKey: extractLeagueKey(url)}

	page.Title = strings.TrimSpace(doc.Find("title").First().Text())
	if page.Title == "" {
		page.Title = strings.TrimSpace(doc.Find("td.main b").First().Text())
	}
	page.Standings = parseStandings(doc)

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
	hasNamedRounds := false

	leagueTables(doc).Each(func(_ int, table *goquery.Selection) {
		if name, ok := roundNameFromTable(table); ok {
			currentName = name
			hasNamedRounds = true
			return
		}

		fixtures := parseFixturesTable(table)
		if len(fixtures) == 0 {
			return
		}

		roundName := strings.TrimSpace(currentName)

		if len(rounds) > 0 && rounds[len(rounds)-1].Name == roundName {
			// Source HTML may split one round into adjacent tables with the same heading.
			rounds[len(rounds)-1].Fixtures = append(rounds[len(rounds)-1].Fixtures, fixtures...)
			return
		}

		rounds = append(rounds, Round{Name: roundName, Fixtures: fixtures})
	})

	if hasNamedRounds {
		// Once explicit round headings exist, unnamed fixture tables in the same area
		// are usually standings/nav spillover rather than standalone rounds.
		namedOnly := make([]Round, 0, len(rounds))
		for _, round := range rounds {
			if strings.TrimSpace(round.Name) == "" {
				continue
			}
			namedOnly = append(namedOnly, round)
		}
		if len(namedOnly) > 0 {
			rounds = namedOnly
		}
	}

	for i := range rounds {
		if strings.TrimSpace(rounds[i].Name) != "" {
			continue
		}
		// Fully headerless fixture pages are normalized to a single fallback round.
		rounds[i].Name = "Wyniki"
	}

	return rounds
}

func leagueTables(doc *goquery.Document) *goquery.Selection {
	mainCell := doc.Find("td.main[align='center']").First()
	if mainCell.Length() > 0 {
		return mainCell.Find("table")
	}

	return doc.Find("table")
}

func roundNameFromTable(table *goquery.Selection) (string, bool) {
	if table.Find("a[href*='mecz.php']").Length() > 0 {
		return "", false
	}

	heading := normalizeWhitespace(table.Find("u").First().Text())
	if looksLikeRoundHeading(heading) {
		return heading, true
	}

	if !looksLikeStandaloneHeadingTable(table) {
		return "", false
	}

	text := normalizeWhitespace(table.ChildrenFiltered("tbody, tr").Text())
	if looksLikeRoundHeading(text) {
		return text, true
	}

	return "", false
}

func looksLikeStandaloneHeadingTable(table *goquery.Selection) bool {
	rows := table.ChildrenFiltered("tbody, tr").ChildrenFiltered("tr")
	if rows.Length() == 0 {
		rows = table.ChildrenFiltered("tr")
	}
	if rows.Length() != 1 {
		return false
	}

	tds := rows.First().ChildrenFiltered("td, th")
	if tds.Length() != 1 {
		return false
	}

	text := normalizeWhitespace(tds.First().Text())
	if text == "" {
		return false
	}

	lower := strings.ToLower(text)
	if strings.Contains(lower, "wyniki") || strings.Contains(lower, "strzelcy") || strings.Contains(lower, "statystyki") || strings.Contains(lower, "ostatnia kolejka") {
		return false
	}

	return true
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

func parseStandings(doc *goquery.Document) []StandingRow {
	var standings []StandingRow

	// Standings pages drift too, so stop at the first table whose header matches
	// the classic league columns and stop once parsed rows give way to other content.
	leagueTables(doc).EachWithBreak(func(_ int, table *goquery.Selection) bool {
		header := table.Find("tr").FilterFunction(func(_ int, row *goquery.Selection) bool {
			text := normalizeWhitespace(row.Text())
			return strings.Contains(text, "Nazwa") && strings.Contains(text, "Pkt.")
		}).First()
		if header.Length() == 0 {
			return true
		}

		headerFound := false
		parsed := make([]StandingRow, 0, 18)
		table.Find("tr").EachWithBreak(func(_ int, row *goquery.Selection) bool {
			if !headerFound {
				if row.IsSelection(header) {
					headerFound = true
				}
				return true
			}

			standing, ok := parseStandingRow(row)
			if !ok {
				return len(parsed) == 0
			}

			parsed = append(parsed, standing)
			return true
		})

		if len(parsed) > 0 {
			standings = parsed
			return false
		}

		return true
	})

	return standings
}

func parseStandingRow(row *goquery.Selection) (StandingRow, bool) {
	tds := row.Find("td")
	if tds.Length() < 7 {
		return StandingRow{}, false
	}

	position := parseIntCell(strings.TrimSuffix(normalizeWhitespace(tds.Eq(0).Text()), "."))
	team := normalizeWhitespace(tds.Eq(1).Text())
	played := parseIntCell(tds.Eq(2).Text())
	points := parseIntCell(tds.Eq(3).Text())
	won := parseIntCell(tds.Eq(4).Text())
	drawn := parseIntCell(tds.Eq(5).Text())
	lost := parseIntCell(tds.Eq(6).Text())

	if position <= 0 || team == "" || played < 0 || points < 0 || won < 0 || drawn < 0 || lost < 0 {
		return StandingRow{}, false
	}

	return StandingRow{
		Position: position,
		Team:     team,
		Played:   played,
		Won:      won,
		Drawn:    drawn,
		Lost:     lost,
		Points:   points,
	}, true
}

func parseIntCell(text string) int {
	value := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(text), "%d", &value); err != nil {
		return -1
	}
	return value
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
