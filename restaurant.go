package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	txtStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))
	contentStyle = lipgloss.NewStyle().
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder())
	restaurantHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("12"))
	mealText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffcd73"))
	veganMealText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ebffa3"))
	dessertText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#eaa3ff"))
)

func (m model) renderRestaurant() string {
	campus := LOCATIONS[m.currentView]
	campusRestaurants := LOCATION_RESTAURANTS[campus]

	var restaurantList strings.Builder
	restaurantList.WriteString(txtStyle.Bold(true).Underline(true).Render(m.selectedDate.Format("Monday 02.01.2006")))
	restaurantList.WriteString("\n")

	found := false
	for _, restaurant := range m.data {
		if slices.Contains(campusRestaurants, restaurant.Title) {
			for _, menu := range restaurant.Menu.Menus {
				restaurantDate := strings.Split(menu.Date, " ")
				currentDate := m.selectedDate.Format("02.01.")
				if restaurantDate[len(restaurantDate)-1] == currentDate {
					found = true
					var menuItems []string

					rank := func(name string) int {
						n := strings.ToLower(strings.TrimSpace(name))
						switch n {
						case "lounas":
							return 0
						case "vegaanilounas":
							return 1
						case "lisäke":
							return 2
						case "jälkiruoka":
							return 3
						default:
							return 100
						}
					}

					sort.SliceStable(menu.Data, func(i, j int) bool {
						ri := rank(menu.Data[i].Price.Name)
						rj := rank(menu.Data[j].Price.Name)
						if ri != rj {
							return ri < rj
						}
						return strings.ToLower(strings.TrimSpace(menu.Data[i].Name)) < strings.ToLower(strings.TrimSpace(menu.Data[j].Name))
					})

					for _, meal := range menu.Data {
						var mealType string
						switch meal.Price.Name {
						case "Lounas":
							mealType = mealText.Render(meal.Price.Name + " ")
						case "Lisäke":
							mealType = mealText.Render(meal.Price.Name + " ")
						case "Buffet":
							mealType = mealText.Render("Lounas" + " ")
						case "Vegaanilounas":
							mealType = veganMealText.Render("Veg ")
						case "Jälkiruoka":
							mealType = dessertText.Render(meal.Price.Name + " ")
						}
						menuItems = append(menuItems, " • "+mealType+strings.Trim(meal.Name, " "))
					}

					fmt.Fprintf(&restaurantList, "\n\n%s\n%s",
						restaurantHeader.Render(restaurant.Title),
						strings.Join(menuItems, "\n"))
				}
			}
		}
	}

	if !found {
		restaurantList.WriteString("\n\nNo data for this date.")
	}

	content := restaurantList.String()
	lines := strings.Split(content, "\n")

	// ai slop. didnt bother to check but seems to work ok
	visibleHeight := m.height - 3 - 2

	maxOffset := max(len(lines)-visibleHeight, 0)
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	if m.scrollOffset > 0 && m.scrollOffset < len(lines) {
		lines = lines[m.scrollOffset:]
	}

	scrolledContent := strings.Join(lines, "\n")

	return contentStyle.
		Width(m.width - 26).
		Height(m.height - 3).
		MaxHeight(m.height - 1).
		Render(scrolledContent)
}
