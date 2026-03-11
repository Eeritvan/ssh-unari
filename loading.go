package main

import "charm.land/lipgloss/v2"

var (
	loadingStyle = lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center)
)

func (m model) renderLoading() string {
	return loadingStyle.
		Width(m.width).
		Height(m.height).
		Render("Loading")
}
