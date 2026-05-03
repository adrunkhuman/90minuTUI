package site

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var roundNumberRe = regexp.MustCompile(`\d+`)
var fixtureDateRe = regexp.MustCompile(`(?i)(\d{1,2})\s+([\p{L}]+)(?:\s+(\d{4}))?(?:,\s*(\d{1,2}):(\d{2}))?`)
var seasonYearsRe = regexp.MustCompile(`(\d{4})\s*/\s*(\d{2,4})`)

// Normalize round and fixture order here so every caller sees the same league structure.
func normalizeLeagueOrder(rounds []Round, leagueTitle string) []Round {
	ordered := make([]Round, len(rounds))
	copy(ordered, rounds)

	for i := range ordered {
		ordered[i].Fixtures = normalizeFixturesByDate(ordered[i].Fixtures, leagueTitle)
	}

	type indexedRound struct {
		round Round
		index int
	}

	indexed := make([]indexedRound, len(ordered))
	for i, round := range ordered {
		indexed[i] = indexedRound{round: round, index: i}
	}

	sort.SliceStable(indexed, func(i, j int) bool {
		numberI, okI := roundNumber(indexed[i].round.Name)
		numberJ, okJ := roundNumber(indexed[j].round.Name)

		switch {
		case okI && okJ && numberI != numberJ:
			return numberI < numberJ
		case okI != okJ:
			return okI
		default:
			return indexed[i].index < indexed[j].index
		}
	})

	for i := range indexed {
		ordered[i] = indexed[i].round
	}

	return ordered
}

func normalizeFixturesByDate(fixtures []Fixture, leagueTitle string) []Fixture {
	ordered := make([]Fixture, len(fixtures))
	copy(ordered, fixtures)

	type indexedFixture struct {
		fixture Fixture
		index   int
	}

	indexed := make([]indexedFixture, len(ordered))
	for i, fixture := range ordered {
		indexed[i] = indexedFixture{fixture: fixture, index: i}
	}

	// Sort parsable dates first; keep undated fixtures in source order at the end.
	sort.SliceStable(indexed, func(i, j int) bool {
		dateI, okI := fixtureDateKey(indexed[i].fixture.WhenInfo, leagueTitle)
		dateJ, okJ := fixtureDateKey(indexed[j].fixture.WhenInfo, leagueTitle)

		switch {
		case okI && okJ && dateI != dateJ:
			return dateI < dateJ
		case okI != okJ:
			return okI
		default:
			return indexed[i].index < indexed[j].index
		}
	})

	for i := range indexed {
		ordered[i] = indexed[i].fixture
	}

	return ordered
}

func roundNumber(name string) (int, bool) {
	lower := strings.ToLower(normalizeWhitespace(name))
	if !strings.Contains(lower, "kolejka") && !strings.Contains(lower, "runda") {
		return 0, false
	}

	match := roundNumberRe.FindString(lower)
	if match == "" {
		return 0, false
	}

	value, err := strconv.Atoi(match)
	if err != nil {
		return 0, false
	}

	return value, true
}

func fixtureDateKey(whenInfo, leagueTitle string) (int, bool) {
	matches := fixtureDateRe.FindStringSubmatch(normalizeWhitespace(whenInfo))
	if len(matches) == 0 {
		return 0, false
	}

	day, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}

	month, ok := polishMonthIndex(matches[2])
	if !ok {
		return 0, false
	}

	year := 0
	if matches[3] != "" {
		year, err = strconv.Atoi(matches[3])
		if err != nil {
			return 0, false
		}
	} else {
		year = inferFixtureYear(month, leagueTitle)
		// Pages without a season in the title keep month/day ordering because no season boundary is available.
	}

	hour := 0
	minute := 0
	if matches[4] != "" {
		hour, err = strconv.Atoi(matches[4])
		if err != nil {
			return 0, false
		}
	}
	if matches[5] != "" {
		minute, err = strconv.Atoi(matches[5])
		if err != nil {
			return 0, false
		}
	}

	return (((year*100)+month)*100+day)*10000 + hour*100 + minute, true
}

func inferFixtureYear(month int, leagueTitle string) int {
	matches := seasonYearsRe.FindStringSubmatch(leagueTitle)
	if len(matches) != 3 {
		return 0
	}

	startYear, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	endYear, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0
	}
	if endYear < 100 {
		endYear += (startYear / 100) * 100
		if endYear < startYear {
			endYear += 100
		}
	}

	// 90minut often omits fixture years; Jul-Dec belongs to the season start, Jan-Jun to the season end.
	if month >= 7 {
		return startYear
	}
	return endYear
}

func polishMonthIndex(value string) (int, bool) {
	switch strings.ToLower(normalizeWhitespace(value)) {
	case "stycznia":
		return 1, true
	case "lutego":
		return 2, true
	case "marca":
		return 3, true
	case "kwietnia":
		return 4, true
	case "maja":
		return 5, true
	case "czerwca":
		return 6, true
	case "lipca":
		return 7, true
	case "sierpnia":
		return 8, true
	case "wrzesnia", "września":
		return 9, true
	case "pazdziernika", "października":
		return 10, true
	case "listopada":
		return 11, true
	case "grudnia":
		return 12, true
	default:
		return 0, false
	}
}
