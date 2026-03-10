package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

func TestLeagueSketchViewShowsStandingsFixturesAndStatus(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.lastFetchAt = time.Date(2026, time.March, 10, 21, 15, 0, 0, time.UTC)

	view := m.View()
	for _, want := range []string{
		"Standings",
		"# Team",
		"Legia Warszawa",
		"Round 1",
		"LEG 2-1 LEC",
		"fetched: 21:15:00",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q\n%s", want, view)
		}
	}
}

func TestMatchSketchViewShowsContextAndLoadingImmediately(t *testing.T) {
	m := sketchModel()
	m.width = 120
	m.matchView = true
	m.loading = true

	view := m.View()
	for _, want := range []string{
		"Context",
		"2024/2025",
		"Ekstraklasa",
		"Round 1",
		"Fixture",
		"LEG 2-1 LEC",
		"Loading match details...",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q\n%s", want, view)
		}
	}
}

func sketchModel() Model {
	return Model{
		width:  120,
		height: 40,
		focus:  focusFixtures,
		seasons: []site.Season{{
			Label:    "2024/2025",
			URL:      "http://www.90minut.pl/archsezon.php?id_sezon=101",
			SeasonID: "101",
			Current:  true,
		}},
		league: &site.LeaguePage{
			Title: "Ekstraklasa",
			Standings: []site.StandingRow{
				{Position: 1, Team: "Legia Warszawa", Played: 24, Won: 16, Drawn: 5, Lost: 3, Points: 53},
				{Position: 2, Team: "Lech Poznan", Played: 24, Won: 15, Drawn: 4, Lost: 5, Points: 49},
			},
			Rounds: []site.Round{{
				Name: "1. kolejka",
				Fixtures: []site.Fixture{
					{Home: "Legia Warszawa", Away: "Lech Poznan", Score: "2-1", WhenInfo: "Fri 20:30", MatchID: "1", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=1"},
					{Home: "Rakow Czestochowa", Away: "Pogon Szczecin", Score: "1-1", MatchID: "2", MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2"},
				},
			}},
		},
	}
}
