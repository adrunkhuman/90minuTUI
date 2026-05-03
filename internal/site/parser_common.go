package site

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var spaceRe = regexp.MustCompile(`\s+`)
var minuteAtEndRe = regexp.MustCompile(`(\d{1,3}(?:\+\d{1,2})?)\s*$`)
var minuteAnywhereRe = regexp.MustCompile(`(\d{1,3}(?:\+\d{1,2})?)`)
var scoreLikeTextRe = regexp.MustCompile(`^\d+\s*-\s*\d+$`)
var leaguePathKeyRe = regexp.MustCompile(`(?i)(?:^|/)liga/\d+/(liga\d+)\.html$`)

func normalizeWhitespace(v string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(v, " "))
}

func extractMinute(text string) string {
	matches := minuteAnywhereRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

// ParseMinute splits a minute string like "45+2" into base minute and stoppage addition.
// Returns (0, 0, false) when the string is empty or cannot be parsed as a valid minute.
func ParseMinute(text string) (int, int, bool) {
	return parseMinute(text)
}

func parseMinute(text string) (int, int, bool) {
	if text == "" {
		return 0, 0, false
	}

	parts := strings.SplitN(text, "+", 2)
	base, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || base < 0 {
		return 0, 0, false
	}

	stoppage := 0
	if len(parts) == 2 {
		if s, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && s > 0 {
			stoppage = s
		}
	}

	return base, stoppage, true
}

func isScoreLikeText(text string) bool {
	return scoreLikeTextRe.MatchString(normalizeWhitespace(text))
}

func canonicalURLKey(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}

	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		sort.Strings(query[key])
	}
	parsed.RawQuery = query.Encode()

	host := strings.ToLower(parsed.Host)
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}

	key := host + path
	if parsed.RawQuery != "" {
		key += "?" + parsed.RawQuery
	}

	if key == "" {
		return strings.TrimSpace(raw)
	}

	return key
}

func extractQueryParam(rawURL, name string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(parsed.Query().Get(name))
}

func extractSeasonID(rawURL string) string {
	return extractQueryParam(rawURL, "id_sezon")
}

func extractMatchID(rawURL string) string {
	return extractQueryParam(rawURL, "id_mecz")
}

func extractLeagueKey(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err == nil {
		matches := leaguePathKeyRe.FindStringSubmatch(parsed.EscapedPath())
		if len(matches) >= 2 {
			return strings.ToLower(strings.TrimSpace(matches[1]))
		}
	}

	return canonicalURLKey(rawURL)
}
