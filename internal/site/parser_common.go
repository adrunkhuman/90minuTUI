package site

import (
	"regexp"
	"strings"
)

var spaceRe = regexp.MustCompile(`\s+`)
var minuteAtEndRe = regexp.MustCompile(`(\d{1,3})\s*$`)
var minuteAnywhereRe = regexp.MustCompile(`(\d{1,3}(?:\+\d{1,2})?)`)
var scoreLikeTextRe = regexp.MustCompile(`^\d+\s*-\s*\d+$`)

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
