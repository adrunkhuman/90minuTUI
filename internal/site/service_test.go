package site

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
