package ui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

// cmdTimeout is generous enough for slow archive pages without blocking the TUI indefinitely.
const cmdTimeout = 20 * time.Second

func (m Model) loadArchiveCmd(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		seasons, selectedIdx, competitions, err := m.service.LoadArchive(ctx, url)
		return archiveLoadedMsg{seasons: seasons, selectedIdx: selectedIdx, competitions: competitions, err: err}
	}
}

// selectorOnly refreshes the selector list without auto-loading a competition or closing the selector.
func (m Model) loadSeasonCompetitionsCmd(url, seasonKey string, selectorOnly bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		_, _, competitions, err := m.service.LoadArchive(ctx, url)
		return competitionsLoadedMsg{seasonKey: seasonKey, competitions: competitions, selectorOnly: selectorOnly, err: err}
	}
}

func (m Model) loadCompetitionCmd(url, competitionKey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		menu, league, err := m.service.LoadCompetition(ctx, url)
		return competitionMenuLoadedMsg{competitionKey: competitionKey, menu: menu, league: league, err: err}
	}
}

func (m Model) loadMatchCmd(url, fixtureKey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
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
