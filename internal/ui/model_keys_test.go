package ui

import (
	"testing"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

func TestRequestKeysPreferStableIDs(t *testing.T) {
	season := site.Season{URL: "http://www.90minut.pl/archsezon.php?id_sezon=97", SeasonID: "97"}
	if got := seasonRequestKey(season); got != "season:97" {
		t.Fatalf("unexpected season request key: %q", got)
	}

	competition := site.Competition{URL: "http://www.90minut.pl/liga/1/liga11233.html", LeagueKey: "liga11233"}
	if got := competitionRequestKey(competition); got != "league:liga11233" {
		t.Fatalf("unexpected competition request key: %q", got)
	}

	fixture := site.Fixture{MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2022810", MatchID: "2022810"}
	if got := fixtureRequestKey(fixture); got != "match:2022810" {
		t.Fatalf("unexpected fixture request key: %q", got)
	}
}

func TestRequestKeysFallbackToURLs(t *testing.T) {
	season := site.Season{URL: "http://www.90minut.pl/archsezon.php?id_sezon=97"}
	if got := seasonRequestKey(season); got != "season-url:http://www.90minut.pl/archsezon.php?id_sezon=97" {
		t.Fatalf("unexpected season fallback key: %q", got)
	}

	competition := site.Competition{URL: "http://www.90minut.pl/liga/1/liga11233.html"}
	if got := competitionRequestKey(competition); got != "league-url:http://www.90minut.pl/liga/1/liga11233.html" {
		t.Fatalf("unexpected competition fallback key: %q", got)
	}

	fixture := site.Fixture{MatchURL: "http://www.90minut.pl/mecz.php?id_mecz=2022810"}
	if got := fixtureRequestKey(fixture); got != "match-url:http://www.90minut.pl/mecz.php?id_mecz=2022810" {
		t.Fatalf("unexpected fixture fallback key: %q", got)
	}
}
