package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

func (m Model) loadArchiveCmd(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		seasons, selectedIdx, competitions, err := m.service.LoadArchive(ctx, url)
		return archiveLoadedMsg{seasons: seasons, selectedIdx: selectedIdx, competitions: competitions, err: err}
	}
}

func (m Model) loadSeasonCompetitionsCmd(url, seasonKey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, _, competitions, err := m.service.LoadArchive(ctx, url)
		return competitionsLoadedMsg{seasonKey: seasonKey, competitions: competitions, err: err}
	}
}

func (m Model) loadLeagueCmd(url, competitionKey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		league, err := m.service.LoadLeague(ctx, url)
		return leagueLoadedMsg{competitionKey: competitionKey, league: league, err: err}
	}
}

func (m Model) loadMatchCmd(url, fixtureKey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		match, err := m.service.LoadMatch(ctx, url)
		return matchLoadedMsg{fixtureKey: fixtureKey, match: match, err: err}
	}
}

func seasonRequestKey(season site.Season) string {
	if season.SeasonID != "" {
		return "season:" + season.SeasonID
	}
	return "season-url:" + strings.TrimSpace(season.URL)
}

func competitionRequestKey(competition site.Competition) string {
	if competition.LeagueKey != "" {
		return "league:" + competition.LeagueKey
	}
	return "league-url:" + strings.TrimSpace(competition.URL)
}

func fixtureRequestKey(fixture site.Fixture) string {
	if fixture.MatchID != "" {
		return "match:" + fixture.MatchID
	}
	return "match-url:" + strings.TrimSpace(fixture.MatchURL)
}
