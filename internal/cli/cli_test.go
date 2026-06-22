package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

type fakeService struct {
	archiveURL string
	menuURL    string
	leagueURL  string
	matchURL   string
	leagueErrs map[string]error

	archiveErr error
	leagueErr  error
	matchErr   error
}

func (s *fakeService) LoadArchive(_ context.Context, archiveURL string) ([]site.Season, int, []site.Competition, error) {
	s.archiveURL = archiveURL
	if s.archiveErr != nil {
		return nil, -1, nil, s.archiveErr
	}
	return []site.Season{{Label: "2025/26", URL: site.BaseURL + "/archsezon.php?id_sezon=107", SeasonID: "107", Current: true}}, 0, []site.Competition{{Name: "Ekstraklasa", URL: site.BaseURL + "/liga/1/liga14072.html", LeagueKey: "liga14072"}}, nil
}

func (s *fakeService) LoadCompetition(_ context.Context, competitionURL string) (*site.CompetitionMenu, *site.LeaguePage, error) {
	s.menuURL = competitionURL
	return &site.CompetitionMenu{
		Title: "III liga 2024/25",
		URL:   site.BaseURL + "/ligireg.php?poziom=4&id_sezon=105",
		Items: []site.Competition{{Name: "III liga, gr. I", URL: site.BaseURL + "/liga/1/liga13507.html", LeagueKey: "liga13507"}},
	}, nil, nil
}

func (s *fakeService) LoadLeague(_ context.Context, leagueURL string) (*site.LeaguePage, error) {
	s.leagueURL = leagueURL
	if err := s.leagueErrs[leagueURL]; err != nil {
		return nil, err
	}
	if s.leagueErr != nil {
		return nil, s.leagueErr
	}
	return &site.LeaguePage{
		Title:     "Ekstraklasa",
		URL:       site.BaseURL + "/liga/1/liga14072.html",
		LeagueKey: "liga14072",
		Standings: []site.StandingRow{{Position: 1, Team: "Team A", Played: 1, Won: 1, Points: 3}},
		Rounds:    []site.Round{{Name: "1. kolejka", Fixtures: []site.Fixture{{Home: "Team A", Away: "Team B", Score: "1-0", WhenInfo: "18 lipca, 18:00", MatchURL: site.BaseURL + "/mecz.php?id_mecz=1930640", MatchID: "1930640"}}}},
	}, nil
}

func (s *fakeService) LoadMatch(_ context.Context, matchURL string) (*site.MatchPage, error) {
	s.matchURL = matchURL
	if s.matchErr != nil {
		return nil, s.matchErr
	}
	return &site.MatchPage{
		URL:        site.BaseURL + "/mecz.php?id_mecz=1930640",
		MatchID:    "1930640",
		HomeTeam:   "Team A",
		AwayTeam:   "Team B",
		Score:      "1-0",
		Events:     []site.MatchEvent{{MinuteText: "35", Minute: 35, HasMinute: true, Kind: site.EventKindGoal, TeamSide: site.TeamSideHome, Text: "Player 35"}},
		HomeLineup: []site.PlayerLine{{Name: "Player"}},
	}, nil
}

func TestRunSeasonsExportsJSON(t *testing.T) {
	svc := &fakeService{}
	stdout, stderr, code := runTestCLI([]string{"seasons"}, svc)
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got seasonsOutput
	decodeJSON(t, stdout, &got)
	if len(got.Seasons) != 1 || got.Seasons[0].SeasonID != "107" || !got.Seasons[0].Current {
		t.Fatalf("unexpected seasons output: %+v", got)
	}
	if svc.archiveURL != site.ArchiveURL {
		t.Fatalf("unexpected archive URL: %q", svc.archiveURL)
	}
}

func TestRunCompetitionsNormalizesSeasonID(t *testing.T) {
	svc := &fakeService{}
	stdout, stderr, code := runTestCLI([]string{"competitions", "--season", "107"}, svc)
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got competitionsOutput
	decodeJSON(t, stdout, &got)
	if got.SeasonID != "107" || len(got.Competitions) != 1 || got.Competitions[0].LeagueKey != "liga14072" {
		t.Fatalf("unexpected competitions output: %+v", got)
	}
	if got.Title != "Competitions" || got.URL == "" {
		t.Fatalf("expected competitions metadata, got %+v", got)
	}
	if svc.archiveURL != site.BaseURL+"/archsezon.php?id_sezon=107" {
		t.Fatalf("unexpected archive URL: %q", svc.archiveURL)
	}
}

func TestRunCompetitionsExpandsSubmenuURL(t *testing.T) {
	svc := &fakeService{}
	stdout, stderr, code := runTestCLI([]string{"competitions", "--url", site.BaseURL + "/ligireg.php?poziom=4&id_sezon=105"}, svc)
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got competitionsOutput
	decodeJSON(t, stdout, &got)
	if got.Title != "III liga 2024/25" || got.SeasonID != "105" || got.Competitions[0].LeagueKey != "liga13507" {
		t.Fatalf("unexpected submenu competitions output: %+v", got)
	}
	if svc.menuURL == "" {
		t.Fatalf("expected submenu URL to be loaded")
	}
}

func TestRunLeagueNormalizesLeagueKey(t *testing.T) {
	svc := &fakeService{}
	stdout, stderr, code := runTestCLI([]string{"league", "liga14072"}, svc)
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got leagueJSON
	decodeJSON(t, stdout, &got)
	if got.LeagueKey != "liga14072" || len(got.Standings) != 1 || len(got.Rounds) != 1 {
		t.Fatalf("unexpected league output: %+v", got)
	}
	if svc.leagueURL != site.BaseURL+"/liga/1/liga14072.html" {
		t.Fatalf("unexpected league URL: %q", svc.leagueURL)
	}
}

func TestRunLeagueKeyFallsBackToLigaZeroPath(t *testing.T) {
	svc := &fakeService{leagueErrs: map[string]error{site.BaseURL + "/liga/1/liga8694.html": errors.New("missing")}}
	stdout, stderr, code := runTestCLI([]string{"league", "liga8694"}, svc)
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q stdout=%q", code, stderr, stdout)
	}
	if svc.leagueURL != site.BaseURL+"/liga/0/liga8694.html" {
		t.Fatalf("expected /liga/0/ fallback, got %q", svc.leagueURL)
	}
}

func TestRunFixturesExportsRoundsOnly(t *testing.T) {
	svc := &fakeService{}
	stdout, stderr, code := runTestCLI([]string{"fixtures", "liga14072"}, svc)
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got fixturesOutput
	decodeJSON(t, stdout, &got)
	if got.LeagueKey != "liga14072" || len(got.Rounds) != 1 || got.Rounds[0].Fixtures[0].MatchID != "1930640" {
		t.Fatalf("unexpected fixtures output: %+v", got)
	}
}

func TestRunMatchNormalizesMatchID(t *testing.T) {
	svc := &fakeService{}
	stdout, stderr, code := runTestCLI([]string{"match", "1930640"}, svc)
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got matchJSON
	decodeJSON(t, stdout, &got)
	if got.MatchID != "1930640" || got.Events[0].Kind != site.EventKindGoal || got.HomeLineup[0].Name != "Player" {
		t.Fatalf("unexpected match output: %+v", got)
	}
	if svc.matchURL != site.BaseURL+"/mecz.php?id_mecz=1930640" {
		t.Fatalf("unexpected match URL: %q", svc.matchURL)
	}
}

func TestRunMatchUsesArrayForEmptyPlayerEvents(t *testing.T) {
	stdout, stderr, code := runTestCLI([]string{"match", "1930640"}, &fakeService{})
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got map[string]any
	decodeJSON(t, stdout, &got)
	homeLineup := got["home_lineup"].([]any)
	player := homeLineup[0].(map[string]any)
	if events, ok := player["events"].([]any); !ok || len(events) != 0 {
		t.Fatalf("expected player events to be [], got %#v", player["events"])
	}
}

func TestRunFixturesContractFields(t *testing.T) {
	stdout, stderr, code := runTestCLI([]string{"fixtures", "liga14072"}, &fakeService{})
	if code != 0 || stderr != "" {
		t.Fatalf("expected success, code=%d stderr=%q", code, stderr)
	}
	var got map[string]any
	decodeJSON(t, stdout, &got)
	for _, key := range []string{"league_key", "url", "rounds"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("expected fixtures output key %q in %#v", key, got)
		}
	}
	rounds := got["rounds"].([]any)
	round := rounds[0].(map[string]any)
	for _, key := range []string{"phase", "section", "name", "fixtures"} {
		if _, ok := round[key]; !ok {
			t.Fatalf("expected round key %q in %#v", key, round)
		}
	}
}

func TestRunReportsErrorsOnStderr(t *testing.T) {
	svc := &fakeService{leagueErr: errors.New("league failed")}
	stdout, stderr, code := runTestCLI([]string{"league", "liga14072"}, svc)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "league failed") {
		t.Fatalf("expected CLI error, code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestRunRequiresSeasonForCompetitions(t *testing.T) {
	stdout, stderr, code := runTestCLI([]string{"competitions"}, &fakeService{})
	if code != 1 || stdout != "" || !strings.Contains(stderr, "--season or --url") {
		t.Fatalf("expected missing season error, code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func runTestCLI(args []string, svc Service) (string, string, int) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, &stdout, &stderr, svc)
	return stdout.String(), stderr.String(), code
}

func decodeJSON(t *testing.T, value string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(value), target); err != nil {
		t.Fatalf("decode JSON %q: %v", value, err)
	}
}
