package site

import (
	"strings"
	"testing"
)

func TestArchiveFixturesCoverage(t *testing.T) {
	m := loadManifest(t)
	archives := fixturesByKind(m, "archive")
	if len(m.Fixtures) < 20 {
		t.Fatalf("expected at least 20 fixtures, got %d", len(m.Fixtures))
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
	if m.Source != "http://www.90minut.pl/archsezon.php" {
		t.Fatalf("unexpected manifest source: %q", m.Source)
	}
	if want := manifestStamp(m.Fixtures); m.GeneratedAt != want {
		t.Fatalf("manifest generated_at mismatch: got %q want %q", m.GeneratedAt, want)
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

			currentCount := 0
			for _, season := range seasons {
				if season.Current {
					currentCount++
				}
			}
			if currentCount != 1 {
				t.Fatalf("expected exactly one current season, got %d", currentCount)
			}
			if fixture.Season != "" && fixture.Season != "current" {
				if !strings.Contains(seasons[selectedIdx].Label, fixture.Season) {
					t.Fatalf("selected season %q does not match fixture season %q", seasons[selectedIdx].Label, fixture.Season)
				}
			}

			competitions := parseCompetitions(doc, c)
			if len(competitions) == 0 {
				t.Fatalf("expected competitions from %s", fixture.Name)
			}

			if fixture.Name == "archive_2020_21" {
				ekstraklasaURL := c.Resolve("/liga/1/liga11233.html")
				iLigaURL := c.Resolve("/liga/1/liga11234.html")
				iiLigaURL := c.Resolve("/liga/1/liga11235.html")

				ekstraklasaIdx := competitionIndexByURL(competitions, ekstraklasaURL)
				iLigaIdx := competitionIndexByURL(competitions, iLigaURL)
				iiLigaIdx := competitionIndexByURL(competitions, iiLigaURL)

				if ekstraklasaIdx < 0 || iLigaIdx < 0 || iiLigaIdx < 0 {
					t.Fatalf("missing expected league links in 2020/21 archive")
				}
				if !(ekstraklasaIdx < iLigaIdx && iLigaIdx < iiLigaIdx) {
					t.Fatalf("competition order broken: Ekstraklasa=%d I liga=%d II liga=%d", ekstraklasaIdx, iLigaIdx, iiLigaIdx)
				}
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
