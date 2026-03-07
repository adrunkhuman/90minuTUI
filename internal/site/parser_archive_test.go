package site

import "testing"

func TestArchiveFixturesCoverage(t *testing.T) {
	m := loadManifest(t)
	archives := fixturesByKind(m, "archive")
	if len(m.Fixtures) != 20 {
		t.Fatalf("expected 20 fixtures, got %d", len(m.Fixtures))
	}
	if len(archives) < 4 {
		t.Fatalf("expected at least 4 archive fixtures, got %d", len(archives))
	}

	if !containsFixtureName(archives, "archive_2019_20") {
		t.Fatalf("expected archive fixture for 2019/20 season")
	}
	if !containsFixtureName(archives, "archive_2020_21") {
		t.Fatalf("expected archive fixture for 2020/21 season")
	}
}

func TestParseSeasonsAndCompetitionsFromArchiveFixtures(t *testing.T) {
	m := loadManifest(t)
	archives := fixturesByKind(m, "archive")
	c := NewClient()
	foundDiacritics := false

	for _, fixture := range archives {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			doc, _ := fixtureDoc(t, fixture.File)

			seasons, selectedIdx := parseSeasons(doc, c)
			if len(seasons) == 0 {
				t.Fatalf("expected seasons from %s", fixture.Name)
			}
			if selectedIdx < 0 || selectedIdx >= len(seasons) {
				t.Fatalf("invalid selected season index: %d", selectedIdx)
			}
			if !seasons[selectedIdx].Current {
				t.Fatalf("selected season is not marked current")
			}

			competitions := parseCompetitions(doc, c)
			if len(competitions) == 0 {
				t.Fatalf("expected competitions from %s", fixture.Name)
			}

			for _, season := range seasons {
				if hasPolishDiacritic(season.Label) {
					foundDiacritics = true
					break
				}
			}
			if !foundDiacritics {
				for _, competition := range competitions {
					if hasPolishDiacritic(competition.Name) {
						foundDiacritics = true
						break
					}
				}
			}
		})
	}

	if !foundDiacritics {
		t.Fatalf("expected at least one decoded Polish diacritic in archive fixtures")
	}
}
