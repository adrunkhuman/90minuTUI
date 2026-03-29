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

func (s *Service) LoadCompetition(ctx context.Context, competitionURL string) (*CompetitionMenu, *LeaguePage, error) {
	doc, err := s.client.Document(ctx, competitionURL)
	if err != nil {
		return nil, nil, err
	}

	resolvedURL := s.client.Resolve(competitionURL)
	var leagueErr error
	if league := parseLeaguePage(doc, resolvedURL); league != nil {
		league.LeagueKey = extractLeagueKey(league.URL)

		for i := range league.Rounds {
			for j := range league.Rounds[i].Fixtures {
				if league.Rounds[i].Fixtures[j].MatchURL == "" {
					league.Rounds[i].Fixtures[j].MatchID = ""
					continue
				}
				league.Rounds[i].Fixtures[j].MatchURL = s.client.Resolve(league.Rounds[i].Fixtures[j].MatchURL)
				league.Rounds[i].Fixtures[j].MatchID = extractMatchID(league.Rounds[i].Fixtures[j].MatchURL)
			}
		}

		if err := validateLeaguePage(league); err == nil {
			return nil, league, nil
		} else {
			leagueErr = err
		}
	}

	menu := parseCompetitionMenu(doc, resolvedURL, s.client)
	if menu == nil || len(menu.Items) == 0 {
		if leagueErr != nil {
			return nil, nil, leagueErr
		}
		return nil, nil, fmt.Errorf("competition parse: no submenu or fixtures found")
	}

	return menu, nil, nil
}

// LoadLeague fetches and parses a league page into a normalized LeaguePage.
// Rounds are ordered by detected round number when available, and fixtures
// inside each round are ordered by parsed date/time in the site layer.
func (s *Service) LoadLeague(ctx context.Context, leagueURL string) (*LeaguePage, error) {
	_, league, err := s.LoadCompetition(ctx, leagueURL)
	if err != nil {
		return nil, err
	}
	if league == nil {
		return nil, fmt.Errorf("league parse: competition is a submenu")
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

	// Archive pages mix direct league pages with seasonal regional overviews.
	doc.Find("a.main").Each(func(_ int, s *goquery.Selection) {
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

func parseCompetitionMenu(doc *goquery.Document, resolvedURL string, c *Client) *CompetitionMenu {
	switch {
	case isIIIligaSelectorURL(resolvedURL):
		return parseIIIligaMenu(doc, resolvedURL, c)
	case isRegionalRootURL(resolvedURL):
		return parseRegionalAssociationMenu(doc, resolvedURL, c)
	case isRegionalAssociationURL(resolvedURL):
		return parseRegionalLeagueMenu(doc, resolvedURL, c)
	case isRegionalCupsURL(resolvedURL):
		return parseRegionalCupMenu(doc, resolvedURL, c)
	default:
		return nil
	}
}

func parseIIIligaMenu(doc *goquery.Document, resolvedURL string, c *Client) *CompetitionMenu {
	title := competitionMenuTitle(doc, "III liga")
	items := make([]Competition, 0, 4)
	seen := map[string]struct{}{}

	doc.Find("a.main[href*='/liga/']").Each(func(_ int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Text())
		if name == "" {
			return
		}
		switch name {
		case "I", "II", "III", "IV":
		default:
			return
		}

		rawURL, ok := s.Attr("href")
		if !ok || !isLeagueLikeURL(rawURL) {
			return
		}

		absoluteURL := c.Resolve(strings.TrimSpace(rawURL))
		if _, exists := seen[absoluteURL]; exists {
			return
		}
		seen[absoluteURL] = struct{}{}
		items = append(items, Competition{Name: title + ", gr. " + name, URL: absoluteURL, LeagueKey: extractLeagueKey(absoluteURL)})
	})

	if len(items) == 0 {
		return nil
	}

	return &CompetitionMenu{Title: title, URL: resolvedURL, Items: items}
}

func parseRegionalAssociationMenu(doc *goquery.Document, resolvedURL string, c *Client) *CompetitionMenu {
	items := parseCompetitionLinks(doc.Find("a.main"), c, func(name string) bool {
		return name != ""
	}, func(rawURL string) bool {
		trimmed := strings.TrimSpace(rawURL)
		return strings.HasPrefix(trimmed, "/ligireg-") || isRegionalAssociationURL(trimmed)
	})
	if len(items) == 0 {
		return nil
	}

	return &CompetitionMenu{Title: competitionMenuTitle(doc, "Ligi regionalne"), URL: resolvedURL, Items: items}
}

func parseRegionalLeagueMenu(doc *goquery.Document, resolvedURL string, c *Client) *CompetitionMenu {
	items := parseCompetitionLinks(doc.Find("a.main"), c, func(name string) bool {
		return name != ""
	}, func(rawURL string) bool {
		return isLeagueLikeURL(rawURL) && strings.Contains(strings.ToLower(rawURL), "/liga/")
	})
	if len(items) == 0 {
		return nil
	}

	return &CompetitionMenu{Title: competitionMenuTitle(doc, "Ligi regionalne"), URL: resolvedURL, Items: items}
}

func parseRegionalCupMenu(doc *goquery.Document, resolvedURL string, c *Client) *CompetitionMenu {
	items := parseCompetitionLinks(doc.Find("a.main"), c, func(name string) bool {
		if name == "" || name == "Puchar Polski" || name == "Superpuchar Polski" {
			return false
		}
		return strings.Contains(strings.ToLower(name), "puchar") || strings.Contains(strings.ToLower(name), "superpuchar")
	}, func(rawURL string) bool {
		return isLeagueLikeURL(rawURL) && strings.Contains(strings.ToLower(rawURL), "/liga/")
	})
	if len(items) == 0 {
		return nil
	}

	return &CompetitionMenu{Title: competitionMenuTitle(doc, "Puchary regionalne"), URL: resolvedURL, Items: items}
}

func parseCompetitionLinks(selection *goquery.Selection, c *Client, keep func(string) bool, keepURL func(string) bool) []Competition {
	items := make([]Competition, 0, 32)
	seen := map[string]struct{}{}

	selection.Each(func(_ int, s *goquery.Selection) {
		rawURL, ok := s.Attr("href")
		if !ok || !keepURL(rawURL) {
			return
		}

		name := strings.TrimSpace(s.Text())
		if !keep(name) {
			return
		}

		absoluteURL := c.Resolve(strings.TrimSpace(rawURL))
		if _, exists := seen[absoluteURL]; exists {
			return
		}
		seen[absoluteURL] = struct{}{}
		items = append(items, Competition{Name: name, URL: absoluteURL, LeagueKey: extractLeagueKey(absoluteURL)})
	})

	return items
}

func competitionMenuTitle(doc *goquery.Document, fallback string) string {
	title := strings.TrimSpace(doc.Find("td[valign='top'] p[align='center'] b").First().Text())
	if title == "" {
		return fallback
	}
	return title
}

func isLeagueLikeURL(raw string) bool {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "/liga/") && strings.HasSuffix(lower, ".html") {
		return true
	}

	return strings.Contains(lower, "/ligireg.php") || strings.Contains(lower, "/polcups.php")
}

func isIIIligaSelectorURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.Contains(lower, "/ligireg.php") && strings.Contains(lower, "poziom=4")
}

func isRegionalRootURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if !strings.Contains(lower, "/ligireg") || strings.Contains(lower, "poziom=") {
		return false
	}
	if strings.Contains(lower, "id_okreg=") || strings.Contains(lower, "/ligireg-") {
		return false
	}
	return strings.Contains(lower, "id_sezon=") || strings.HasSuffix(lower, "/ligireg.html")
}

func isRegionalAssociationURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if strings.Contains(lower, "/ligireg-") {
		return true
	}
	return strings.Contains(lower, "/ligireg.php") && strings.Contains(lower, "id_okreg=")
}

func isRegionalCupsURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.Contains(lower, "/polcups.php")
}
