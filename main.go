package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"slices"
	"sort"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/eeritvan/unari-ssh/pkg/fetch"
	"github.com/joho/godotenv"
	zone "github.com/lrstanley/bubblezone"
)

type unicafeDataMsg []fetch.Unicafe

var LOCATIONS = [...]string{"Keskusta", "Kumpula", "Meilahti", "Töölö", "Viikki"}
var unicafeData []fetch.Unicafe

var LOCATION_RESTAURANTS = map[string][]string{
	"Keskusta": {"Myöhä Café & Bar", "Kaivopiha", "Kaisa-talo", "Soc&Kom", "Rotunda", "Porthania Opettajien ravintola", "Porthania", "Topelias", "Olivia", "Metsätalo"},
	"Kumpula":  {"Physicum", "Exactum", "Chemicum", "Chemicum Opettajien ravintola"},
	"Meilahti": {"Terkko", "Meilahti"},
	"Töölö":    {"Serpens"},
	"Viikki":   {"Tähkä", "Biokeskus 2", "Infokeskus alakerta", "Viikuna", "Infokeskus", "Biokeskus"},
}

const (
	keskustaView int = iota
	kumpulaView
	meilahtiView
	töölöView
	viikkiView
	totalViews
)

func main() {
	err := godotenv.Load()
	if err != nil {
		// TODO: error handling
		//    - must not break dockerfile
		fmt.Println("Error loading .env file")
	}

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	zone.NewGlobal()

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)
	txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))
	sidebarStyle := renderer.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Foreground(lipgloss.Color("#04B575")).
		Padding(1, 2).
		Width(20)
	sidebarItemStyle := renderer.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Margin(1, 0).
		Width(16).
		Align(lipgloss.Center)
	sidebarSelectedItemStyle := renderer.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#FFFF00")).
		Padding(1, 2).
		Margin(1, 0).
		Width(16).
		Align(lipgloss.Center)
	footerStyle := renderer.NewStyle().
		Bold(true).
		Italic(true).
		TabWidth(4).
		Foreground(lipgloss.Color("#3C3C3C")).
		Align(lipgloss.Right)
	contentStyle := renderer.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder())
	restaurantHeader := renderer.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))
	mealText := renderer.NewStyle().
		Foreground(lipgloss.Color("#ffcd73"))
	veganMealText := renderer.NewStyle().
		Foreground(lipgloss.Color("#ebffa3"))
	dessertText := renderer.NewStyle().
		Foreground(lipgloss.Color("#eaa3ff"))
	termTooSmallStyle := renderer.NewStyle().
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center)
	loadingStyle := renderer.NewStyle().
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center)
	bg := "light"
	if renderer.HasDarkBackground() {
		bg = "dark"
	}

	finland, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		// TODO: better error message
		fmt.Println(err)
	}
	currentDate := time.Now().In(finland)

	m := model{
		term:                     pty.Term,
		profile:                  renderer.ColorProfile().Name(),
		width:                    pty.Window.Width,
		height:                   pty.Window.Height,
		bg:                       bg,
		txtStyle:                 txtStyle,
		sidebarStyle:             sidebarStyle,
		sidebarItemStyle:         sidebarItemStyle,
		sidebarSelectedItemStyle: sidebarSelectedItemStyle,
		footerStyle:              footerStyle,
		contentStyle:             contentStyle,
		restaurantHeader:         restaurantHeader,
		mealText:                 mealText,
		veganMealText:            veganMealText,
		dessertText:              dessertText,
		termTooSmallStyle:        termTooSmallStyle,
		loadingStyle:             loadingStyle,
		currentView:              kumpulaView,
		data:                     unicafeData,
		selectedDate:             currentDate,
		loading:                  true,
	}
	return m, []tea.ProgramOption{tea.WithMouseCellMotion(), tea.WithAltScreen()}
}

type model struct {
	term                     string
	profile                  string
	width                    int
	height                   int
	bg                       string
	currentView              int
	txtStyle                 lipgloss.Style
	footerStyle              lipgloss.Style
	sidebarStyle             lipgloss.Style
	sidebarItemStyle         lipgloss.Style
	sidebarSelectedItemStyle lipgloss.Style
	contentStyle             lipgloss.Style
	restaurantHeader         lipgloss.Style
	mealText                 lipgloss.Style
	veganMealText            lipgloss.Style
	dessertText              lipgloss.Style
	termTooSmallStyle        lipgloss.Style
	loadingStyle             lipgloss.Style
	data                     []fetch.Unicafe
	selectedDate             time.Time
	loading                  bool
	scrollOffset             int
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		data, err := fetch.GetUnicafe()
		if err != nil {
			// TODO: error handling
			return unicafeDataMsg([]fetch.Unicafe{})
		}
		return unicafeDataMsg(data)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case unicafeDataMsg:
		m.loading = false
		m.data = msg
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelDown {
			m.scrollOffset += 2
		}
		if msg.Button == tea.MouseButtonWheelUp {
			if m.scrollOffset >= 2 {
				m.scrollOffset -= 2
			}
		}
		if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		for i, campus := range LOCATIONS {
			if m.currentView != i && zone.Get(campus).InBounds(msg) {
				m.currentView = i
				m.scrollOffset = 0
			}
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.currentView--
			if m.currentView < 0 {
				m.currentView = totalViews - 1
			}
			m.scrollOffset = 0
		case "down", "j":
			m.currentView++
			if m.currentView >= totalViews {
				m.currentView = 0
			}
			m.scrollOffset = 0
		case "right", "l": // next day
			// TODO: check that unicafe has this date
			m.selectedDate = m.selectedDate.AddDate(0, 0, 1)
			m.scrollOffset = 0
		case "left", "h": // prev day
			// TODO: check that unicafe has this date
			m.selectedDate = m.selectedDate.AddDate(0, 0, -1)
			m.scrollOffset = 0
		case "t", "T": // current date
			finland, err := time.LoadLocation("Europe/Helsinki")
			if err != nil {
				// TODO: better error message
				fmt.Println(err)
			}
			m.selectedDate = time.Now().In(finland)
			m.scrollOffset = 0
			// case "ctrl+f":
			// 	fmt.Println("find")
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.loading {
		return m.renderLoading()
	}
	if m.width < 40 || m.height < 10 {
		return m.renderTermTooSmall()
	}

	sidebar := m.renderSidebar()
	restaurantView := m.renderRestaurant(m.currentView)
	footer := m.renderFooter()

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, restaurantView)

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Left, mainView, footer))
}

func (m model) renderRestaurant(idx int) string {
	campus := LOCATIONS[idx]
	campusRestaurants := LOCATION_RESTAURANTS[campus]

	var restaurantList strings.Builder
	restaurantList.WriteString(m.txtStyle.Bold(true).Underline(true).Render(m.selectedDate.Format("Monday 02.01.2006")))
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
							mealType = m.mealText.Render(meal.Price.Name + " ")
						case "Lisäke":
							mealType = m.mealText.Render(meal.Price.Name + " ")
						case "Buffet":
							mealType = m.mealText.Render(meal.Price.Name + " ")
						case "Vegaanilounas":
							mealType = m.veganMealText.Render("Veg ")
						case "Jälkiruoka":
							mealType = m.dessertText.Render(meal.Price.Name + " ")
						}
						menuItems = append(menuItems, " • "+mealType+strings.Trim(meal.Name, " "))
					}

					restaurantList.WriteString(fmt.Sprintf("\n\n%s\n%s",
						m.restaurantHeader.Render(restaurant.Title),
						strings.Join(menuItems, "\n")))
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
	// ----

	return m.contentStyle.
		Width(m.width - 26).
		Height(m.height - 3).
		MaxHeight(m.height - 1).
		Render(scrolledContent)
}

// TODO: a: about?
func (m model) renderFooter() string {
	left := m.footerStyle.Render("q: quit")
	right := m.footerStyle.Render("↑/↓: campus\tt: today\t←/→: date")

	leftView := m.footerStyle.Render(left)

	infoWidth := m.width - lipgloss.Width(leftView)

	rightView := m.footerStyle.
		Width(infoWidth).
		Render(right)

	return lipgloss.JoinHorizontal(lipgloss.Bottom, leftView, rightView)
}

func (m model) renderSidebar() string {
	var campusList []string

	for i, campus := range LOCATIONS {
		var style lipgloss.Style
		if i == m.currentView {
			style = m.sidebarSelectedItemStyle
		} else {
			style = m.sidebarItemStyle
		}
		sideBarItem := style.Render(campus)

		sideBarItem = zone.Mark(fmt.Sprintf(campus), sideBarItem)

		campusList = append(campusList, sideBarItem)
	}

	sidebarList := lipgloss.JoinVertical(lipgloss.Center, campusList...)

	sidebarStyleWithHeight := m.sidebarStyle.
		Width(22).
		Height(m.height - 3).
		MaxHeight(m.height - 1)

	return sidebarStyleWithHeight.Render(sidebarList)
}

func (m model) renderTermTooSmall() string {
	return m.termTooSmallStyle.
		Width(m.width).
		Height(m.height).
		Render("Terminal too small")
}

func (m model) renderLoading() string {
	return m.loadingStyle.
		Width(m.width).
		Height(m.height).
		Render("Loading")
}
