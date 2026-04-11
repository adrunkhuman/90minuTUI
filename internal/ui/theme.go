package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Three-color palette. accent = cursor/focus, dim = chrome, subtle = secondary labels.
	colorAccent = lipgloss.Color("39")  // bright azure
	colorDim    = lipgloss.Color("238") // near-black grey
	colorSubtle = lipgloss.Color("243") // mid grey

	styleAccent = lipgloss.NewStyle().Foreground(colorAccent)
	styleDim    = lipgloss.NewStyle().Foreground(colorDim)
	styleSubtle = lipgloss.NewStyle().Foreground(colorSubtle)
	styleBold   = lipgloss.NewStyle().Bold(true)
)
