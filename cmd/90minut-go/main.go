package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"90minut_go/internal/site"
	"90minut_go/internal/ui"
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
