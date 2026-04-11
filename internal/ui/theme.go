package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Three-color palette. accent = cursor/focus, dim = chrome, subtle = secondary labels.
	colorAccent = lipgloss.Color("39")  // bright azure
	colorDim    = lipgloss.Color("240") // dark grey
	colorSubtle = lipgloss.Color("243") // mid grey

	// Semantic card colors — only used for YC/RC lineup markers.
	colorYellow = lipgloss.Color("226")
	colorRed    = lipgloss.Color("196")

	styleAccent  = lipgloss.NewStyle().Foreground(colorAccent)
	styleDim     = lipgloss.NewStyle().Foreground(colorDim)
	styleSubtle  = lipgloss.NewStyle().Foreground(colorSubtle)
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleYellow  = lipgloss.NewStyle().Foreground(colorYellow)
	styleRed     = lipgloss.NewStyle().Foreground(colorRed)
)
