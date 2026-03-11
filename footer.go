package main

import "charm.land/lipgloss/v2"

var (
	footerStyle = lipgloss.NewStyle().
		Bold(true).
		Italic(true).
		TabWidth(4).
		Foreground(lipgloss.Color("#3C3C3C")).
		Align(lipgloss.Right)
)

// TODO: a: about?
func (m model) renderFooter() string {
	left := footerStyle.Render("q: quit")
	right := footerStyle.Render("↑/↓: campus\tt: today\t←/→: date")

	leftView := footerStyle.Render(left)

	infoWidth := m.width - lipgloss.Width(leftView)

	rightView := footerStyle.
		Width(infoWidth).
		Render(right)

	return lipgloss.JoinHorizontal(lipgloss.Bottom, leftView, rightView)
}
