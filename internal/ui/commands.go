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
const selectorSeasonLoadDelay = 300 * time.Millisecond

func (m Model) loadArchiveCmd(url string) tea.Cmd {
	return m.loadArchive(url, false)
}

func (m Model) loadArchiveFreshCmd(url string) tea.Cmd {
	return m.loadArchive(url, true)
}

func (m Model) loadArchive(url string, fresh bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		seasons, selectedIdx, competitions, err := m.loadArchivePage(ctx, url, fresh)
		return archiveLoadedMsg{seasons: seasons, selectedIdx: selectedIdx, competitions: competitions, fresh: fresh, err: err}
	}
}

// selectorOnly refreshes the selector list without auto-loading a competition or closing the selector.
func (m Model) loadSeasonCompetitionsCmd(url, seasonKey string, selectorOnly bool) tea.Cmd {
	return m.loadSeasonCompetitions(url, seasonKey, selectorOnly, false)
}

func (m Model) loadSeasonCompetitionsFreshCmd(url, seasonKey string, selectorOnly bool) tea.Cmd {
	return m.loadSeasonCompetitions(url, seasonKey, selectorOnly, true)
}

func (m Model) loadSeasonCompetitions(url, seasonKey string, selectorOnly bool, fresh bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		_, _, competitions, err := m.loadArchivePage(ctx, url, fresh)
		return competitionsLoadedMsg{seasonKey: seasonKey, competitions: competitions, selectorOnly: selectorOnly, fresh: fresh, err: err}
	}
}

func (m Model) settleSeasonSelectionCmd(url, seasonKey string) tea.Cmd {
	return tea.Tick(selectorSeasonLoadDelay, func(time.Time) tea.Msg {
		return seasonSelectionSettledMsg{seasonKey: seasonKey, seasonURL: url}
	})
}

func (m Model) loadCompetitionCmd(url, competitionKey string) tea.Cmd {
	return m.loadCompetition(url, competitionKey, false)
}

func (m Model) loadCompetitionFreshCmd(url, competitionKey string) tea.Cmd {
	return m.loadCompetition(url, competitionKey, true)
}

func (m Model) loadCompetition(url, competitionKey string, fresh bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		menu, league, err := m.loadCompetitionPage(ctx, url, fresh)
		return competitionMenuLoadedMsg{competitionKey: competitionKey, menu: menu, league: league, err: err}
	}
}

func (m Model) loadMatchCmd(url, fixtureKey string) tea.Cmd {
	return m.loadMatch(url, fixtureKey, false)
}

func (m Model) loadMatchFreshCmd(url, fixtureKey string) tea.Cmd {
	return m.loadMatch(url, fixtureKey, true)
}

func (m Model) loadMatch(url, fixtureKey string, fresh bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		match, err := m.loadMatchPage(ctx, url, fresh)
		return matchLoadedMsg{fixtureKey: fixtureKey, match: match, err: err}
	}
}

func (m Model) loadArchivePage(ctx context.Context, url string, fresh bool) ([]site.Season, int, []site.Competition, error) {
	if fresh {
		if loader, ok := m.service.(freshLoader); ok {
			return loader.LoadArchiveFresh(ctx, url)
		}
	}
	return m.service.LoadArchive(ctx, url)
}

func (m Model) loadCompetitionPage(ctx context.Context, url string, fresh bool) (*site.CompetitionMenu, *site.LeaguePage, error) {
	if fresh {
		if loader, ok := m.service.(freshLoader); ok {
			return loader.LoadCompetitionFresh(ctx, url)
		}
	}
	return m.service.LoadCompetition(ctx, url)
}

func (m Model) loadMatchPage(ctx context.Context, url string, fresh bool) (*site.MatchPage, error) {
	if fresh {
		if loader, ok := m.service.(freshLoader); ok {
			return loader.LoadMatchFresh(ctx, url)
		}
	}
	return m.service.LoadMatch(ctx, url)
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
