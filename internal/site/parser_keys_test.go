package site

import "testing"

func TestExtractStableKeysFromURLs(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		seasonID  string
		matchID   string
		leagueKey string
	}{
		{
			name:      "season archive URL",
			url:       "http://www.90minut.pl/archsezon.php?id_sezon=97",
			seasonID:  "97",
			leagueKey: "www.90minut.pl/archsezon.php?id_sezon=97",
		},
		{
			name:      "match URL",
			url:       "http://www.90minut.pl/mecz.php?id_mecz=2022810",
			matchID:   "2022810",
			leagueKey: "www.90minut.pl/mecz.php?id_mecz=2022810",
		},
		{
			name:      "league URL",
			url:       "http://www.90minut.pl/liga/1/liga11233.html",
			leagueKey: "liga11233",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractSeasonID(tc.url); got != tc.seasonID {
				t.Fatalf("extractSeasonID(%q)=%q want %q", tc.url, got, tc.seasonID)
			}
			if got := extractMatchID(tc.url); got != tc.matchID {
				t.Fatalf("extractMatchID(%q)=%q want %q", tc.url, got, tc.matchID)
			}
			if got := extractLeagueKey(tc.url); got != tc.leagueKey {
				t.Fatalf("extractLeagueKey(%q)=%q want %q", tc.url, got, tc.leagueKey)
			}
		})
	}
}
