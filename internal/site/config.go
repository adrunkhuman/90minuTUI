package site

import "time"

const (
	BaseURL    = "http://www.90minut.pl"
	ArchiveURL = BaseURL + "/archsezon.php"

	// HTTPTimeout caps individual HTTP round-trips; kept short to surface
	// connectivity failures quickly without blocking the TUI.
	HTTPTimeout = 15 * time.Second
)
