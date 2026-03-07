package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

type focusArea int

const (
	focusSeasons focusArea = iota
	focusCompetitions
	focusFixtures
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

type matchLoadedMsg struct {
	matchURL string
	match    *site.MatchPage
	err      error
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

	league        *site.LeaguePage
	roundCursor   int
	fixtureCursor int

	matchView bool
	match     *site.MatchPage

	sidebarCollapsed bool
}

func NewModel(svc *site.Service) Model {
	return Model{service: svc, focus: focusCompetitions, loading: true}
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
		case "left", "h":
			m.shiftRound(-1)
			return m, nil
		case "right", "l":
			m.shiftRound(1)
			return m, nil
		case "enter":
			return m, m.handleEnter()
		case "s":
			m.sidebarCollapsed = !m.sidebarCollapsed
			return m, nil
		case "esc", "backspace":
			if m.matchView {
				m.matchView = false
				m.match = nil
				m.err = ""
			}
			return m, nil
		case "r":
			m.loading = true
			m.err = ""
			m.matchView = false
			m.match = nil
			if len(m.seasons) == 0 {
				return m, m.loadArchiveCmd("http://www.90minut.pl/archsezon.php")
			}
			if m.focus == focusSeasons {
				return m, m.loadSeasonCompetitionsCmd(m.seasons[m.seasonCursor].URL, m.seasonCursor)
			}
			if len(m.competitions) == 0 {
				m.loading = false
				return m, nil
			}
			return m, m.loadLeagueCmd(m.competitions[m.competitionCursor].URL, m.competitionCursor)
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
		m.matchView = false
		m.match = nil

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
		m.matchView = false
		m.match = nil
		m.league = msg.league
		m.roundCursor = clamp(len(msg.league.Rounds)-1, 0, len(msg.league.Rounds)-1)
		m.fixtureCursor = 0
		m.focus = focusFixtures
		return m, nil

	case matchLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.matchView = false
			m.match = nil
			return m, nil
		}

		current := m.currentFixture()
		if current == nil || current.MatchURL != msg.matchURL {
			return m, nil
		}

		m.err = ""
		m.matchView = true
		m.match = msg.match
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading terminal size..."
	}

	emphasizeRight := m.matchView || m.focus == focusFixtures
	leftWidth, rightWidth := layoutWidths(m.width, m.sidebarCollapsed, emphasizeRight)
	right := m.rightPaneView(rightWidth)
	left := ""
	if leftWidth > 0 {
		left = m.leftPaneView(leftWidth)
	}

	if m.loading {
		right += "\n\nLoading..."
	}
	if m.err != "" {
		right += "\n\nError: " + m.err
	}

	help := "tab: focus  enter: load/open  j/k: move  h/l: round  s: sidebar  esc: back  r: reload  q: quit"
	if left == "" {
		return right + "\n\n" + help
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n\n" + help
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
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	if m.matchView && m.match != nil {
		var b strings.Builder
		heading := m.match.Title
		if m.match.HomeTeam != "" && m.match.AwayTeam != "" && m.match.Score != "" {
			heading = fmt.Sprintf("%s %s %s", m.match.HomeTeam, m.match.Score, m.match.AwayTeam)
		}

		b.WriteString(title.Render(heading))
		b.WriteString("\n")
		if m.match.Competition != "" {
			b.WriteString(m.match.Competition)
			b.WriteString("\n")
		}
		if m.match.Meta != "" {
			b.WriteString(m.match.Meta)
			b.WriteString("\n")
		}
		if m.match.Weather != "" {
			b.WriteString("Pogoda: ")
			b.WriteString(m.match.Weather)
			b.WriteString("\n")
		}

		if len(m.match.Events) > 0 {
			b.WriteString("\n")
			b.WriteString(title.Render("Timeline"))
			b.WriteString("\n")

			events := sortedEvents(m.match.Events)
			for _, event := range events {
				homeText, awayText := "", ""
				eventText := formatEventLabel(event)
				if event.TeamSide == "home" {
					homeText = eventText
				} else {
					awayText = eventText
				}

				line := renderSideBySide(homeText, event.MinuteText, awayText, width-4)
				b.WriteString(line)
				b.WriteString("\n")
			}
		}

		if len(m.match.HomeLineup) > 0 || len(m.match.AwayLineup) > 0 {
			b.WriteString("\n")
			b.WriteString(title.Render("Lineups"))
			b.WriteString("\n")
			b.WriteString(renderSideBySide(m.match.HomeTeam, "", m.match.AwayTeam, width-4))
			b.WriteString("\n")

			maxPlayers := len(m.match.HomeLineup)
			if len(m.match.AwayLineup) > maxPlayers {
				maxPlayers = len(m.match.AwayLineup)
			}

			for i := 0; i < maxPlayers; i++ {
				homeText, awayText := "", ""
				if i < len(m.match.HomeLineup) {
					homeText = renderPlayerLine(m.match.HomeLineup[i])
				}
				if i < len(m.match.AwayLineup) {
					awayText = renderPlayerLine(m.match.AwayLineup[i])
				}

				b.WriteString(renderSideBySide(homeText, "", awayText, width-4))
				b.WriteString("\n")
			}
		}

		if m.match.NewsTitle != "" {
			b.WriteString("\n")
			b.WriteString(title.Render("Related news"))
			b.WriteString("\n")
			b.WriteString(m.match.NewsTitle)
			if m.match.NewsURL != "" {
				b.WriteString(" | ")
				b.WriteString(m.match.NewsURL)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n(esc to go back)")
		return base.Render(strings.TrimRight(b.String(), "\n"))
	}

	if m.league == nil || len(m.league.Rounds) == 0 {
		return base.Render("No league loaded yet")
	}

	round := m.currentRound()
	if round == nil {
		return base.Render("No fixtures in selected round")
	}

	var b strings.Builder
	b.WriteString(title.Render(m.league.Title))
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(m.league.URL))
	b.WriteString("\n\n")
	b.WriteString(title.Render(fmt.Sprintf("Round %d/%d: %s", m.roundCursor+1, len(m.league.Rounds), round.Name)))
	if m.focus == focusFixtures {
		b.WriteString(" " + focusStyle.Render("[focus]"))
	}
	b.WriteString("\n")

	for i, f := range round.Fixtures {
		prefix := "  "
		if i == m.fixtureCursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%s %s %s", prefix, f.Home, f.Score, f.Away)
		if f.WhenInfo != "" {
			line += " | " + f.WhenInfo
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return base.Render(strings.TrimRight(b.String(), "\n"))
}

func (m *Model) toggleFocus() {
	order := []focusArea{focusSeasons, focusCompetitions}
	if m.league != nil && len(m.league.Rounds) > 0 {
		order = append(order, focusFixtures)
	}

	for i := range order {
		if order[i] != m.focus {
			continue
		}
		m.focus = order[(i+1)%len(order)]
		return
	}

	m.focus = focusSeasons
}

func (m *Model) moveCursor(delta int) {
	if m.matchView {
		return
	}

	switch m.focus {
	case focusSeasons:
		m.seasonCursor = clamp(m.seasonCursor+delta, 0, len(m.seasons)-1)
	case focusCompetitions:
		m.competitionCursor = clamp(m.competitionCursor+delta, 0, len(m.competitions)-1)
	case focusFixtures:
		round := m.currentRound()
		if round == nil {
			return
		}
		m.fixtureCursor = clamp(m.fixtureCursor+delta, 0, len(round.Fixtures)-1)
	}
}

func (m *Model) shiftRound(delta int) {
	if m.matchView || m.focus != focusFixtures || m.league == nil {
		return
	}

	m.roundCursor = clamp(m.roundCursor+delta, 0, len(m.league.Rounds)-1)
	m.fixtureCursor = 0
}

func (m *Model) handleEnter() tea.Cmd {
	m.loading = true
	m.err = ""

	switch m.focus {
	case focusSeasons:
		if len(m.seasons) == 0 {
			m.loading = false
			return nil
		}
		m.matchView = false
		m.match = nil
		return m.loadSeasonCompetitionsCmd(m.seasons[m.seasonCursor].URL, m.seasonCursor)

	case focusCompetitions:
		if len(m.competitions) == 0 {
			m.loading = false
			return nil
		}
		m.matchView = false
		m.match = nil
		return m.loadLeagueCmd(m.competitions[m.competitionCursor].URL, m.competitionCursor)

	case focusFixtures:
		fixture := m.currentFixture()
		if fixture == nil {
			m.loading = false
			return nil
		}
		return m.loadMatchCmd(fixture.MatchURL)
	}

	m.loading = false
	return nil
}

func (m Model) currentRound() *site.Round {
	if m.league == nil || len(m.league.Rounds) == 0 {
		return nil
	}
	if m.roundCursor < 0 || m.roundCursor >= len(m.league.Rounds) {
		return nil
	}
	return &m.league.Rounds[m.roundCursor]
}

func (m Model) currentFixture() *site.Fixture {
	round := m.currentRound()
	if round == nil || len(round.Fixtures) == 0 {
		return nil
	}
	if m.fixtureCursor < 0 || m.fixtureCursor >= len(round.Fixtures) {
		return nil
	}
	return &round.Fixtures[m.fixtureCursor]
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
		return archiveLoadedMsg{seasons: seasons, selectedIdx: selectedIdx, competitions: competitions, err: err}
	}
}

func (m Model) loadSeasonCompetitionsCmd(url string, seasonIdx int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, _, competitions, err := m.service.LoadArchive(ctx, url)
		return competitionsLoadedMsg{seasonIdx: seasonIdx, competitions: competitions, err: err}
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

func (m Model) loadMatchCmd(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		match, err := m.service.LoadMatch(ctx, url)
		return matchLoadedMsg{matchURL: url, match: match, err: err}
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

func renderPlayerLine(player site.PlayerLine) string {
	line := player.Name
	if len(player.Events) == 0 {
		return line
	}

	line += " [" + strings.Join(player.Events, ", ") + "]"
	return line
}

func sortedEvents(events []site.MatchEvent) []site.MatchEvent {
	ordered := make([]site.MatchEvent, len(events))
	copy(ordered, events)

	sort.SliceStable(ordered, func(i, j int) bool {
		mi, hi := minuteSortKey(ordered[i].MinuteText)
		mj, hj := minuteSortKey(ordered[j].MinuteText)

		if hi != hj {
			return hi
		}
		if hi && mj != mi {
			return mi < mj
		}

		weightI := eventWeight(ordered[i].Kind)
		weightJ := eventWeight(ordered[j].Kind)
		if weightI != weightJ {
			return weightI < weightJ
		}

		return false
	})

	return ordered
}

func eventWeight(kind string) int {
	switch kind {
	case "GOAL":
		return 0
	case "RC":
		return 1
	case "YC":
		return 2
	case "SUB":
		return 3
	default:
		return 9
	}
}

func minuteSortKey(text string) (int, bool) {
	if text == "" {
		return 0, false
	}

	parts := strings.SplitN(text, "+", 2)
	base := atoiOrNeg(parts[0])
	if base < 0 {
		return 0, false
	}

	extra := 0
	if len(parts) == 2 {
		extra = max(0, atoiOrNeg(parts[1]))
	}

	return base*100 + extra, true
}

func atoiOrNeg(s string) int {
	value := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &value); err != nil {
		return -1
	}
	return value
}

func formatEventLabel(event site.MatchEvent) string {
	prefix := event.Kind
	if prefix == "" {
		prefix = "EVENT"
	}

	if event.Text == "" {
		return prefix
	}

	return prefix + " " + event.Text
}

func renderSideBySide(left, middle, right string, width int) string {
	if width < 30 {
		if middle == "" {
			return left + " | " + right
		}
		return left + " | " + middle + " | " + right
	}

	midWidth := 8
	gap := 2
	sideWidth := max(8, (width-midWidth-(gap*2))/2)

	leftText := padRight(truncate(left, sideWidth), sideWidth)
	midText := padRight(truncate(middle, midWidth), midWidth)
	rightText := truncate(right, sideWidth)

	return leftText + strings.Repeat(" ", gap) + midText + strings.Repeat(" ", gap) + rightText
}

func truncate(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	r := []rune(value)
	if len(r) <= maxLen {
		return value
	}
	if maxLen == 1 {
		return "…"
	}

	return string(r[:maxLen-1]) + "…"
}

func padRight(value string, width int) string {
	pad := width - len([]rune(value))
	if pad <= 0 {
		return value
	}

	return value + strings.Repeat(" ", pad)
}

func layoutWidths(total int, collapsed, emphasizeRight bool) (int, int) {
	if total < 40 {
		return 0, total
	}

	if collapsed {
		return 0, total
	}

	leftWidth := 36
	if emphasizeRight {
		leftWidth = clamp(total/4, 28, 42)
	} else {
		leftWidth = clamp(total/3, 32, 50)
	}

	rightWidth := total - leftWidth - 1
	if rightWidth < 40 {
		rightWidth = 40
		leftWidth = max(0, total-rightWidth-1)
	}

	if leftWidth < 24 {
		leftWidth = 0
		rightWidth = total
	}

	return leftWidth, rightWidth
}
