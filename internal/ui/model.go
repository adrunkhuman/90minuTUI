package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"90minut_go/internal/site"
)

type focusArea int

const (
	focusSeasons focusArea = iota
	focusCompetitions
)

type archiveLoadedMsg struct {
	seasons      []site.Season
	selectedIdx  int
	competitions []site.Competition
	err          error
}

type competitionsLoadedMsg struct {
	seasonIdx    int
	competitions []site.Competition
	err          error
}

type leagueLoadedMsg struct {
	competitionIdx int
	league         *site.LeaguePage
	err            error
}

type Model struct {
	service *site.Service

	width  int
	height int

	loading bool
	err     string
	focus   focusArea

	seasons      []site.Season
	seasonCursor int

	competitions      []site.Competition
	competitionCursor int

	league *site.LeaguePage
}

func NewModel(svc *site.Service) Model {
	return Model{
		service: svc,
		focus:   focusCompetitions,
		loading: true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadArchiveCmd("http://www.90minut.pl/archsezon.php")
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.toggleFocus()
			return m, nil
		case "up", "k":
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			return m, nil
		case "enter":
			return m, m.handleEnter()
		case "r":
			m.loading = true
			m.err = ""
			if len(m.seasons) == 0 {
				return m, m.loadArchiveCmd("http://www.90minut.pl/archsezon.php")
			}
			return m, m.loadSeasonCompetitionsCmd(m.seasons[m.seasonCursor].URL, m.seasonCursor)
		}

	case archiveLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}

		m.err = ""
		m.seasons = msg.seasons
		m.seasonCursor = clamp(msg.selectedIdx, 0, len(m.seasons)-1)
		m.competitions = msg.competitions
		m.competitionCursor = m.preferredCompetitionIndex()

		if len(m.competitions) == 0 {
			return m, nil
		}

		m.loading = true
		return m, m.loadLeagueCmd(m.competitions[m.competitionCursor].URL, m.competitionCursor)

	case competitionsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}

		if msg.seasonIdx != m.seasonCursor {
			return m, nil
		}

		m.err = ""
		m.competitions = msg.competitions
		m.competitionCursor = m.preferredCompetitionIndex()
		m.league = nil

		if len(m.competitions) == 0 {
			return m, nil
		}

		m.loading = true
		return m, m.loadLeagueCmd(m.competitions[m.competitionCursor].URL, m.competitionCursor)

	case leagueLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.league = nil
			return m, nil
		}

		if msg.competitionIdx != m.competitionCursor {
			return m, nil
		}

		m.err = ""
		m.league = msg.league
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading terminal size..."
	}

	leftWidth := max(36, m.width/2)
	rightWidth := max(36, m.width-leftWidth-2)

	left := m.leftPaneView(leftWidth)
	right := m.rightPaneView(rightWidth)

	if m.loading {
		right = right + "\n\nLoading..."
	}

	if m.err != "" {
		right = right + "\n\nError: " + m.err
	}

	help := "tab: switch focus  enter: load  j/k: move  r: reload  q: quit"

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return body + "\n\n" + help
}

func (m Model) leftPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	var b strings.Builder
	b.WriteString(title.Render("Season"))
	if m.focus == focusSeasons {
		b.WriteString(" " + focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")

	for _, line := range renderSeasonsWindow(m.seasons, m.seasonCursor) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(title.Render("Leagues"))
	if m.focus == focusCompetitions {
		b.WriteString(" " + focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")

	for _, line := range renderCompetitionWindow(m.competitions, m.competitionCursor) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) rightPaneView(width int) string {
	base := lipgloss.NewStyle().Width(width).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true)

	if m.league == nil {
		return base.Render("No league loaded yet")
	}

	var b strings.Builder
	b.WriteString(title.Render(m.league.Title))
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(m.league.URL))
	b.WriteString("\n\n")
	b.WriteString(title.Render("Latest round: " + m.league.LatestRound.Name))
	b.WriteString("\n")

	for _, f := range m.league.LatestRound.Fixtures {
		line := fmt.Sprintf("%s %s %s", f.Home, f.Score, f.Away)
		if f.WhenInfo != "" {
			line += " | " + f.WhenInfo
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m *Model) toggleFocus() {
	if m.focus == focusSeasons {
		m.focus = focusCompetitions
		return
	}

	m.focus = focusSeasons
}

func (m *Model) moveCursor(delta int) {
	if m.focus == focusSeasons {
		m.seasonCursor = clamp(m.seasonCursor+delta, 0, len(m.seasons)-1)
		return
	}

	m.competitionCursor = clamp(m.competitionCursor+delta, 0, len(m.competitions)-1)
}

func (m *Model) handleEnter() tea.Cmd {
	m.loading = true
	m.err = ""

	if m.focus == focusSeasons {
		if len(m.seasons) == 0 {
			m.loading = false
			return nil
		}
		return m.loadSeasonCompetitionsCmd(m.seasons[m.seasonCursor].URL, m.seasonCursor)
	}

	if len(m.competitions) == 0 {
		m.loading = false
		return nil
	}

	return m.loadLeagueCmd(m.competitions[m.competitionCursor].URL, m.competitionCursor)
}

func (m Model) preferredCompetitionIndex() int {
	if len(m.competitions) == 0 {
		return 0
	}

	for i, c := range m.competitions {
		if strings.Contains(strings.ToLower(c.Name), "ekstraklasa") {
			return i
		}
	}

	return 0
}

func (m Model) loadArchiveCmd(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		seasons, selectedIdx, competitions, err := m.service.LoadArchive(ctx, url)
		return archiveLoadedMsg{
			seasons:      seasons,
			selectedIdx:  selectedIdx,
			competitions: competitions,
			err:          err,
		}
	}
}

func (m Model) loadSeasonCompetitionsCmd(url string, seasonIdx int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		_, _, competitions, err := m.service.LoadArchive(ctx, url)
		return competitionsLoadedMsg{
			seasonIdx:    seasonIdx,
			competitions: competitions,
			err:          err,
		}
	}
}

func (m Model) loadLeagueCmd(url string, competitionIdx int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		league, err := m.service.LoadLeague(ctx, url)
		return leagueLoadedMsg{competitionIdx: competitionIdx, league: league, err: err}
	}
}

func renderSeasonsWindow(seasons []site.Season, cursor int) []string {
	if len(seasons) == 0 {
		return []string{"(none)"}
	}

	start, end := windowBounds(len(seasons), cursor, 10)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		marker := ""
		if seasons[i].Current {
			marker = " *"
		}

		lines = append(lines, fmt.Sprintf("%s%s%s", prefix, seasons[i].Label, marker))
	}

	return lines
}

func renderCompetitionWindow(items []site.Competition, cursor int) []string {
	if len(items) == 0 {
		return []string{"(none)"}
	}

	start, end := windowBounds(len(items), cursor, 18)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		lines = append(lines, prefix+items[i].Name)
	}

	return lines
}

func windowBounds(total, cursor, maxItems int) (int, int) {
	if total <= maxItems {
		return 0, total
	}

	half := maxItems / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}

	end := start + maxItems
	if end > total {
		end = total
		start = end - maxItems
	}

	return start, end
}

func clamp(v, minV, maxV int) int {
	if maxV < minV {
		return minV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
