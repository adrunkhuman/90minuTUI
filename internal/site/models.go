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

type Fixture struct {
	Home     string
	Away     string
	Score    string
	WhenInfo string
	MatchURL string
	MatchID  string
}

type Round struct {
	Name     string
	Fixtures []Fixture
}

type LeaguePage struct {
	Title     string
	URL       string
	LeagueKey string
	Rounds    []Round
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
	Text       string
}

type PlayerLine struct {
	Name    string
	Events  []string
	RawText string
}
