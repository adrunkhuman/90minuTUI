package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

const timeout = 20 * time.Second

var leagueKeyRe = regexp.MustCompile(`(?i)^liga\d+$`)
var numericIDRe = regexp.MustCompile(`^\d+$`)

type Service interface {
	LoadArchive(ctx context.Context, archiveURL string) ([]site.Season, int, []site.Competition, error)
	LoadCompetition(ctx context.Context, competitionURL string) (*site.CompetitionMenu, *site.LeaguePage, error)
	LoadLeague(ctx context.Context, leagueURL string) (*site.LeaguePage, error)
	LoadMatch(ctx context.Context, matchURL string) (*site.MatchPage, error)
}

func IsCommand(name string) bool {
	switch name {
	case "seasons", "competitions", "league", "fixtures", "match", "help", "-h", "--help":
		return true
	default:
		return false
	}
}

func Run(args []string, stdout, stderr io.Writer, svc Service) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printUsage(stdout)
		return 0
	}

	if err := run(args, stdout, svc); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func run(args []string, stdout io.Writer, svc Service) error {
	switch args[0] {
	case "seasons":
		return runSeasons(args[1:], stdout, svc)
	case "competitions":
		return runCompetitions(args[1:], stdout, svc)
	case "league":
		return runLeague(args[1:], stdout, svc)
	case "fixtures":
		return runFixtures(args[1:], stdout, svc)
	case "match":
		return runMatch(args[1:], stdout, svc)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runSeasons(args []string, stdout io.Writer, svc Service) error {
	fs := newFlagSet("seasons")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("seasons does not accept positional arguments")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	seasons, _, _, err := svc.LoadArchive(ctx, site.ArchiveURL)
	if err != nil {
		return err
	}

	return writeJSON(stdout, seasonsOutput{Seasons: mapSeasons(seasons)})
}

func runCompetitions(args []string, stdout io.Writer, svc Service) error {
	fs := newFlagSet("competitions")
	season := fs.String("season", "", "season id or archive URL")
	competitionURL := fs.String("url", "", "competition submenu URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("competitions does not accept positional arguments")
	}
	if strings.TrimSpace(*season) == "" && strings.TrimSpace(*competitionURL) == "" {
		return errors.New("competitions requires --season or --url")
	}
	if strings.TrimSpace(*season) != "" && strings.TrimSpace(*competitionURL) != "" {
		return errors.New("competitions accepts only one of --season or --url")
	}
	if strings.TrimSpace(*competitionURL) != "" {
		return runCompetitionMenu(stdout, svc, *competitionURL)
	}

	archiveURL := seasonURL(*season)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, _, competitions, err := svc.LoadArchive(ctx, archiveURL)
	if err != nil {
		return err
	}

	return writeJSON(stdout, competitionsOutput{Title: "Competitions", SeasonID: seasonID(archiveURL), URL: archiveURL, Competitions: mapCompetitions(competitions)})
}

func runCompetitionMenu(stdout io.Writer, svc Service, rawURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	menu, league, err := svc.LoadCompetition(ctx, rawURL)
	if err != nil {
		return err
	}
	if menu == nil {
		if league != nil {
			return errors.New("competition URL is a terminal league; use league or fixtures")
		}
		return errors.New("competition URL returned no submenu")
	}

	return writeJSON(stdout, competitionsOutput{Title: menu.Title, SeasonID: seasonID(menu.URL), URL: menu.URL, Competitions: mapCompetitions(menu.Items)})
}

func runLeague(args []string, stdout io.Writer, svc Service) error {
	league, err := loadLeagueArg(args, svc)
	if err != nil {
		return err
	}
	return writeJSON(stdout, mapLeague(league, true))
}

func runFixtures(args []string, stdout io.Writer, svc Service) error {
	league, err := loadLeagueArg(args, svc)
	if err != nil {
		return err
	}
	return writeJSON(stdout, fixturesOutput{LeagueKey: league.LeagueKey, URL: league.URL, Rounds: mapRounds(league.Rounds)})
}

func runMatch(args []string, stdout io.Writer, svc Service) error {
	fs := newFlagSet("match")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("match requires match id or URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	match, err := svc.LoadMatch(ctx, matchURL(fs.Arg(0)))
	if err != nil {
		return err
	}
	return writeJSON(stdout, mapMatch(match))
}

func loadLeagueArg(args []string, svc Service) (*site.LeaguePage, error) {
	fs := newFlagSet("league")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 1 {
		return nil, errors.New("league requires league key or URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return loadLeague(ctx, svc, fs.Arg(0))
}

func loadLeague(ctx context.Context, svc Service, value string) (*site.LeaguePage, error) {
	candidates := leagueURLs(value)
	var lastErr error
	for _, candidate := range candidates {
		league, err := svc.LoadLeague(ctx, candidate)
		if err == nil {
			return league, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage:
  90minutui seasons
  90minutui competitions --season <season-id-or-url>
  90minutui competitions --url <competition-submenu-url>
  90minutui league <league-key-or-url>
  90minutui fixtures <league-key-or-url>
  90minutui match <match-id-or-url>
`)
}

func seasonURL(value string) string {
	value = strings.TrimSpace(value)
	if isAbsURL(value) {
		return value
	}
	if numericIDRe.MatchString(value) {
		return site.BaseURL + "/archsezon.php?id_sezon=" + value
	}
	return value
}

func leagueURL(value string) string {
	return leagueURLs(value)[0]
}

func leagueURLs(value string) []string {
	value = strings.TrimSpace(value)
	if isAbsURL(value) {
		return []string{value}
	}
	if leagueKeyRe.MatchString(value) {
		key := strings.ToLower(value)
		// Older 90minut league pages may live under /liga/0/.
		return []string{site.BaseURL + "/liga/1/" + key + ".html", site.BaseURL + "/liga/0/" + key + ".html"}
	}
	return []string{value}
}

func matchURL(value string) string {
	value = strings.TrimSpace(value)
	if isAbsURL(value) {
		return value
	}
	if numericIDRe.MatchString(value) {
		return site.BaseURL + "/mecz.php?id_mecz=" + value
	}
	return value
}

func isAbsURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.IsAbs()
}

func seasonID(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Query().Get("id_sezon"))
}

type seasonsOutput struct {
	Seasons []seasonJSON `json:"seasons"`
}

type seasonJSON struct {
	Label    string `json:"label"`
	SeasonID string `json:"season_id"`
	URL      string `json:"url"`
	Current  bool   `json:"current"`
}

type competitionsOutput struct {
	Title        string            `json:"title"`
	SeasonID     string            `json:"season_id"`
	URL          string            `json:"url"`
	Competitions []competitionJSON `json:"competitions"`
}

type competitionJSON struct {
	Name      string `json:"name"`
	LeagueKey string `json:"league_key"`
	URL       string `json:"url"`
}

type leagueJSON struct {
	Title     string         `json:"title"`
	LeagueKey string         `json:"league_key"`
	URL       string         `json:"url"`
	Standings []standingJSON `json:"standings"`
	Rounds    []roundJSON    `json:"rounds"`
}

type fixturesOutput struct {
	LeagueKey string      `json:"league_key"`
	URL       string      `json:"url"`
	Rounds    []roundJSON `json:"rounds"`
}

type standingJSON struct {
	Position int    `json:"position"`
	Team     string `json:"team"`
	Played   int    `json:"played"`
	Won      int    `json:"won"`
	Drawn    int    `json:"drawn"`
	Lost     int    `json:"lost"`
	Points   int    `json:"points"`
}

type roundJSON struct {
	Phase    site.RoundPhase `json:"phase"`
	Section  string          `json:"section"`
	Name     string          `json:"name"`
	Fixtures []fixtureJSON   `json:"fixtures"`
}

type fixtureJSON struct {
	Home     string `json:"home"`
	Away     string `json:"away"`
	Score    string `json:"score"`
	When     string `json:"when"`
	MatchID  string `json:"match_id"`
	MatchURL string `json:"match_url"`
}

type matchJSON struct {
	MatchID     string           `json:"match_id"`
	URL         string           `json:"url"`
	Title       string           `json:"title"`
	Competition string           `json:"competition"`
	Meta        string           `json:"meta"`
	Weather     string           `json:"weather"`
	HomeTeam    string           `json:"home_team"`
	AwayTeam    string           `json:"away_team"`
	Score       string           `json:"score"`
	Events      []matchEventJSON `json:"events"`
	HomeLineup  []playerLineJSON `json:"home_lineup"`
	AwayLineup  []playerLineJSON `json:"away_lineup"`
	NewsTitle   string           `json:"news_title"`
	NewsURL     string           `json:"news_url"`
}

type matchEventJSON struct {
	MinuteText      string              `json:"minute_text"`
	Minute          int                 `json:"minute"`
	Stoppage        int                 `json:"stoppage"`
	HasMinute       bool                `json:"has_minute"`
	Kind            site.MatchEventKind `json:"kind"`
	TeamSide        site.TeamSide       `json:"team_side"`
	Text            string              `json:"text"`
	SubstitutionOut string              `json:"substitution_out"`
	SubstitutionIn  string              `json:"substitution_in"`
}

type playerLineJSON struct {
	Name    string   `json:"name"`
	Events  []string `json:"events"`
	RawText string   `json:"raw_text"`
}

func mapSeasons(seasons []site.Season) []seasonJSON {
	out := make([]seasonJSON, 0, len(seasons))
	for _, season := range seasons {
		out = append(out, seasonJSON{Label: season.Label, SeasonID: season.SeasonID, URL: season.URL, Current: season.Current})
	}
	return out
}

func mapCompetitions(competitions []site.Competition) []competitionJSON {
	out := make([]competitionJSON, 0, len(competitions))
	for _, competition := range competitions {
		out = append(out, competitionJSON{Name: competition.Name, LeagueKey: competition.LeagueKey, URL: competition.URL})
	}
	return out
}

func mapLeague(league *site.LeaguePage, includeRounds bool) leagueJSON {
	out := leagueJSON{Title: league.Title, LeagueKey: league.LeagueKey, URL: league.URL, Standings: mapStandings(league.Standings), Rounds: []roundJSON{}}
	if includeRounds {
		out.Rounds = mapRounds(league.Rounds)
	}
	return out
}

func mapStandings(rows []site.StandingRow) []standingJSON {
	out := make([]standingJSON, 0, len(rows))
	for _, row := range rows {
		out = append(out, standingJSON{Position: row.Position, Team: row.Team, Played: row.Played, Won: row.Won, Drawn: row.Drawn, Lost: row.Lost, Points: row.Points})
	}
	return out
}

func mapRounds(rounds []site.Round) []roundJSON {
	out := make([]roundJSON, 0, len(rounds))
	for _, round := range rounds {
		out = append(out, roundJSON{Phase: round.Phase, Section: round.Section, Name: round.Name, Fixtures: mapFixtures(round.Fixtures)})
	}
	return out
}

func mapFixtures(fixtures []site.Fixture) []fixtureJSON {
	out := make([]fixtureJSON, 0, len(fixtures))
	for _, fixture := range fixtures {
		out = append(out, fixtureJSON{Home: fixture.Home, Away: fixture.Away, Score: fixture.Score, When: fixture.WhenInfo, MatchID: fixture.MatchID, MatchURL: fixture.MatchURL})
	}
	return out
}

func mapMatch(match *site.MatchPage) matchJSON {
	return matchJSON{
		MatchID:     match.MatchID,
		URL:         match.URL,
		Title:       match.Title,
		Competition: match.Competition,
		Meta:        match.Meta,
		Weather:     match.Weather,
		HomeTeam:    match.HomeTeam,
		AwayTeam:    match.AwayTeam,
		Score:       match.Score,
		Events:      mapMatchEvents(match.Events),
		HomeLineup:  mapPlayerLines(match.HomeLineup),
		AwayLineup:  mapPlayerLines(match.AwayLineup),
		NewsTitle:   match.NewsTitle,
		NewsURL:     match.NewsURL,
	}
}

func mapMatchEvents(events []site.MatchEvent) []matchEventJSON {
	out := make([]matchEventJSON, 0, len(events))
	for _, event := range events {
		out = append(out, matchEventJSON{
			MinuteText:      event.MinuteText,
			Minute:          event.Minute,
			Stoppage:        event.Stoppage,
			HasMinute:       event.HasMinute,
			Kind:            event.Kind,
			TeamSide:        event.TeamSide,
			Text:            event.Text,
			SubstitutionOut: event.SubstitutionOut,
			SubstitutionIn:  event.SubstitutionIn,
		})
	}
	return out
}

func mapPlayerLines(lines []site.PlayerLine) []playerLineJSON {
	out := make([]playerLineJSON, 0, len(lines))
	for _, line := range lines {
		// Keep the JSON contract at [] instead of null for players without events.
		events := make([]string, 0, len(line.Events))
		events = append(events, line.Events...)
		out = append(out, playerLineJSON{Name: line.Name, Events: events, RawText: line.RawText})
	}
	return out
}
