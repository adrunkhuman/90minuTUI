package site

import (
	"context"
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
