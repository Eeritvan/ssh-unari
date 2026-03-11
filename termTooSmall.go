package main

import "charm.land/lipgloss/v2"

var (
	termTooSmallStyle = lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center)
)

func (m model) renderTermTooSmall() string {
	return termTooSmallStyle.
		Width(m.width).
		Height(m.height).
		Render("Terminal too small")
}
