package main

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/adrunkhuman/90minuTUI/internal/cli"
	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/adrunkhuman/90minuTUI/internal/ui"
)

func main() {
	svc := site.NewService()
	if code, handled := runCLI(os.Args, os.Stdout, os.Stderr, svc); handled {
		os.Exit(code)
	}

	model := ui.NewModel(svc)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "app error: %v\n", err)
		os.Exit(1)
	}
}

func runCLI(args []string, stdout, stderr io.Writer, svc cli.Service) (int, bool) {
	if len(args) <= 1 {
		return 0, false
	}
	if cli.IsCommand(args[1]) {
		return cli.Run(args[1:], stdout, stderr, svc), true
	}
	fmt.Fprintf(stderr, "unknown command %q\n", args[1])
	return 1, true
}
