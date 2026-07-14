package site

import (
	"testing"
)

var expectedCardSidesByFixture = map[string]map[string]TeamSide{
	"match_2022810": {
		"(2) Paskal Meyer":    TeamSideAway,
		"(39) Filip Kocaba":   TeamSideHome,
		"(68) Bartosz Wolski": TeamSideAway,
	},
}

var expectedMissedPenaltiesByFixture = map[string][]MatchEvent{
	"match_2022961": {
		{MinuteText: "52", Kind: EventKindMiss, TeamSide: TeamSideHome, Text: "Gierman Barkowskij 52 (nk)"},
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
		t.Run(fixture.Name, func(t *testing.T) {
			doc, _ := fixtureDoc(t, fixture.File)
			page := parseMatchPage(doc, fixture.URL)
			if page == nil {
				t.Fatalf("expected match page for %s", fixture.Name)
			}
			if page.Title == "" {
				t.Fatalf("expected title for %s", fixture.Name)
			}
			if page.MatchID == "" {
				t.Fatalf("expected match id for %s", fixture.Name)
			}
			if fixtureID := extractMatchID(fixture.URL); fixtureID != "" && page.MatchID != fixtureID {
				t.Fatalf("match id mismatch for %s: got %q want %q", fixture.Name, page.MatchID, fixtureID)
			}

			if hasPolishDiacritic(page.HomeTeam) || hasPolishDiacritic(page.AwayTeam) {
				foundDiacritics = true
			}

			if len(page.Events) > 0 {
				foundEventTimeline = true
			}

			expectedFixtureCards := expectedCardSidesByFixture[fixture.Name]
			expectedMissedPenalties := expectedMissedPenaltiesByFixture[fixture.Name]
			seenExpectedCards := map[string]bool{}
			seenExpectedMissed := make([]bool, len(expectedMissedPenalties))

			for _, event := range page.Events {
				if event.TeamSide != TeamSideHome && event.TeamSide != TeamSideAway {
					t.Fatalf("invalid event side %q in %s", event.TeamSide, fixture.Name)
				}
				if !validMatchEventKind(event.Kind) {
					t.Fatalf("invalid event kind %q in %s", event.Kind, fixture.Name)
				}

				for i, want := range expectedMissedPenalties {
					if event.Kind != want.Kind || event.TeamSide != want.TeamSide || event.MinuteText != want.MinuteText || event.Text != want.Text {
						continue
					}
					seenExpectedMissed[i] = true
				}

				if event.Kind != EventKindYellowCard && event.Kind != EventKindRedCard {
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

			for i, seen := range seenExpectedMissed {
				if seen {
					continue
				}
				t.Fatalf("expected missed-penalty event not found in %s: %#v", fixture.Name, expectedMissedPenalties[i])
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

func validMatchEventKind(kind MatchEventKind) bool {
	switch kind {
	case EventKindGoal, EventKindMiss, EventKindYellowCard, EventKindRedCard, EventKindSubstitution, EventKindGeneric:
		return true
	default:
		return false
	}
}

func TestParseMatchFixtureKeepsChainedSubstitutions(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/match_2023004.html")
	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=2023004")
	if page == nil {
		t.Fatalf("expected match page")
	}

	firstCount := 0
	secondCount := 0
	for _, event := range page.Events {
		if event.Kind != EventKindSubstitution || event.TeamSide != TeamSideAway {
			continue
		}

		switch {
		case event.SubstitutionOut == "(8) Ali Gholizadeh" && event.SubstitutionIn == "(11) Daniel Håkans":
			if event.MinuteText != "19" || event.Minute != 19 || event.Stoppage != 0 || !event.HasMinute {
				t.Fatalf("unexpected first chained substitution minute: %#v", event)
			}
			firstCount++
		case event.SubstitutionOut == "(11) Daniel Håkans" && event.SubstitutionIn == "(77) Luis Palma":
			if event.MinuteText != "65" || event.Minute != 65 || event.Stoppage != 0 || !event.HasMinute {
				t.Fatalf("unexpected second chained substitution minute: %#v", event)
			}
			secondCount++
		case event.SubstitutionOut == "(8) Ali Gholizadeh" && event.SubstitutionIn == "(77) Luis Palma":
			t.Fatalf("parser collapsed chained substitutions into final entrant: %#v", event)
		}
	}

	if firstCount != 1 || secondCount != 1 {
		t.Fatalf("expected both chained substitution events, got %#v", page.Events)
	}
}

func TestParseMatchFixtureExtractsPlayerAndRefereeIDs(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/match_2022810.html")
	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=2022810")
	if page == nil {
		t.Fatalf("expected match page")
	}
	if page.Referee != "Łukasz Karski (Słupsk)" || page.RefereeID != "1125" {
		t.Fatalf("unexpected referee identity: name=%q id=%q", page.Referee, page.RefereeID)
	}
	if len(page.HomeLineup) == 0 || page.HomeLineup[0].Name != "(30) Dominik Hładun" || page.HomeLineup[0].PlayerID != "22468" {
		t.Fatalf("unexpected home lineup identity: %+v", page.HomeLineup)
	}
	if len(page.AwayLineup) == 0 || page.AwayLineup[0].Name != "(1) Ivan Brkić" || page.AwayLineup[0].PlayerID != "49041" {
		t.Fatalf("unexpected away lineup identity: %+v", page.AwayLineup)
	}
}
