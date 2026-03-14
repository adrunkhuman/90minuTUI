package site

import (
	"strings"
	"testing"
)

type leagueRoundExpectation struct {
	firstRoundName     string
	firstRoundFixtures int
}

func TestParseLeagueFixturesFromCorpus(t *testing.T) {
	m := loadManifest(t)
	expected := map[string]leagueRoundExpectation{
		"league_14072": {firstRoundName: "Kolejka 1 - 19-20 lipca", firstRoundFixtures: 9},
		"league_14073": {firstRoundName: "Kolejka 1 - 19-20 lipca", firstRoundFixtures: 9},
	}
	leagues := fixturesByKind(m, "league")
	if len(leagues) < 6 {
		t.Fatalf("expected at least 6 league fixtures, got %d", len(leagues))
	}
	if !containsFixtureName(leagues, "league_14072") {
		t.Fatalf("expected league fixture for liga14072")
	}
	if !containsFixtureName(leagues, "league_14073") {
		t.Fatalf("expected league fixture for liga14073")
	}

	for _, fixture := range leagues {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			doc, _ := fixtureDoc(t, fixture.File)
			page := parseLeaguePage(doc, fixture.URL)
			if page == nil {
				t.Fatalf("expected league page for %s", fixture.Name)
			}
			if len(page.Rounds) == 0 {
				t.Fatalf("expected rounds for %s", fixture.Name)
			}
			if page.LeagueKey == "" {
				t.Fatalf("expected league key for %s", fixture.Name)
			}
			for _, round := range page.Rounds {
				if strings.TrimSpace(round.Name) == "" {
					t.Fatalf("round with empty name in %s", fixture.Name)
				}
				if len(round.Fixtures) == 0 {
					t.Fatalf("round %q has no fixtures in %s", round.Name, fixture.Name)
				}
				assertFixturesSortedByDate(t, fixture.Name, round)
			}

			totalFixtures := 0
			for _, round := range page.Rounds {
				for _, match := range round.Fixtures {
					totalFixtures++
					if match.Home == "" || match.Away == "" || match.Score == "" {
						t.Fatalf("fixture has empty fields in %s", fixture.Name)
					}
					if isScoreLikeText(match.Home) || isScoreLikeText(match.Away) {
						t.Fatalf("fixture side parsed as score token in %s: home=%q away=%q", fixture.Name, match.Home, match.Away)
					}
					if !strings.Contains(match.MatchURL, "mecz.php") {
						t.Fatalf("fixture match url is not a match link in %s: %q", fixture.Name, match.MatchURL)
					}
					if match.MatchID == "" {
						t.Fatalf("fixture missing match id in %s: %q", fixture.Name, match.MatchURL)
					}
				}
			}
			if totalFixtures == 0 {
				t.Fatalf("expected fixtures for %s", fixture.Name)
			}

			firstRound := page.Rounds[0]
			if firstRound.Name == "Wyniki" {
				t.Fatalf("expected first extracted round to be named in %s", fixture.Name)
			}
			for _, round := range page.Rounds {
				lower := strings.ToLower(round.Name)
				if strings.Contains(lower, "strzelcy") || strings.Contains(lower, "statystyki") || strings.Contains(lower, "ostatnia kolejka") {
					t.Fatalf("navigation block parsed as round %q in %s", round.Name, fixture.Name)
				}
			}

			if want, ok := expected[fixture.Name]; ok {
				if firstRound.Name != want.firstRoundName {
					t.Fatalf("unexpected first round name in %s: got %q want %q", fixture.Name, firstRound.Name, want.firstRoundName)
				}
				if len(firstRound.Fixtures) != want.firstRoundFixtures {
					t.Fatalf("unexpected first round fixture count in %s: got %d want %d", fixture.Name, len(firstRound.Fixtures), want.firstRoundFixtures)
				}
			}

			if fixture.Name == "league_14072" || fixture.Name == "league_14073" {
				if totalFixtures < 12 {
					t.Fatalf("expected dense fixture extraction for %s, got %d", fixture.Name, totalFixtures)
				}
			}
		})
	}
}

func assertFixturesSortedByDate(t *testing.T, fixtureName string, round Round) {
	t.Helper()

	lastKey := 0
	lastWhenInfo := ""
	seenDate := false
	seenUndated := false

	for _, fixture := range round.Fixtures {
		dateKey, ok := fixtureDateKey(fixture.WhenInfo)
		if !ok {
			seenUndated = true
			continue
		}
		if seenUndated {
			t.Fatalf("dated fixture %q appears after undated fixture in %s round %q", fixture.MatchID, fixtureName, round.Name)
		}
		if seenDate && dateKey < lastKey {
			t.Fatalf("fixtures not sorted by date in %s round %q: %q before %q", fixtureName, round.Name, fixture.WhenInfo, lastWhenInfo)
		}
		lastKey = dateKey
		lastWhenInfo = fixture.WhenInfo
		seenDate = true
	}
}

func TestParseLeagueStandingsFromCorpus(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/league_liga11233.html")
	page := parseLeaguePage(doc, "http://www.90minut.pl/liga/1/liga11233.html")
	if page == nil {
		t.Fatalf("expected league page")
	}
	if len(page.Standings) != 16 {
		t.Fatalf("unexpected standings row count: got %d want 16", len(page.Standings))
	}

	first := page.Standings[0]
	if first.Position != 1 || first.Team != "Legia Warszawa" {
		t.Fatalf("unexpected first standings row: %+v", first)
	}
	if first.Played != 30 || first.Won != 19 || first.Drawn != 7 || first.Lost != 4 || first.Points != 64 {
		t.Fatalf("unexpected first standings stats: %+v", first)
	}
}
