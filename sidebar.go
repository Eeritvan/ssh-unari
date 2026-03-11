package main

import (
	"fmt"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

var (
	sidebarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Foreground(lipgloss.Color("#04B575")).
			Padding(1, 2).
			Width(20)
	sidebarItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(1, 2).
				Margin(1, 0).
				Width(16).
				Align(lipgloss.Center)
	sidebarSelectedItemStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#000000")).
					Background(lipgloss.Color("#FFFF00")).
					Padding(1, 2).
					Margin(1, 0).
					Width(16).
					Align(lipgloss.Center)
)

func (m model) renderSidebar() string {
	var campusList []string

	for i, campus := range LOCATIONS {
		var style lipgloss.Style
		if i == m.currentView {
			style = sidebarSelectedItemStyle
		} else {
			style = sidebarItemStyle
		}
		sideBarItem := style.Render(campus)

		sideBarItem = zone.Mark(fmt.Sprintf("%s", campus), sideBarItem)

		campusList = append(campusList, sideBarItem)
	}

	sidebarList := lipgloss.JoinVertical(lipgloss.Center, campusList...)

	sidebarStyleWithHeight := sidebarStyle.
		Width(22).
		Height(m.height - 3).
		MaxHeight(m.height - 1)

	return sidebarStyleWithHeight.Render(sidebarList)
}
