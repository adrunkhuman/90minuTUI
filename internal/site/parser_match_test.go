package site

import "testing"

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

			homePlayers := map[string]struct{}{}
			for _, player := range page.HomeLineup {
				homePlayers[player.Name] = struct{}{}
			}
			awayPlayers := map[string]struct{}{}
			for _, player := range page.AwayLineup {
				awayPlayers[player.Name] = struct{}{}
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

				_, inHome := homePlayers[event.Text]
				_, inAway := awayPlayers[event.Text]
				if !inHome && !inAway {
					continue
				}

				foundPlayerSideEvidence = true
				if event.TeamSide == "home" && !inHome {
					t.Fatalf("event side mismatch for player %q in %s: got home", event.Text, fixture.Name)
				}
				if event.TeamSide == "away" && !inAway {
					t.Fatalf("event side mismatch for player %q in %s: got away", event.Text, fixture.Name)
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
