package site

type Season struct {
	Label    string
	URL      string
	SeasonID string
	Current  bool
}

type Competition struct {
	Name string
	// Archive grouping can emit synthetic submenu URLs like
	// <archive-url>#women-tier=iii-liga-kobiet for drill-down-only entries.
	URL string
	// LeagueKey is a stable request identity and may be synthetic for submenu nodes.
	LeagueKey string
}

type CompetitionMenu struct {
	Title string
	URL   string
	Items []Competition
}

type Fixture struct {
	Home  string
	Away  string
	Score string
	// WhenInfo keeps date/time text and any trailing source metadata that does not belong in the score cell.
	WhenInfo string
	// MatchURL stays empty when 90minut publishes a fixture row without a drillable match page.
	MatchURL string
	// MatchID stays empty when MatchURL is empty.
	MatchID string
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
	// Events combines score-row incidents and lineup-derived cards/substitutions; consumers sort when order matters.
	Events     []MatchEvent
	HomeLineup []PlayerLine
	AwayLineup []PlayerLine
	NewsTitle  string
	NewsURL    string
}

type MatchEvent struct {
	MinuteText string
	// Minute and Stoppage are the parsed components of MinuteText (e.g. "45+2" → 45, 2).
	// HasMinute is false when MinuteText was absent or unparseable.
	Minute    int
	Stoppage  int
	HasMinute bool
	Kind      MatchEventKind
	// TeamSide is event ownership relative to the parsed home/away teams.
	TeamSide TeamSide
	// Text is normalized source text; substitution callers should prefer SubstitutionOut/SubstitutionIn.
	Text string
	// Substitution events keep parsed participants so UI code doesn't reparse display text.
	SubstitutionOut string
	SubstitutionIn  string
}

// MatchEventKind is the parser's normalized event vocabulary.
type MatchEventKind string

const (
	EventKindGoal         MatchEventKind = "GOAL"
	EventKindMiss         MatchEventKind = "MISS"
	EventKindYellowCard   MatchEventKind = "YC"
	EventKindRedCard      MatchEventKind = "RC"
	EventKindSubstitution MatchEventKind = "SUB"
	// EventKindGeneric preserves lineup markers without dedicated UI handling.
	EventKindGeneric MatchEventKind = "EVENT"
)

// TeamSide uses the parser's closed home/away vocabulary for match-owned events.
type TeamSide string

const (
	TeamSideHome TeamSide = "home"
	TeamSideAway TeamSide = "away"
)

type PlayerLine struct {
	Name string
	// Events stores raw lineup-cell markers until parser normalization emits MatchEvent values.
	Events  []string
	RawText string
}
