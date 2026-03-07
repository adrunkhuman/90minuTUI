package site

import (
	"testing"

	"github.com/PuerkitoBio/goquery"
)

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
			expectedCardSides := expectedCardPlayerSides(doc)
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

				expectedSide, ok := expectedCardSides[event.Text]
				if !ok || expectedSide == "" {
					continue
				}

				foundPlayerSideEvidence = true
				if event.TeamSide != expectedSide {
					t.Fatalf("event side mismatch for player %q in %s: got %q want %q", event.Text, fixture.Name, event.TeamSide, expectedSide)
				}
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

func expectedCardPlayerSides(doc *goquery.Document) map[string]string {
	players := map[string]string{}
	table := doc.Find("table.main[width='480']").First()
	if table.Length() == 0 {
		return players
	}

	table.Find("tr[bgcolor]").Each(func(_ int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() != 3 {
			return
		}

		collectCardPlayer(players, tds.Eq(0), "home")
		collectCardPlayer(players, tds.Eq(2), "away")
	})

	return players
}

func collectCardPlayer(players map[string]string, cell *goquery.Selection, side string) {
	if cell.Find("img[src*='yel.gif'], img[src*='red.gif'], img[src*='red2.gif']").Length() == 0 {
		return
	}

	name := normalizeWhitespace(cell.Find("a").First().Text())
	if name == "" {
		return
	}

	if current, exists := players[name]; exists && current != side {
		players[name] = ""
		return
	}
	players[name] = side
}
