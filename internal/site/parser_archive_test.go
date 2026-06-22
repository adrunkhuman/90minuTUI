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
				if season.SeasonID == "" {
					t.Fatalf("expected season id for %q", season.URL)
				}
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

			competitions := parseCompetitions(doc, c, c.Resolve("/archiwum.php"))
			if len(competitions) == 0 {
				t.Fatalf("expected competitions from %s", fixture.Name)
			}
			for _, competition := range competitions {
				if competition.LeagueKey == "" {
					t.Fatalf("expected league key for %q", competition.URL)
				}
			}

			if fixture.Name == "archive_2020_21" {
				ekstraklasaURL := c.Resolve("/liga/1/liga11233.html")
				iLigaURL := c.Resolve("/liga/1/liga11234.html")
				iiLigaURL := c.Resolve("/liga/1/liga11235.html")
				iiiLigaSelectorURL := c.Resolve("/ligireg.php?poziom=4&id_sezon=97")
				regionalneURL := c.Resolve("/ligireg.php?id_sezon=97")
				regionalCupsURL := c.Resolve("/polcups.php?id_sezon=97")

				ekstraklasaIdx := competitionIndexByURL(competitions, ekstraklasaURL)
				iLigaIdx := competitionIndexByURL(competitions, iLigaURL)
				iiLigaIdx := competitionIndexByURL(competitions, iiLigaURL)
				iiiLigaSelectorIdx := competitionIndexByURL(competitions, iiiLigaSelectorURL)
				regionalneIdx := competitionIndexByURL(competitions, regionalneURL)
				regionalCupsIdx := competitionIndexByURL(competitions, regionalCupsURL)

				if ekstraklasaIdx < 0 || iLigaIdx < 0 || iiLigaIdx < 0 {
					t.Fatalf("missing expected league links in 2020/21 archive")
				}
				if !(ekstraklasaIdx < iLigaIdx && iLigaIdx < iiLigaIdx) {
					t.Fatalf("competition order broken: Ekstraklasa=%d I liga=%d II liga=%d", ekstraklasaIdx, iLigaIdx, iiLigaIdx)
				}
				if iiiLigaSelectorIdx < 0 || regionalneIdx < 0 {
					t.Fatalf("missing III liga or ligi regionalne links in 2020/21 archive")
				}
				if regionalCupsIdx < 0 {
					t.Fatalf("missing regional cups link in 2020/21 archive")
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

func TestParseCompetitionsGroupsWomenLeagueTiers(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/archive_current.html")

	c := NewClient()
	archiveURL := c.Resolve("/archiwum.php")
	competitions := parseCompetitions(doc, c, archiveURL)

	thirdTierIdx := competitionIndexByURL(competitions, womenTierMenuURL(archiveURL, "iii-liga-kobiet"))
	if thirdTierIdx < 0 {
		t.Fatalf("expected grouped III liga kobiet submenu in competitions: %+v", competitions)
	}
	if competitions[thirdTierIdx].Name != "III liga kobiet 2025/2026" {
		t.Fatalf("unexpected women submenu title: %q", competitions[thirdTierIdx].Name)
	}

	for _, competition := range competitions {
		if strings.Contains(competition.Name, "grupa: I") && strings.Contains(competition.Name, "III liga kobiet") {
			t.Fatalf("expected grouped III liga kobiet entries to stay out of top-level list: %+v", competitions)
		}
	}

	if competitionIndexByURL(competitions, c.Resolve("/liga/1/liga14141.html")) < 0 {
		t.Fatalf("expected Ekstraliga kobiet to remain directly selectable")
	}
}

func TestParseCompetitionsGroupsFutsalLeagueTiers(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/archive_current.html")

	c := NewClient()
	archiveURL := c.Resolve("/archiwum.php")
	competitions := parseCompetitions(doc, c, archiveURL)

	futsalTierIdx := competitionIndexByURL(competitions, futsalTierMenuURL(archiveURL, "i-liga-futsalu"))
	if futsalTierIdx < 0 {
		t.Fatalf("expected grouped I liga futsalu submenu in competitions: %+v", competitions)
	}
	if competitions[futsalTierIdx].Name != "I liga futsalu 2025/2026" {
		t.Fatalf("unexpected futsal submenu title: %q", competitions[futsalTierIdx].Name)
	}

	for _, competition := range competitions {
		if strings.Contains(competition.Name, "I liga futsalu") && strings.Contains(competition.Name, "grupa:") {
			t.Fatalf("expected grouped I liga futsalu entries to stay out of top-level list: %+v", competitions)
		}
	}

	if competitionIndexByURL(competitions, c.Resolve("/liga/1/liga14148.html")) < 0 {
		t.Fatalf("expected Futsal Ekstraklasa to remain directly selectable")
	}
}

func TestParseCompetitionMenuForIIILigaSelector(t *testing.T) {
	html := `<html><body><table class="main"><tr><td valign="top"><p align="center"><b>III liga 2025/26</b></p><table class="main"><tr><td width="100"></td><td><a href="/liga/1/liga14154.html" class="main">I</a></td></tr><tr><td width="100"></td><td><a href="/liga/1/liga14155.html" class="main">II</a></td></tr></table></td></tr></table></body></html>`
	doc, err := decodeAndParse([]byte(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	menu := parseCompetitionMenu(doc, "http://www.90minut.pl/ligireg.php?poziom=4&id_sezon=107", NewClient())
	if menu == nil {
		t.Fatalf("expected III liga submenu")
	}
	if menu.Title != "III liga 2025/26" {
		t.Fatalf("unexpected menu title: %q", menu.Title)
	}
	if len(menu.Items) != 2 || menu.Items[0].Name != "III liga 2025/26, gr. I" || menu.Items[1].Name != "III liga 2025/26, gr. II" {
		t.Fatalf("unexpected III liga submenu items: %+v", menu.Items)
	}
}

func TestParseCompetitionMenuForRegionalRoot(t *testing.T) {
	html := `<html><body><table class="main"><tr><td valign="top"><p align="center"><b>Ligi regionalne 2025/26</b></p><a href="/ligireg-16.html" class="main">Dolnośląski ZPN</a><a href="/mecze_okreg.php?id_okreg=16" class="main">Dziś grają</a><a href="/ligireg-1.html" class="main">Kujawsko-Pomorski ZPN</a></td></tr></table></body></html>`
	doc, err := decodeAndParse([]byte(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	menu := parseCompetitionMenu(doc, "http://www.90minut.pl/ligireg.php?id_sezon=107", NewClient())
	if menu == nil {
		t.Fatalf("expected regional submenu")
	}
	if len(menu.Items) != 2 || menu.Items[0].Name != "Dolnośląski ZPN" || menu.Items[1].Name != "Kujawsko-Pomorski ZPN" {
		t.Fatalf("unexpected regional submenu items: %+v", menu.Items)
	}
}

func TestParseCompetitionMenuForRegionalRootWithAssociationQueryLinks(t *testing.T) {
	html := `<html><body><table class="main"><tr><td valign="top"><p align="center"><b>Ligi regionalne 2025/26</b></p><a href="/ligireg.php?id_okreg=16&id_sezon=107" class="main">Dolnośląski ZPN</a><a href="/ligireg.php?id_okreg=8&id_sezon=107" class="main">Lubuski ZPN</a></td></tr></table></body></html>`
	doc, err := decodeAndParse([]byte(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	menu := parseCompetitionMenu(doc, "http://www.90minut.pl/ligireg.php?id_sezon=107", NewClient())
	if menu == nil {
		t.Fatalf("expected regional submenu")
	}
	if len(menu.Items) != 2 {
		t.Fatalf("unexpected regional submenu items: %+v", menu.Items)
	}
}

func TestParseCompetitionMenuForRegionalAssociationPage(t *testing.T) {
	html := `<html><body><table class="main"><tr><td valign="top"><p align="center"><b>Ligi regionalne 2025/26 - Dolnośląski ZPN</b></p><a href="/liga/1/liga14169.html" class="main">IV liga 2025/2026, grupa: dolnośląska</a><a href="/liga/1/liga14204.html" class="main">Klasa okręgowa 2025/2026, grupa: Jelenia Góra</a></td></tr></table></body></html>`
	doc, err := decodeAndParse([]byte(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	menu := parseCompetitionMenu(doc, "http://www.90minut.pl/ligireg-16.html", NewClient())
	if menu == nil {
		t.Fatalf("expected regional association submenu")
	}
	if len(menu.Items) != 2 {
		t.Fatalf("unexpected regional association item count: %d", len(menu.Items))
	}
}

func TestParseCompetitionMenuForRegionalCupsPage(t *testing.T) {
	html := `<html><body><table class="main"><tr><td valign="top"><p align="center"><b>Puchary krajowe 2025/26</b></p><a href="/liga/1/liga14076.html" class="main">Puchar Polski</a><a href="/liga/1/liga14075.html" class="main">Superpuchar Polski</a><a href="/liga/1/liga14636.html" class="main">Puchar Polski 2025/2026, grupa: Lubuski ZPN</a><a href="/liga/1/liga14069.html" class="main">Puchar Polski 2025/2026, grupa: Lubuski ZPN - Gorzów Wielkopolski</a></td></tr></table></body></html>`
	doc, err := decodeAndParse([]byte(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	menu := parseCompetitionMenu(doc, "http://www.90minut.pl/polcups.php?id_sezon=107", NewClient())
	if menu == nil {
		t.Fatalf("expected regional cups submenu")
	}
	if len(menu.Items) != 2 {
		t.Fatalf("expected regional cups only, got %+v", menu.Items)
	}
}

func TestParseCompetitionMenuForWomenTierPage(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/archive_current.html")

	c := NewClient()
	menu := parseCompetitionMenu(doc, womenTierMenuURL(c.Resolve("/archiwum.php"), "iii-liga-kobiet"), c)
	if menu == nil {
		t.Fatalf("expected women tier submenu")
	}
	if menu.Title != "III liga kobiet 2025/2026" {
		t.Fatalf("unexpected women tier title: %q", menu.Title)
	}
	if len(menu.Items) != 4 {
		t.Fatalf("expected grouped III liga kobiet items, got %+v", menu.Items)
	}
	if menu.Items[0].Name != "III liga kobiet 2025/2026, grupa: I" {
		t.Fatalf("unexpected first women tier item: %+v", menu.Items[0])
	}
	if menu.Items[3].Name != "III liga kobiet 2025/2026, grupa: IV" {
		t.Fatalf("unexpected last women tier item: %+v", menu.Items[3])
	}
}

func TestParseCompetitionMenuForSponsoredWomenTierPage(t *testing.T) {
	html := `<html><body><table class="main"><tr><td colspan="2"><p align="center"><img src="http://img.90minut.pl/img/line.gif"></p></td></tr><tr><td></td><td rowspan="10" align="right"><img src="http://img.90minut.pl/logo/kobiety.gif"></td></tr><tr><td><a href="/liga/1/liga11667.html" class="main">III liga kobiet 2020/2021, grupa: I</a></td></tr><tr><td><a href="/liga/1/liga11661.html" class="main">III liga kobiet 2020/2021, grupa: II</a></td></tr><tr><td><a href="/liga/1/liga11674.html" class="main">III liga kobiet 2020/2021, grupa: III</a></td></tr><tr><td><a href="/liga/1/liga11675.html" class="main">Keeza III liga kobiet 2020/2021, grupa: IV</a></td></tr></table></body></html>`
	doc, err := decodeAndParse([]byte(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("parse synthetic HTML: %v", err)
	}

	c := NewClient()
	menu := parseCompetitionMenu(doc, womenTierMenuURL(c.Resolve("/archiwum.php?id_sezon=97"), "iii-liga-kobiet"), c)
	if menu == nil {
		t.Fatalf("expected sponsored women tier submenu")
	}
	if menu.Title != "III liga kobiet 2020/2021" {
		t.Fatalf("unexpected sponsored women tier title: %q", menu.Title)
	}
	if len(menu.Items) != 4 {
		t.Fatalf("expected 4 sponsored women tier items, got %+v", menu.Items)
	}
	if menu.Items[3].Name != "Keeza III liga kobiet 2020/2021, grupa: IV" {
		t.Fatalf("expected sponsored child to keep original name, got %+v", menu.Items[3])
	}
}

func TestParseCompetitionMenuForFutsalTierPage(t *testing.T) {
	doc, _ := fixtureDoc(t, "fixtures/archive_current.html")

	c := NewClient()
	menu := parseCompetitionMenu(doc, futsalTierMenuURL(c.Resolve("/archiwum.php"), "i-liga-futsalu"), c)
	if menu == nil {
		t.Fatalf("expected futsal tier submenu")
	}
	if menu.Title != "I liga futsalu 2025/2026" {
		t.Fatalf("unexpected futsal tier title: %q", menu.Title)
	}
	if len(menu.Items) != 2 {
		t.Fatalf("expected grouped futsal items, got %+v", menu.Items)
	}
	if menu.Items[0].Name != "I liga futsalu 2025/2026, grupa: południowa" {
		t.Fatalf("unexpected first futsal tier item: %+v", menu.Items[0])
	}
	if menu.Items[1].Name != "I liga futsalu 2025/2026, grupa: północna" {
		t.Fatalf("unexpected second futsal tier item: %+v", menu.Items[1])
	}
}

func womenTierMenuURL(archiveURL, key string) string {
	return archiveCompetitionMenuURL(archiveURL, key, archiveGroupSpecs[0])
}

func futsalTierMenuURL(archiveURL, key string) string {
	return archiveCompetitionMenuURL(archiveURL, key, archiveGroupSpecs[1])
}
