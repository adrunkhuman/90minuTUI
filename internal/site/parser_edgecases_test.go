package site

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
)

func TestDecodeAndParseISO88592FromContentType(t *testing.T) {
	html := `<html><head><title>Zażółć gęślą jaźń</title></head><body><div class="main">Piątek</div></body></html>`
	encoded, err := charmap.ISO8859_2.NewEncoder().Bytes([]byte(html))
	if err != nil {
		t.Fatalf("encode fixture html: %v", err)
	}

	doc, err := decodeAndParse(encoded, "text/html; charset=iso-8859-2")
	if err != nil {
		t.Fatalf("decode and parse: %v", err)
	}

	title := normalizeWhitespace(doc.Find("title").First().Text())
	if title != "Zażółć gęślą jaźń" {
		t.Fatalf("unexpected decoded title: %q", title)
	}

	body := normalizeWhitespace(doc.Find("body").Text())
	if !strings.Contains(body, "Piątek") {
		t.Fatalf("expected decoded body to contain diacritics, got %q", body)
	}
	if strings.ContainsRune(body, '�') {
		t.Fatalf("decoded body contains replacement rune: %q", body)
	}
}

func TestParseSeasonsDefaultsToFirstWhenNoOptionSelected(t *testing.T) {
	html := `<html><body><select name="urljump">
		<option value="/archsezon.php?id_sezon=97">2020/21</option>
		<option value="/archsezon.php?id_sezon=98">2021/22</option>
	</select></body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic html: %v", err)
	}

	seasons, selectedIdx := parseSeasons(doc, NewClient())
	if len(seasons) != 2 {
		t.Fatalf("expected 2 seasons, got %d", len(seasons))
	}
	if selectedIdx != 0 {
		t.Fatalf("expected selected index 0, got %d", selectedIdx)
	}
	if !seasons[0].Current || seasons[1].Current {
		t.Fatalf("expected only first season marked current: %#v", seasons)
	}
}

func TestParseMatchPageGoalSideAssignmentAndStoppageMinutes(t *testing.T) {
	html := `
	<html><head><title>Match Test</title></head><body>
	<table class="main" width="620">
	<tr><td colspan="3"><b>I liga</b></td></tr>
	<tr><td colspan="3">1 marca 2026, 18:00</td></tr>
	<tr><td>GKS Tychy</td><td>2-2</td><td>Odra Opole</td></tr>
	<tr><td>(45+1) Jan Kowalski</td><td>-</td><td></td></tr>
	<tr><td></td><td>-</td><td>(90+2) Adam Nowak</td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic html: %v", err)
	}

	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=777")
	if page == nil {
		t.Fatalf("expected parsed match page")
	}

	goals := make([]MatchEvent, 0, 2)
	for _, event := range page.Events {
		if event.Kind == "GOAL" {
			goals = append(goals, event)
		}
	}

	if len(goals) != 2 {
		t.Fatalf("expected 2 goals, got %d", len(goals))
	}
	if goals[0].TeamSide != "home" || goals[0].MinuteText != "45+1" {
		t.Fatalf("unexpected first goal: %#v", goals[0])
	}
	if goals[1].TeamSide != "away" || goals[1].MinuteText != "90+2" {
		t.Fatalf("unexpected second goal: %#v", goals[1])
	}
}

func TestParseMatchPageMissedPenaltyTimelineEvent(t *testing.T) {
	html := `
	<html><head><title>Match Test</title></head><body>
	<table class="main" width="480">
	<tr><td colspan="3"><b>Ekstraklasa</b></td></tr>
	<tr><td colspan="3">20 marca 2026, 18:00</td></tr>
	<tr><td>Piast Gliwice</td><td>3 - 1</td><td>Radomiak Radom</td></tr>
	<tr><td align="right">&nbsp;<img src="http://img.90minut.pl/img/missed.gif" width="10" height="10" align="absmiddle" alt="(nk)"> Gierman Barkowskij 52 (nk)&nbsp;&nbsp;&nbsp;&nbsp;</td><td></td><td></td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic html: %v", err)
	}

	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=2022961")
	if page == nil {
		t.Fatalf("expected parsed match page")
	}
	if len(page.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(page.Events))
	}

	event := page.Events[0]
	if event.Kind != "MISS" {
		t.Fatalf("unexpected event kind: %#v", event)
	}
	if event.TeamSide != "home" {
		t.Fatalf("unexpected event side: %#v", event)
	}
	if event.MinuteText != "52" {
		t.Fatalf("unexpected missed penalty minute: %#v", event)
	}
	if event.Text != "Gierman Barkowskij 52 (nk)" {
		t.Fatalf("unexpected missed penalty text: %#v", event)
	}
}

func TestParseMinuteSplitsBaseAndStoppage(t *testing.T) {
	cases := []struct {
		input    string
		minute   int
		stoppage int
		ok       bool
	}{
		{"45+2", 45, 2, true},
		{"90+5", 90, 5, true},
		{"67", 67, 0, true},
		{"1", 1, 0, true},
		{"", 0, 0, false},
		{"abc", 0, 0, false},
		{"45+abc", 45, 0, true}, // invalid stoppage treated as 0
	}

	for _, c := range cases {
		m, s, ok := ParseMinute(c.input)
		if ok != c.ok || m != c.minute || s != c.stoppage {
			t.Errorf("ParseMinute(%q) = (%d, %d, %v), want (%d, %d, %v)",
				c.input, m, s, ok, c.minute, c.stoppage, c.ok)
		}
	}
}

func TestParseMatchPageEventsCarryStructuredMinutes(t *testing.T) {
	html := `
	<html><head><title>Match Test</title></head><body>
	<table class="main" width="480">
	<tr><td colspan="3"><b>Ekstraklasa</b></td></tr>
	<tr><td colspan="3">20 marca 2026, 18:00</td></tr>
	<tr><td>Home FC</td><td>1 - 0</td><td>Away FC</td></tr>
	<tr><td align="right">Kowalski 45+1&nbsp;&nbsp;&nbsp;&nbsp;</td><td>1 - 0</td><td></td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}

	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=9999")
	if page == nil {
		t.Fatalf("expected parsed match page")
	}

	var goalEvent *MatchEvent
	for i := range page.Events {
		if page.Events[i].Kind == "GOAL" {
			goalEvent = &page.Events[i]
			break
		}
	}
	if goalEvent == nil {
		t.Fatalf("expected GOAL event, got %#v", page.Events)
	}
	if !goalEvent.HasMinute {
		t.Fatalf("expected HasMinute=true for event with MinuteText %q", goalEvent.MinuteText)
	}
	if goalEvent.Minute != 45 || goalEvent.Stoppage != 1 {
		t.Fatalf("expected Minute=45 Stoppage=1, got Minute=%d Stoppage=%d", goalEvent.Minute, goalEvent.Stoppage)
	}
}

func TestParseMatchPageSubstitutionCellAssignsCardsToBothPlayers(t *testing.T) {
	html := `
	<html><head><title>Match Test</title></head><body>
	<table class="main" width="480">
	<tr><td colspan="3"><b>Ekstraklasa</b></td></tr>
	<tr><td colspan="3">20 marca 2026, 18:00</td></tr>
	<tr><td>Arka Gdynia</td><td>3 - 1</td><td>Zaglebie Lubin</td></tr>
	<tr height="20" valign="middle" align="center" bgcolor="#F5F5F5">
	<td width="45%"><a href="/wystepy.php?id=1" class="main">Oskar Jakubczyk</a>&nbsp;<img src="http://img.90minut.pl/img/yel.gif" width="15" height="15" align="absmiddle" alt="ZK"><br>
	<img src="http://img.90minut.pl/img/sub.gif" width="15" height="15" align="absmiddle">&nbsp;
	72 <a href="/wystepy.php?id=2" class="main">Michal Rzuchowski</a>&nbsp;<img src="http://img.90minut.pl/img/yel.gif" width="15" height="15" align="absmiddle" alt="ZK"></td>
	<td bgcolor="#FFFFFF"></td><td width="45%"></td></tr>
	</table>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse synthetic html: %v", err)
	}

	page := parseMatchPage(doc, "http://www.90minut.pl/mecz.php?id_mecz=2023999")
	if page == nil {
		t.Fatalf("expected parsed match page")
	}
	if len(page.HomeLineup) != 1 {
		t.Fatalf("expected one starter row, got %#v", page.HomeLineup)
	}
	if page.HomeLineup[0].Name != "Oskar Jakubczyk" {
		t.Fatalf("unexpected starter: %#v", page.HomeLineup[0])
	}

	seenSub := false
	seenStarterCard := false
	seenEntrantCard := false
	for _, event := range page.Events {
		switch {
		case event.Kind == "SUB" && event.TeamSide == "home" && event.Text == "Oskar Jakubczyk -> Michal Rzuchowski":
			seenSub = true
		case event.Kind == "YC" && event.TeamSide == "home" && event.Text == "Oskar Jakubczyk":
			seenStarterCard = true
		case event.Kind == "YC" && event.TeamSide == "home" && event.Text == "Michal Rzuchowski":
			seenEntrantCard = true
		}
	}

	if !seenSub || !seenStarterCard || !seenEntrantCard {
		t.Fatalf("expected sub and both YC events, got %#v", page.Events)
	}
}
