package site

type Season struct {
	Label    string
	URL      string
	SeasonID string
	Current  bool
}

type Competition struct {
	Name      string
	URL       string
	LeagueKey string
}

type CompetitionMenu struct {
	Title string
	URL   string
	Items []Competition
}

type Fixture struct {
	Home     string
	Away     string
	Score    string
	WhenInfo string
	MatchURL string
	MatchID  string
}

// Round fixtures are normalized to ascending parsed date/time when available;
// undated entries keep their relative source order after dated fixtures.
type Round struct {
	Name     string
	Fixtures []Fixture
}

// StandingRow keeps league-table order from the source page.
type StandingRow struct {
	Position int
	Team     string
	Played   int
	Won      int
	Drawn    int
	Lost     int
	Points   int
}

// LeaguePage is the normalized league view returned by the site layer.
type LeaguePage struct {
	Title     string
	URL       string
	LeagueKey string
	// Standings stays empty when the page has no detectable table.
	Standings []StandingRow
	// Rounds are ordered by detected round number when available.
	Rounds []Round
}

type MatchPage struct {
	Title       string
	URL         string
	MatchID     string
	Competition string
	Meta        string
	Weather     string
	HomeTeam    string
	AwayTeam    string
	Score       string
	Events      []MatchEvent
	HomeLineup  []PlayerLine
	AwayLineup  []PlayerLine
	NewsTitle   string
	NewsURL     string
}

type MatchEvent struct {
	MinuteText string
	Kind       string
	TeamSide   string
	// SUB events carry "<outgoing> -> <incoming>" so renderers can place both players.
	Text string
}

type PlayerLine struct {
	Name    string
	Events  []string
	RawText string
}
