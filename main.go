package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	fetch "github.com/eeritvan/unari-ssh/pkg/fetch"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/joho/godotenv"
)

var data []fetch.Unicafe

type viewType int

const (
	homeView viewType = iota
	restaurantView
	terminalInfoView
	totalViews
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

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
	quitStyle := renderer.NewStyle().Foreground(lipgloss.Color("8"))
	titleStyle := renderer.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		BorderStyle(lipgloss.NormalBorder())
	navStyle := renderer.NewStyle().
		Foreground(lipgloss.Color("12")).
		Italic(true)
	sidebarStyle := renderer.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Align(lipgloss.Left)

	bg := "light"
	if renderer.HasDarkBackground() {
		bg = "dark"
	}

	m := model{
		term:         pty.Term,
		profile:      renderer.ColorProfile().Name(),
		width:        pty.Window.Width,
		height:       pty.Window.Height,
		bg:           bg,
		txtStyle:     txtStyle,
		quitStyle:    quitStyle,
		titleStyle:   titleStyle,
		navStyle:     navStyle,
		sidebarStyle: sidebarStyle,
		currentView:  homeView,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type model struct {
	term         string
	profile      string
	width        int
	height       int
	bg           string
	txtStyle     lipgloss.Style
	quitStyle    lipgloss.Style
	titleStyle   lipgloss.Style
	navStyle     lipgloss.Style
	sidebarStyle lipgloss.Style
	currentView  viewType
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.currentView--
			if m.currentView < 0 {
				m.currentView = totalViews - 1
			}
		case "down", "j":
			m.currentView++
			if m.currentView >= totalViews {
				m.currentView = 0
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var content string

	switch m.currentView {
	case homeView:
		content = m.renderHomeView()
	case restaurantView:
		content = m.renderRestaurantView()
	case terminalInfoView:
		content = m.renderTerminalInfoView()
	}

	nav := m.navStyle.Render(fmt.Sprintf("\n\nView %d/%d",
		int(m.currentView)+1, int(totalViews)))
	quit := m.quitStyle.Render("\nPress 'q' to quit")

	return content + nav + quit
}

func (m model) renderHomeView() string {
	title := m.titleStyle.Render("view 1")
	content := m.txtStyle.Render("yo yo yo.")
	return title + content
}

func (m model) renderRestaurantView() string {
	title := m.titleStyle.Render("view 2")

	if len(data) == 0 {
		var err error
		restaurants, err := fetch.GetUnicafe()
		if err != nil {
			return title + "\n" + m.txtStyle.Render(fmt.Sprintf("\nError loading restaurants: %v", err))
		}
		data = restaurants
	}

	var restaurantList string
	for index, restaurant := range data {
		restaurantList += fmt.Sprintf("\n  %d. %s", index+1, restaurant.Title)
	}

	content := m.txtStyle.Render(restaurantList)
	return title + content
}

func (m model) renderTerminalInfoView() string {
	title := m.titleStyle.Render("view 3")

	info := fmt.Sprintf(`
		Terminal: %s
		Window Size: %dx%d
		Background: %s
		Color Profile: %s`,
		m.term, m.width, m.height, m.bg, m.profile)

	content := m.txtStyle.Render(info)
	return title + content
}
