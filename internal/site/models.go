package site

type Season struct {
	Label   string
	URL     string
	Current bool
}

type Competition struct {
	Name string
	URL  string
}

type Fixture struct {
	Home     string
	Away     string
	Score    string
	WhenInfo string
	MatchURL string
}

type Round struct {
	Name     string
	Fixtures []Fixture
}

type LeaguePage struct {
	Title  string
	URL    string
	Rounds []Round
}

type MatchPage struct {
	Title string
	URL   string
	Lines []string
}
