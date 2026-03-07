package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrunkhuman/90minuTUI/internal/site"
	"github.com/adrunkhuman/90minuTUI/internal/ui"
)

func main() {
	svc := site.NewService()
	model := ui.NewModel(svc)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "app error: %v\n", err)
		os.Exit(1)
	}
}
