package site

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Service struct {
	client *Client
}

func NewService() *Service {
	return &Service{client: NewClient()}
}

func (s *Service) LoadArchive(ctx context.Context, archiveURL string) ([]Season, int, []Competition, error) {
	doc, err := s.client.Document(ctx, archiveURL)
	if err != nil {
		return nil, -1, nil, err
	}

	seasons, selectedIdx := parseSeasons(doc, s.client)
	competitions := parseCompetitions(doc, s.client)

	if len(seasons) == 0 {
		return nil, -1, nil, fmt.Errorf("archive parse: no seasons found")
	}

	if len(competitions) == 0 {
		return seasons, selectedIdx, nil, fmt.Errorf("archive parse: no league links found")
	}

	return seasons, selectedIdx, competitions, nil
}

// LoadLeague fetches and parses a league page into a normalized LeaguePage.
// Rounds are ordered by detected round number when available, and fixtures
// inside each round are ordered by parsed date/time in the site layer.
func (s *Service) LoadLeague(ctx context.Context, leagueURL string) (*LeaguePage, error) {
	doc, err := s.client.Document(ctx, leagueURL)
	if err != nil {
		return nil, err
	}

	league := parseLeaguePage(doc, s.client.Resolve(leagueURL))
	if league == nil {
		return nil, fmt.Errorf("league parse: no round fixtures found")
	}
	league.LeagueKey = extractLeagueKey(league.URL)

	for i := range league.Rounds {
		for j := range league.Rounds[i].Fixtures {
			league.Rounds[i].Fixtures[j].MatchURL = s.client.Resolve(league.Rounds[i].Fixtures[j].MatchURL)
			league.Rounds[i].Fixtures[j].MatchID = extractMatchID(league.Rounds[i].Fixtures[j].MatchURL)
		}
	}

	if err := validateLeaguePage(league); err != nil {
		return league, err
	}

	return league, nil
}

func (s *Service) LoadMatch(ctx context.Context, matchURL string) (*MatchPage, error) {
	doc, err := s.client.Document(ctx, matchURL)
	if err != nil {
		return nil, err
	}

	match := parseMatchPage(doc, s.client.Resolve(matchURL))
	if match == nil {
		return nil, fmt.Errorf("match parse: no details found")
	}
	match.MatchID = extractMatchID(match.URL)
	if match.NewsURL != "" {
		match.NewsURL = s.client.Resolve(match.NewsURL)
	}

	if err := validateMatchPage(match); err != nil {
		return match, err
	}

	return match, nil
}

func validateLeaguePage(page *LeaguePage) error {
	if page == nil {
		return fmt.Errorf("league parse: empty page")
	}

	if len(page.Rounds) == 0 {
		return fmt.Errorf("league parse: no rounds found")
	}

	fixtureCount := 0
	for _, round := range page.Rounds {
		fixtureCount += len(round.Fixtures)
	}
	if fixtureCount == 0 {
		return fmt.Errorf("league parse: rounds found but fixtures are empty")
	}

	return nil
}

func validateMatchPage(page *MatchPage) error {
	if page == nil {
		return fmt.Errorf("match parse: empty page")
	}

	if page.HomeTeam == "" || page.AwayTeam == "" {
		if page.Score == "" && len(page.Events) == 0 && len(page.HomeLineup) == 0 && len(page.AwayLineup) == 0 {
			return fmt.Errorf("match parse: missing teams and score")
		}
	}

	return nil
}

// If archive markup has no selected option, we default to the first season.
func parseSeasons(doc *goquery.Document, c *Client) ([]Season, int) {
	seasons := make([]Season, 0, 80)
	selectedIdx := -1

	doc.Find("select[name='urljump'] option").Each(func(i int, s *goquery.Selection) {
		rawURL, ok := s.Attr("value")
		if !ok || strings.TrimSpace(rawURL) == "" {
			return
		}

		resolvedURL := c.Resolve(rawURL)

		season := Season{
			Label:    strings.TrimSpace(s.Text()),
			URL:      resolvedURL,
			SeasonID: extractSeasonID(resolvedURL),
		}

		if _, isSelected := s.Attr("selected"); isSelected {
			season.Current = true
			selectedIdx = len(seasons)
		}

		seasons = append(seasons, season)
	})

	if selectedIdx == -1 && len(seasons) > 0 {
		selectedIdx = 0
		seasons[0].Current = true
	}

	return seasons, selectedIdx
}

func parseCompetitions(doc *goquery.Document, c *Client) []Competition {
	links := make([]Competition, 0, 64)
	seen := map[string]struct{}{}

	// Archive pages render season-specific competition links in the central
	// content area as a.main href="/liga/..." entries.
	doc.Find("a.main[href*='/liga/']").Each(func(_ int, s *goquery.Selection) {
		rawURL, ok := s.Attr("href")
		if !ok {
			return
		}

		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" || !isLeagueLikeURL(rawURL) {
			return
		}

		absoluteURL := c.Resolve(rawURL)
		if _, exists := seen[absoluteURL]; exists {
			return
		}

		seen[absoluteURL] = struct{}{}

		name := strings.TrimSpace(s.Text())
		if name == "" {
			name = absoluteURL
		}

		links = append(links, Competition{Name: name, URL: absoluteURL, LeagueKey: extractLeagueKey(absoluteURL)})
	})

	return links
}

func isLeagueLikeURL(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(lower, "/liga/") && strings.HasSuffix(lower, ".html")
}
