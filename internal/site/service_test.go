package site

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLoadLeagueAllowsFixturesWithoutMatchLinks(t *testing.T) {
	_, body := fixtureDoc(t, "fixtures/league_14141.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-2")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	svc := &Service{client: &Client{baseURL: baseURL, http: server.Client()}}
	page, err := svc.LoadLeague(context.Background(), server.URL+"/liga/1/liga14141.html")
	if err != nil {
		t.Fatalf("expected LoadLeague to succeed, got %v", err)
	}

	linklessFixtures := 0
	for _, round := range page.Rounds {
		for _, fixture := range round.Fixtures {
			if fixture.MatchURL != "" {
				continue
			}
			linklessFixtures++
			if fixture.MatchID != "" {
				t.Fatalf("expected empty match id for linkless fixture, got %q", fixture.MatchID)
			}
		}
	}

	if linklessFixtures == 0 {
		t.Fatalf("expected linkless fixtures in loaded league page")
	}
}

func TestLoadArchiveIncludesRegionalCupsCompetition(t *testing.T) {
	_, body := fixtureDoc(t, "fixtures/archive_2020_21.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-2")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	svc := &Service{client: &Client{baseURL: baseURL, http: server.Client()}}
	_, _, competitions, err := svc.LoadArchive(context.Background(), server.URL+"/archiwum/2020-21")
	if err != nil {
		t.Fatalf("expected LoadArchive to succeed, got %v", err)
	}

	for _, competition := range competitions {
		if competition.URL == server.URL+"/polcups.php?id_sezon=97" {
			return
		}
	}

	t.Fatalf("expected regional cups competition in archive: %+v", competitions)
}

func TestLoadCompetitionLoadsRegionalCupsMenu(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<html><body><table class="main"><tr><td valign="top"><p align="center"><b>Puchary krajowe 2025/26</b></p><a href="/liga/1/liga14076.html" class="main">Puchar Polski</a><a href="/liga/1/liga14636.html" class="main">Puchar Polski 2025/2026, grupa: Lubuski ZPN</a><a href="/liga/1/liga14069.html" class="main">Puchar Polski 2025/2026, grupa: Lubuski ZPN - Gorzów Wielkopolski</a></td></tr></table></body></html>`)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	svc := &Service{client: &Client{baseURL: baseURL, http: server.Client()}}
	menu, league, err := svc.LoadCompetition(context.Background(), server.URL+"/polcups.php?id_sezon=107")
	if err == nil {
		if league != nil {
			t.Fatalf("expected submenu, got league %+v", league)
		}
		if menu == nil || len(menu.Items) != 2 {
			t.Fatalf("expected regional cups submenu, got %+v", menu)
		}
		return
	}
	t.Fatalf("expected submenu load to succeed, got %v", err)
}

func TestServiceLoadArchiveReportsEmptyArchive(t *testing.T) {
	svc := serviceWithHTML(t, `<html><body>empty</body></html>`)

	seasons, selectedIdx, competitions, err := svc.LoadArchive(context.Background(), "/archsezon.php")
	if err == nil || !strings.Contains(err.Error(), "archive parse: no seasons found") {
		t.Fatalf("expected no-seasons error, got %v", err)
	}
	if seasons != nil || selectedIdx != -1 || competitions != nil {
		t.Fatalf("expected empty archive result on parse failure, got seasons=%v selected=%d competitions=%v", seasons, selectedIdx, competitions)
	}
}

func TestServiceLoadArchivePropagatesClientError(t *testing.T) {
	svc := serviceWithClientError(t, "network down")

	_, _, _, err := svc.LoadArchive(context.Background(), "/archsezon.php")
	if err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected propagated archive client error, got %v", err)
	}
}

func TestServiceLoadArchiveReportsMissingCompetitions(t *testing.T) {
	svc := serviceWithHTML(t, `<html><body><select name="urljump"><option selected value="/archsezon.php?id_sezon=107">2025/26</option></select></body></html>`)

	seasons, selectedIdx, competitions, err := svc.LoadArchive(context.Background(), "/archsezon.php")
	if err == nil || !strings.Contains(err.Error(), "archive parse: no league links found") {
		t.Fatalf("expected no-competitions error, got %v", err)
	}
	if len(seasons) != 1 || selectedIdx != 0 || competitions != nil {
		t.Fatalf("expected partial archive result, got seasons=%v selected=%d competitions=%v", seasons, selectedIdx, competitions)
	}
}

func TestServiceLoadCompetitionReportsUnclassifiedPage(t *testing.T) {
	svc := serviceWithHTML(t, `<html><body>not a supported competition page</body></html>`)

	menu, league, err := svc.LoadCompetition(context.Background(), "/misc.html")
	if err == nil || !strings.Contains(err.Error(), "competition parse: no submenu or fixtures found") {
		t.Fatalf("expected unclassified competition error, got %v", err)
	}
	if menu != nil || league != nil {
		t.Fatalf("expected no competition result, got menu=%v league=%v", menu, league)
	}
}

func TestServiceLoadCompetitionPropagatesClientError(t *testing.T) {
	svc := serviceWithClientError(t, "competition fetch failed")

	menu, league, err := svc.LoadCompetition(context.Background(), "/liga/1/liga1.html")
	if err == nil || !strings.Contains(err.Error(), "competition fetch failed") {
		t.Fatalf("expected propagated competition client error, got %v", err)
	}
	if menu != nil || league != nil {
		t.Fatalf("expected no competition result, got menu=%v league=%v", menu, league)
	}
}

func TestServiceLoadLeagueReportsSubmenu(t *testing.T) {
	svc := serviceWithHTML(t, `<html><body><table class="main"><tr><td valign="top"><p align="center"><b>Puchary krajowe 2025/26</b></p><a href="/liga/1/liga14076.html" class="main">Puchar Polski</a><a href="/liga/1/liga14636.html" class="main">Puchar Polski 2025/2026, grupa: Lubuski ZPN</a><a href="/liga/1/liga14069.html" class="main">Puchar Polski 2025/2026, grupa: Lubuski ZPN - Gorzów Wielkopolski</a></td></tr></table></body></html>`)

	league, err := svc.LoadLeague(context.Background(), "/polcups.php?id_sezon=107")
	if err == nil || !strings.Contains(err.Error(), "league parse: competition is a submenu") {
		t.Fatalf("expected submenu error, got %v", err)
	}
	if league != nil {
		t.Fatalf("expected no league result, got %+v", league)
	}
}

func TestServiceLoadMatchReportsMissingDetails(t *testing.T) {
	svc := serviceWithHTML(t, `<html><head><title>Broken match</title></head><body>not a match page</body></html>`)

	match, err := svc.LoadMatch(context.Background(), "/mecz.php?id_mecz=1")
	if err == nil || !strings.Contains(err.Error(), "match parse: missing teams and score") {
		t.Fatalf("expected missing-details error, got %v", err)
	}
	if match == nil || match.Title != "Broken match" || !strings.HasSuffix(match.URL, "/mecz.php?id_mecz=1") || match.MatchID != "1" {
		t.Fatalf("expected populated partial match result on validation failure, got %+v", match)
	}
}

func TestServiceLoadMatchPropagatesClientError(t *testing.T) {
	svc := serviceWithClientError(t, "match fetch failed")

	match, err := svc.LoadMatch(context.Background(), "/mecz.php?id_mecz=1")
	if err == nil || !strings.Contains(err.Error(), "match fetch failed") {
		t.Fatalf("expected propagated match client error, got %v", err)
	}
	if match != nil {
		t.Fatalf("expected no match result, got %+v", match)
	}
}

func serviceWithHTML(t *testing.T, body string) *Service {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	return &Service{client: &Client{baseURL: baseURL, http: server.Client()}}
}

func serviceWithClientError(t *testing.T, message string) *Service {
	t.Helper()

	baseURL, err := url.Parse(BaseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}

	return &Service{client: &Client{
		baseURL: baseURL,
		http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("%s", message)
		})},
	}}
}
