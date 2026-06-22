package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/adrunkhuman/90minuTUI/internal/site"
)

type mainFakeService struct{}

func (mainFakeService) LoadArchive(context.Context, string) ([]site.Season, int, []site.Competition, error) {
	return []site.Season{{Label: "2025/26", SeasonID: "107"}}, 0, nil, nil
}

func (mainFakeService) LoadCompetition(context.Context, string) (*site.CompetitionMenu, *site.LeaguePage, error) {
	return nil, nil, nil
}

func (mainFakeService) LoadLeague(context.Context, string) (*site.LeaguePage, error) {
	return &site.LeaguePage{}, nil
}

func (mainFakeService) LoadMatch(context.Context, string) (*site.MatchPage, error) {
	return &site.MatchPage{}, nil
}

func TestRunCLIRoutesKnownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, handled := runCLI([]string{"90minutui", "seasons"}, &stdout, &stderr, mainFakeService{})
	if !handled || code != 0 || stderr.Len() != 0 {
		t.Fatalf("expected handled success, handled=%v code=%d stderr=%q", handled, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"seasons"`) {
		t.Fatalf("expected seasons JSON on stdout, got %q", stdout.String())
	}
}

func TestRunCLIRejectsUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, handled := runCLI([]string{"90minutui", "bogus"}, &stdout, &stderr, mainFakeService{})
	if !handled || code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `unknown command "bogus"`) {
		t.Fatalf("expected unknown command error, handled=%v code=%d stdout=%q stderr=%q", handled, code, stdout.String(), stderr.String())
	}
}

func TestRunCLIIgnoresNoArgs(t *testing.T) {
	code, handled := runCLI([]string{"90minutui"}, ioDiscard{}, ioDiscard{}, mainFakeService{})
	if handled || code != 0 {
		t.Fatalf("expected no args to fall through to TUI, handled=%v code=%d", handled, code)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
