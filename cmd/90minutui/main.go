package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/adrunkhuman/90minuTUI/internal/ui"
)

func main() {
	svc := site.NewService()
	model := ui.NewModel(svc)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "app error: %v\n", err)
		os.Exit(1)
	}
}
