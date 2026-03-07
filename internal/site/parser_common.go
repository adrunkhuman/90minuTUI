package site

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var spaceRe = regexp.MustCompile(`\s+`)
var minuteAtEndRe = regexp.MustCompile(`(\d{1,3})\s*$`)
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
