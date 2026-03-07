package site

import (
	"testing"
)

var expectedCardSidesByFixture = map[string]map[string]string{
	"match_2022810": {
		"(2) Paskal Meyer":    "away",
		"(39) Filip Kocaba":   "home",
		"(68) Bartosz Wolski": "away",
	},
}

func TestParseMatchFixturesFromCorpus(t *testing.T) {
	m := loadManifest(t)
	matches := fixturesByKind(m, "match")
	if len(matches) < 6 {
		t.Fatalf("expected at least 6 match fixtures, got %d", len(matches))
	}

	foundEventTimeline := false
	foundDiacritics := false
	foundPlayerSideEvidence := false

	for _, fixture := range matches {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			doc, _ := fixtureDoc(t, fixture.File)
			page := parseMatchPage(doc, fixture.URL)
			if page == nil {
				t.Fatalf("expected match page for %s", fixture.Name)
			}
			if page.Title == "" {
				t.Fatalf("expected title for %s", fixture.Name)
			}

			if hasPolishDiacritic(page.HomeTeam) || hasPolishDiacritic(page.AwayTeam) {
				foundDiacritics = true
			}

			if len(page.Events) > 0 {
				foundEventTimeline = true
			}

			expectedFixtureCards := expectedCardSidesByFixture[fixture.Name]
			seenExpectedCards := map[string]bool{}

			for _, event := range page.Events {
				if event.TeamSide != "home" && event.TeamSide != "away" {
					t.Fatalf("invalid event side %q in %s", event.TeamSide, fixture.Name)
				}

				if event.Kind != "YC" && event.Kind != "RC" {
					continue
				}
				if event.Text == "" {
					continue
				}

				expectedSide, ok := expectedFixtureCards[event.Text]
				if !ok || expectedSide == "" {
					continue
				}

				seenExpectedCards[event.Text] = true
				foundPlayerSideEvidence = true
				if event.TeamSide != expectedSide {
					t.Fatalf("event side mismatch for player %q in %s: got %q want %q", event.Text, fixture.Name, event.TeamSide, expectedSide)
				}
			}

			for player := range expectedFixtureCards {
				if seenExpectedCards[player] {
					continue
				}
				t.Fatalf("expected YC/RC event not found for %q in %s", player, fixture.Name)
			}
		})
	}

	if !foundEventTimeline {
		t.Fatalf("expected at least one match fixture with timeline events")
	}
	if !foundDiacritics {
		t.Fatalf("expected at least one match fixture with Polish diacritics")
	}
	if !foundPlayerSideEvidence {
		t.Fatalf("expected at least one YC/RC event tied to lineup players")
	}
}
