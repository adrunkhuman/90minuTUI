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

func (s *Service) LoadLeague(ctx context.Context, leagueURL string) (*LeaguePage, error) {
	doc, err := s.client.Document(ctx, leagueURL)
	if err != nil {
		return nil, err
	}

	league := parseLeaguePage(doc, s.client.Resolve(leagueURL))
	if league == nil {
		return nil, fmt.Errorf("league parse: no round fixtures found")
	}

	return league, nil
}

func parseSeasons(doc *goquery.Document, c *Client) ([]Season, int) {
	seasons := make([]Season, 0, 80)
	selectedIdx := -1

	doc.Find("select[name='urljump'] option").Each(func(i int, s *goquery.Selection) {
		rawURL, ok := s.Attr("value")
		if !ok || strings.TrimSpace(rawURL) == "" {
			return
		}

		season := Season{
			Label: strings.TrimSpace(s.Text()),
			URL:   c.Resolve(rawURL),
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

		links = append(links, Competition{Name: name, URL: absoluteURL})
	})

	return links
}

func isLeagueLikeURL(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(lower, "/liga/") && strings.HasSuffix(lower, ".html")
}
