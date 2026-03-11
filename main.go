package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eeritvan/unari-ssh/pkg/fetch"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"
	"github.com/joho/godotenv"
	zone "github.com/lrstanley/bubblezone/v2"
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

	finland, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		// TODO: better error message
		fmt.Println(err)
	}
	currentDate := time.Now().In(finland)

	m := model{
		term:         pty.Term,
		width:        pty.Window.Width,
		height:       pty.Window.Height,
		currentView:  kumpulaView,
		data:         unicafeData,
		selectedDate: currentDate,
		loading:      true,
	}
	return m, nil
}

type model struct {
	term         string
	profile      string
	width        int
	height       int
	currentView  int
	data         []fetch.Unicafe
	selectedDate time.Time
	loading      bool
	scrollOffset int
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
	case tea.MouseReleaseMsg:
		if msg.Mouse().Button == tea.MouseWheelDown {
			m.scrollOffset += 2
		}
		if msg.Mouse().Button == tea.MouseWheelUp {
			if m.scrollOffset >= 2 {
				m.scrollOffset -= 2
			}
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

func (m model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if m.loading {
		v.SetContent(m.renderLoading())
		return v
	}
	if m.width < 40 || m.height < 10 {
		v.SetContent(m.renderTermTooSmall())
		return v
	}

	v.SetContent(zone.Scan(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			m.renderSidebar(),
			m.renderRestaurant(),
		),
		m.renderFooter(),
	)))

	return v
}
