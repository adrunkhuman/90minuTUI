package ui

import "charm.land/lipgloss/v2"

var (
	colorBg            = lipgloss.Color("#0d0f0e")
	colorBgPanel       = lipgloss.Color("#121510")
	colorBgPane        = lipgloss.Color("#0f1210")
	colorBgHeader      = lipgloss.Color("#161b14")
	colorBgSelected    = lipgloss.Color("#1a2118")
	colorBgHover       = lipgloss.Color("#151a13")
	colorBgModal       = lipgloss.Color("#0e1210")
	colorStatusBar     = lipgloss.Color("#0a0d09")
	colorBorder        = lipgloss.Color("#2a3028")
	colorBorderStrong  = lipgloss.Color("#3d4d38")
	colorAccent        = lipgloss.Color("#b8f000")
	colorAccentDim     = lipgloss.Color("#7aaa00")
	colorTextPrimary   = lipgloss.Color("#e8edd4")
	colorTextSecondary = lipgloss.Color("#7a8a72")
	colorTextMuted     = lipgloss.Color("#4a5a44")
	colorWin           = colorAccent
	colorDraw          = lipgloss.Color("#f0b800")
	colorLoss          = lipgloss.Color("#e04040")
	colorRowOdd        = lipgloss.Color("#0f1210")
	colorRowEven       = lipgloss.Color("#111410")

	// Backward-compatible names for helpers that still provide semantic markers.
	colorDim    = colorTextMuted
	colorSubtle = colorTextSecondary
	colorYellow = colorDraw
	colorRed    = colorLoss

	styleAccent    = lipgloss.NewStyle().Foreground(colorAccent)
	styleAccentDim = lipgloss.NewStyle().Foreground(colorAccentDim)
	styleDim       = lipgloss.NewStyle().Foreground(colorTextMuted)
	styleSubtle    = lipgloss.NewStyle().Foreground(colorTextSecondary)
	stylePrimary   = lipgloss.NewStyle().Foreground(colorTextPrimary)
	styleBold      = lipgloss.NewStyle().Bold(true).Foreground(colorTextPrimary)
	styleHeader    = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBgHeader).Bold(true)
	styleYellow    = lipgloss.NewStyle().Foreground(colorYellow).Background(colorBgPanel)
	styleRed       = lipgloss.NewStyle().Foreground(colorRed).Background(colorBgPanel)
)
