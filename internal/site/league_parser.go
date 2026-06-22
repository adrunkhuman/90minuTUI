package site

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// League layouts drift across seasons, so round parsing keys off headings and
// `mecz.php` links, not table shape.
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

	page.Rounds = normalizeLeagueOrder(rounds, page.Title)
	return page
}

func parseRounds(doc *goquery.Document) []Round {
	rounds := make([]Round, 0, 16)
	currentName := ""
	currentPhase := RoundPhaseRegular
	currentSection := ""
	sectionOneShot := false
	hasNamedRounds := false

	processHeading := func(name string) {
		if looksLikeSectionHeading(name) {
			// Some pages use a standalone group heading for one fixture table; others keep it for repeated matchdays.
			currentName = ""
			currentSection = name
			currentPhase = RoundPhaseGroup
			sectionOneShot = true
			return
		}

		currentName = name
		matchdayHeading := looksLikeMatchdayHeading(name)
		if currentSection != "" && !matchdayHeading && !sectionOneShot {
			currentSection = ""
		}
		currentPhase = classifyRoundPhase(name, currentSection)
		if currentSection != "" && matchdayHeading {
			sectionOneShot = false
		}
		if currentSection != "" && !matchdayHeading {
			sectionOneShot = true
		}
	}

	leagueTables(doc).Each(func(_ int, table *goquery.Selection) {
		if names := roundNamesFromTable(table); len(names) > 0 {
			for _, name := range names {
				processHeading(name)
			}
			hasNamedRounds = true
			return
		}

		if table.Find("table").Length() > 0 {
			return
		}

		fixtures := parseFixturesTable(table)
		if len(fixtures) == 0 {
			return
		}

		roundName := strings.TrimSpace(currentName)
		round := Round{Phase: currentPhase, Section: currentSection, Name: roundName, Fixtures: fixtures}

		if len(rounds) > 0 && sameRoundContext(rounds[len(rounds)-1], round) {
			// 90minut sometimes splits one round across adjacent tables under the same heading.
			rounds[len(rounds)-1].Fixtures = append(rounds[len(rounds)-1].Fixtures, fixtures...)
			if sectionOneShot {
				currentSection = ""
				sectionOneShot = false
			}
			return
		}

		rounds = append(rounds, round)
		if sectionOneShot {
			currentSection = ""
			sectionOneShot = false
		}
	})

	if hasNamedRounds {
		// If headings exist, unnamed fixture tables here are usually standings/nav spillover,
		// not separate rounds.
		namedOnly := make([]Round, 0, len(rounds))
		for _, round := range rounds {
			if strings.TrimSpace(round.Name) == "" && strings.TrimSpace(round.Section) == "" {
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
		if strings.TrimSpace(rounds[i].Section) != "" {
			continue
		}
		// Headerless fixture pages still need a stable round label for downstream rendering.
		rounds[i].Name = "Wyniki"
	}

	return rounds
}

func sameRoundContext(a, b Round) bool {
	return a.Name == b.Name && a.Section == b.Section && a.Phase == b.Phase
}

func roundNamesFromTable(table *goquery.Selection) []string {
	if table.Find("a[href*='mecz.php']").Length() > 0 || len(parseFixturesTable(table)) > 0 {
		return nil
	}

	var names []string
	table.Find("u").Each(func(_ int, s *goquery.Selection) {
		heading := normalizeWhitespace(s.Text())
		if looksLikeRoundHeading(heading) || looksLikeStageHeading(heading) || looksLikeSectionHeading(heading) {
			names = append(names, heading)
		}
	})
	if len(names) > 0 {
		return names
	}

	if !looksLikeStandaloneHeadingTable(table) {
		return nil
	}

	text := normalizeWhitespace(table.ChildrenFiltered("tbody, tr").Text())
	if looksLikeRoundHeading(text) || looksLikeStageHeading(text) || looksLikeSectionHeading(text) {
		return []string{text}
	}

	return nil
}

func leagueTables(doc *goquery.Document) *goquery.Selection {
	mainCell := doc.Find("td.main[align='center']").First()
	if mainCell.Length() > 0 {
		return mainCell.Find("table")
	}

	return doc.Find("table")
}

func roundNameFromTable(table *goquery.Selection) (string, bool) {
	names := roundNamesFromTable(table)
	if len(names) == 0 {
		return "", false
	}
	return names[len(names)-1], true
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
	return looksLikeMatchdayHeading(lower) || strings.Contains(lower, "runda")
}

func looksLikeMatchdayHeading(text string) bool {
	return strings.Contains(strings.ToLower(text), "kolejka")
}

func looksLikeStageHeading(text string) bool {
	if text == "" {
		return false
	}

	lower := strings.ToLower(text)
	return strings.Contains(lower, "fina") || strings.Contains(lower, "bara") || strings.Contains(lower, "play-off") || strings.Contains(lower, "playoff")
}

func looksLikeSectionHeading(text string) bool {
	fields := strings.Fields(normalizeWhitespace(text))
	if len(fields) < 2 || !strings.EqualFold(fields[0], "grupa") {
		return false
	}

	lower := strings.ToLower(text)
	return !strings.Contains(lower, "kolejka") && !strings.Contains(lower, "runda")
}

func classifyRoundPhase(name, section string) RoundPhase {
	lower := strings.ToLower(normalizeWhitespace(name))
	switch {
	case strings.Contains(lower, "elimin"):
		return RoundPhaseQualification
	case strings.Contains(lower, "przedwst") || strings.Contains(lower, "wst"):
		return RoundPhasePreliminary
	case strings.Contains(lower, "fina") || strings.Contains(lower, "bara") || strings.Contains(lower, "play-off") || strings.Contains(lower, "playoff"):
		return RoundPhaseKnockout
	case section != "" && strings.Contains(lower, "kolejka"):
		return RoundPhaseGroup
	default:
		return RoundPhaseRegular
	}
}

func parseFixturesTable(table *goquery.Selection) []Fixture {
	fixtures := make([]Fixture, 0, 16)

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		fixture, ok := parseFixtureRow(row)
		if !ok {
			return
		}

		fixtures = append(fixtures, fixture)
	})

	return fixtures
}

func parseFixtureRow(row *goquery.Selection) (Fixture, bool) {
	tds := row.Find("td")
	if tds.Length() < 3 {
		return Fixture{}, false
	}

	scoreCell, scoreIdx, matchLink, ok := fixtureScoreCell(row, tds)
	if !ok {
		return Fixture{}, false
	}

	home, _ := nearestTeamCellText(tds, scoreIdx-1, -1)
	away, awayIdx := nearestTeamCellText(tds, scoreIdx+1, 1)
	if home == "" || away == "" || isScoreLikeText(home) || isScoreLikeText(away) {
		return Fixture{}, false
	}

	score := normalizeWhitespace(scoreCell.Text())
	if !isFixtureScoreText(score) {
		return Fixture{}, false
	}

	return Fixture{
		Home:     home,
		Away:     away,
		Score:    score,
		WhenInfo: joinNonEmptyCells(tds, awayIdx+1),
		MatchURL: matchLink,
		MatchID:  extractMatchID(matchLink),
	}, true
}

// Some result tables omit `mecz.php` links entirely, so the fallback accepts
// only one unambiguous score-like cell with team-like neighbors and drops rows
// that still look ambiguous.
func fixtureScoreCell(row *goquery.Selection, tds *goquery.Selection) (*goquery.Selection, int, string, bool) {
	matchLinks := row.Find("a[href*='mecz.php']")
	if matchLinks.Length() > 1 {
		return nil, -1, "", false
	}
	if matchLinks.Length() == 1 {
		scoreCell := matchLinks.First().Closest("td")
		if scoreCell.Length() == 0 {
			return nil, -1, "", false
		}

		matchLink := strings.TrimSpace(matchLinks.First().AttrOr("href", ""))
		if matchLink == "" {
			return nil, -1, "", false
		}

		return scoreCell, scoreCell.Index(), matchLink, true
	}

	scoreIdx := -1
	for idx := 0; idx < tds.Length(); idx++ {
		if !isFixtureScoreText(tds.Eq(idx).Text()) {
			continue
		}
		home, _ := nearestTeamCellText(tds, idx-1, -1)
		away, _ := nearestTeamCellText(tds, idx+1, 1)
		if home == "" || away == "" || isScoreLikeText(home) || isScoreLikeText(away) {
			continue
		}
		if scoreIdx >= 0 {
			return nil, -1, "", false
		}
		scoreIdx = idx
	}
	if scoreIdx < 0 {
		return nil, -1, "", false
	}

	return tds.Eq(scoreIdx), scoreIdx, "", true
}

func isFixtureScoreText(text string) bool {
	cleaned := normalizeWhitespace(text)
	return cleaned == "-" || isScoreLikeText(cleaned)
}

func parseStandings(doc *goquery.Document) []StandingRow {
	var standings []StandingRow

	// Standings tables drift too; take the first table with classic league headers
	// and stop when rows stop parsing.
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
