package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

const (
	baseURL        = "http://www.90minut.pl"
	archiveURL     = "http://www.90minut.pl/archsezon.php"
	targetFixtures = 20
)

var ligaIDRe = regexp.MustCompile(`liga/(\d+)/(liga\d+)\.html`)
var meczIDRe = regexp.MustCompile(`id_mecz=(\d+)`)

type seasonLink struct {
	Label string
	URL   string
}

type fixtureMeta struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Season string `json:"season"`
	File   string `json:"file"`
	Note   string `json:"note,omitempty"`
}

type manifest struct {
	GeneratedAt string        `json:"generated_at"`
	Source      string        `json:"source"`
	Fixtures    []fixtureMeta `json:"fixtures"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rng := rand.New(rand.NewSource(90))
	client := &http.Client{Timeout: 30 * time.Second}

	root := filepath.Join("internal", "site", "testdata")
	fixtureDir := filepath.Join(root, "fixtures")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		return fmt.Errorf("mkdir fixtures dir: %w", err)
	}

	archiveBody, archiveDoc, err := fetchDoc(client, archiveURL)
	if err != nil {
		return err
	}

	seasons := parseSeasonLinks(archiveDoc)
	if len(seasons) == 0 {
		return fmt.Errorf("no seasons discovered from archive page")
	}

	metas := make([]fixtureMeta, 0, targetFixtures)
	saved := map[string]struct{}{}
	requiredNames := map[string]struct{}{
		"archive_2019_20": {},
		"archive_2020_21": {},
		"league_14072":    {},
		"league_14073":    {},
	}

	if err := saveFixture(fixtureDir, "archive_current", archiveBody); err != nil {
		return err
	}
	metas = append(metas, fixtureMeta{
		Name: "archive_current", Kind: "archive", URL: archiveURL,
		Season: "current", File: filepath.ToSlash(filepath.Join("fixtures", "archive_current.html")),
	})
	saved[archiveURL] = struct{}{}

	covidSeasons := pickSeasons(seasons, []string{"2019/20", "2019/2020", "2020/21", "2020/2021"})
	randomSeasonPool := make([]seasonLink, 0, len(seasons))
	for _, s := range seasons {
		if _, ok := saved[s.URL]; ok {
			continue
		}
		randomSeasonPool = append(randomSeasonPool, s)
	}

	rng.Shuffle(len(randomSeasonPool), func(i, j int) {
		randomSeasonPool[i], randomSeasonPool[j] = randomSeasonPool[j], randomSeasonPool[i]
	})

	seasonFixtures := append([]seasonLink{}, covidSeasons...)
	for _, s := range randomSeasonPool {
		if len(seasonFixtures) >= 4 {
			break
		}
		if containsSeason(seasonFixtures, s.URL) {
			continue
		}
		seasonFixtures = append(seasonFixtures, s)
	}

	leagueCandidates := []fixtureMeta{
		{
			Name:   "league_14072",
			Kind:   "league",
			URL:    "http://www.90minut.pl/liga/1/liga14072.html",
			Season: "2025/26",
			Note:   "addenda-heavy fixture rows",
		},
		{
			Name:   "league_14073",
			Kind:   "league",
			URL:    "http://www.90minut.pl/liga/1/liga14073.html",
			Season: "2025/26",
			Note:   "round text annotations",
		},
		{
			Name:   "league_liga10550",
			Kind:   "league",
			URL:    "http://www.90minut.pl/liga/1/liga10550.html",
			Season: "2019/20",
		},
		{
			Name:   "league_liga10551",
			Kind:   "league",
			URL:    "http://www.90minut.pl/liga/1/liga10551.html",
			Season: "2019/20",
		},
		{
			Name:   "league_liga11233",
			Kind:   "league",
			URL:    "http://www.90minut.pl/liga/1/liga11233.html",
			Season: "2020/21",
		},
		{
			Name:   "league_liga11234",
			Kind:   "league",
			URL:    "http://www.90minut.pl/liga/1/liga11234.html",
			Season: "2020/21",
		},
		{
			Name:   "league_liga11235",
			Kind:   "league",
			URL:    "http://www.90minut.pl/liga/1/liga11235.html",
			Season: "2020/21",
		},
	}

	for _, s := range seasonFixtures {
		if len(metas) >= targetFixtures {
			break
		}

		if _, ok := saved[s.URL]; ok {
			continue
		}

		seasonBody, seasonDoc, fetchErr := fetchDoc(client, s.URL)
		if fetchErr != nil {
			fmt.Fprintf(os.Stderr, "warn: skip season %q (%s): %v\n", s.Label, s.URL, fetchErr)
			continue
		}

		archiveName := "archive_" + slugSeason(s.Label)
		if err := saveFixture(fixtureDir, archiveName, seasonBody); err != nil {
			return err
		}
		metas = append(metas, fixtureMeta{
			Name:   archiveName,
			Kind:   "archive",
			URL:    s.URL,
			Season: s.Label,
			File:   filepath.ToSlash(filepath.Join("fixtures", archiveName+".html")),
		})
		saved[s.URL] = struct{}{}

		leagues := parseLeagueLinks(seasonDoc)
		rng.Shuffle(len(leagues), func(i, j int) {
			leagues[i], leagues[j] = leagues[j], leagues[i]
		})

		addedForSeason := 0
		for _, league := range leagues {
			if len(leagueCandidates) >= 12 {
				break
			}
			if addedForSeason >= 4 {
				break
			}
			if league == "" || containsURL(leagueCandidates, league) {
				continue
			}
			name := filenameFromLeagueURL(league)
			if name == "" {
				continue
			}
			leagueCandidates = append(leagueCandidates, fixtureMeta{Name: name, Kind: "league", URL: league, Season: s.Label})
			addedForSeason++
		}
	}

	rng.Shuffle(len(leagueCandidates), func(i, j int) {
		leagueCandidates[i], leagueCandidates[j] = leagueCandidates[j], leagueCandidates[i]
	})

	leagueCandidates = prioritizeLeagues(leagueCandidates)

	matchCandidates := make([]fixtureMeta, 0, 20)

	for _, league := range leagueCandidates {
		if len(metas) >= 12 {
			break
		}
		if _, ok := saved[league.URL]; ok {
			continue
		}

		leagueBody, leagueDoc, fetchErr := fetchDoc(client, league.URL)
		if fetchErr != nil {
			fmt.Fprintf(os.Stderr, "warn: skip league %q (%s): %v\n", league.Name, league.URL, fetchErr)
			continue
		}

		if err := saveFixture(fixtureDir, league.Name, leagueBody); err != nil {
			return err
		}
		league.File = filepath.ToSlash(filepath.Join("fixtures", league.Name+".html"))
		metas = append(metas, league)
		saved[league.URL] = struct{}{}

		matches := parseMatchLinks(leagueDoc)
		rng.Shuffle(len(matches), func(i, j int) {
			matches[i], matches[j] = matches[j], matches[i]
		})

		addedMatches := 0
		for _, match := range matches {
			if len(matchCandidates) >= 16 {
				break
			}
			if addedMatches >= 2 {
				break
			}
			if containsURL(matchCandidates, match) {
				continue
			}
			name := filenameFromMatchURL(match)
			if name == "" {
				continue
			}
			matchCandidates = append(matchCandidates, fixtureMeta{Name: name, Kind: "match", URL: match, Season: league.Season})
			addedMatches++
		}
	}

	for _, match := range matchCandidates {
		if len(metas) >= targetFixtures {
			break
		}
		if _, ok := saved[match.URL]; ok {
			continue
		}

		matchBody, _, fetchErr := fetchDoc(client, match.URL)
		if fetchErr != nil {
			fmt.Fprintf(os.Stderr, "warn: skip match %q (%s): %v\n", match.Name, match.URL, fetchErr)
			continue
		}

		if err := saveFixture(fixtureDir, match.Name, matchBody); err != nil {
			return err
		}
		match.File = filepath.ToSlash(filepath.Join("fixtures", match.Name+".html"))
		metas = append(metas, match)
		saved[match.URL] = struct{}{}
	}

	if len(metas) < targetFixtures {
		return fmt.Errorf("collected %d fixtures, expected at least %d", len(metas), targetFixtures)
	}

	if len(metas) > targetFixtures {
		metas = metas[:targetFixtures]
	}

	if err := validateRequiredFixtures(metas, requiredNames); err != nil {
		return err
	}

	manifestData, err := json.MarshalIndent(manifest{
		GeneratedAt: deterministicManifestStamp(metas),
		Source:      archiveURL,
		Fixtures:    metas,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifestPath, append(manifestData, '\n'), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	fmt.Printf("saved %d fixtures under %s\n", len(metas), fixtureDir)
	return nil
}

func fetchDoc(client *http.Client, rawURL string) ([]byte, *goquery.Document, error) {
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch %q: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fetch %q: unexpected status %s", rawURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read %q body: %w", rawURL, err)
	}

	enc, _, _ := charset.DetermineEncoding(body, resp.Header.Get("Content-Type"))
	decoded := enc.NewDecoder().Reader(bytes.NewReader(body))
	doc, err := goquery.NewDocumentFromReader(decoded)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %q: %w", rawURL, err)
	}

	return body, doc, nil
}

func parseSeasonLinks(doc *goquery.Document) []seasonLink {
	out := make([]seasonLink, 0, 100)
	seen := map[string]struct{}{}

	doc.Find("select[name='urljump'] option").Each(func(_ int, sel *goquery.Selection) {
		raw, ok := sel.Attr("value")
		if !ok {
			return
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}

		absolute := resolveURL(raw)
		if _, exists := seen[absolute]; exists {
			return
		}

		seen[absolute] = struct{}{}
		out = append(out, seasonLink{Label: strings.TrimSpace(sel.Text()), URL: absolute})
	})

	return out
}

func parseLeagueLinks(doc *goquery.Document) []string {
	out := make([]string, 0, 64)
	seen := map[string]struct{}{}

	doc.Find("a.main[href*='/liga/']").Each(func(_ int, sel *goquery.Selection) {
		href := strings.TrimSpace(sel.AttrOr("href", ""))
		if href == "" {
			return
		}
		absolute := resolveURL(href)
		if !strings.Contains(strings.ToLower(absolute), "/liga/") {
			return
		}
		if _, exists := seen[absolute]; exists {
			return
		}
		seen[absolute] = struct{}{}
		out = append(out, absolute)
	})

	return out
}

func parseMatchLinks(doc *goquery.Document) []string {
	out := make([]string, 0, 128)
	seen := map[string]struct{}{}

	doc.Find("a[href*='mecz.php?id_mecz=']").Each(func(_ int, sel *goquery.Selection) {
		href := strings.TrimSpace(sel.AttrOr("href", ""))
		if href == "" {
			return
		}
		absolute := resolveURL(href)
		if _, exists := seen[absolute]; exists {
			return
		}
		seen[absolute] = struct{}{}
		out = append(out, absolute)
	})

	return out
}

func resolveURL(raw string) string {
	base, _ := url.Parse(baseURL)
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.IsAbs() {
		return u.String()
	}
	return base.ResolveReference(u).String()
}

func saveFixture(dir, name string, body []byte) error {
	path := filepath.Join(dir, name+".html")
	return os.WriteFile(path, body, 0o644)
}

func slugSeason(label string) string {
	v := strings.ToLower(strings.TrimSpace(label))
	v = strings.ReplaceAll(v, " ", "_")
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "-", "_")
	v = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(v, "")
	v = strings.Trim(v, "_")
	if v == "" {
		return "unknown"
	}
	return v
}

func pickSeasons(seasons []seasonLink, needles []string) []seasonLink {
	out := make([]seasonLink, 0, 4)
	for _, needle := range needles {
		needle = strings.ToLower(needle)
		for _, season := range seasons {
			if strings.Contains(strings.ToLower(season.Label), needle) && !containsSeason(out, season.URL) {
				out = append(out, season)
			}
		}
	}
	return out
}

func containsSeason(items []seasonLink, targetURL string) bool {
	for _, item := range items {
		if item.URL == targetURL {
			return true
		}
	}
	return false
}

func containsURL(items []fixtureMeta, targetURL string) bool {
	for _, item := range items {
		if item.URL == targetURL {
			return true
		}
	}
	return false
}

func filenameFromLeagueURL(raw string) string {
	m := ligaIDRe.FindStringSubmatch(raw)
	if len(m) != 3 {
		return ""
	}
	return "league_" + m[2]
}

func filenameFromMatchURL(raw string) string {
	m := meczIDRe.FindStringSubmatch(raw)
	if len(m) != 2 {
		return ""
	}
	return "match_" + m[1]
}

func prioritizeLeagues(items []fixtureMeta) []fixtureMeta {
	priority := []string{"league_14072", "league_14073", "league_liga11233", "league_liga11234", "league_liga10550", "league_liga10551", "league_liga11235"}
	out := make([]fixtureMeta, 0, len(items))
	seen := map[string]struct{}{}

	for _, p := range priority {
		for _, item := range items {
			if item.Name != p {
				continue
			}
			out = append(out, item)
			seen[item.Name] = struct{}{}
		}
	}

	for _, item := range items {
		if _, ok := seen[item.Name]; ok {
			continue
		}
		out = append(out, item)
	}

	return out
}

func deterministicManifestStamp(fixtures []fixtureMeta) string {
	h := sha256.New()
	for _, fixture := range fixtures {
		_, _ = h.Write([]byte(fixture.Name))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(fixture.Kind))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(fixture.URL))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(fixture.Season))
		_, _ = h.Write([]byte("\n"))
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

func validateRequiredFixtures(fixtures []fixtureMeta, required map[string]struct{}) error {
	seen := map[string]struct{}{}
	for _, fixture := range fixtures {
		seen[fixture.Name] = struct{}{}
	}

	missing := make([]string, 0, len(required))
	for name := range required {
		if _, ok := seen[name]; ok {
			continue
		}
		missing = append(missing, name)
	}
	sort.Strings(missing)

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("missing required fixtures: %s", strings.Join(missing, ", "))
}
