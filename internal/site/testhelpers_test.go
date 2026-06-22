package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

type fixtureManifest struct {
	GeneratedAt string         `json:"generated_at"`
	Source      string         `json:"source"`
	Fixtures    []fixtureEntry `json:"fixtures"`
}

type fixtureEntry struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Season string `json:"season"`
	File   string `json:"file"`
	Note   string `json:"note"`
}

func loadManifest(t *testing.T) fixtureManifest {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var m fixtureManifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	return m
}

func fixtureDoc(t *testing.T, fixturePath string) (*goquery.Document, []byte) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(fixturePath)))
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixturePath, err)
	}

	doc, err := decodeAndParse(data, "")
	if err != nil {
		t.Fatalf("parse fixture %q: %v", fixturePath, err)
	}

	return doc, data
}

func fixturesByKind(m fixtureManifest, kind string) []fixtureEntry {
	out := make([]fixtureEntry, 0, len(m.Fixtures))
	for _, fixture := range m.Fixtures {
		if fixture.Kind == kind {
			out = append(out, fixture)
		}
	}
	return out
}

func hasPolishDiacritic(s string) bool {
	for _, r := range s {
		switch r {
		case 'ą', 'ć', 'ę', 'ł', 'ń', 'ó', 'ś', 'ź', 'ż', 'Ą', 'Ć', 'Ę', 'Ł', 'Ń', 'Ó', 'Ś', 'Ź', 'Ż':
			return true
		}
	}
	return false
}

func containsFixtureName(fixtures []fixtureEntry, name string) bool {
	for _, fixture := range fixtures {
		if strings.EqualFold(fixture.Name, name) {
			return true
		}
	}
	return false
}

func competitionIndexByURL(competitions []Competition, url string) int {
	for i, competition := range competitions {
		if competition.URL == url {
			return i
		}
	}
	return -1
}
