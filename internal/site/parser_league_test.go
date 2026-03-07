package site

import (
	"strings"
	"testing"
)

func TestParseLeagueFixturesFromCorpus(t *testing.T) {
	m := loadManifest(t)
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

			if fixture.Name == "league_14072" || fixture.Name == "league_14073" {
				if totalFixtures < 12 {
					t.Fatalf("expected dense fixture extraction for %s, got %d", fixture.Name, totalFixtures)
				}
			}
		})
	}
}
