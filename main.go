package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionState uint

const (
	defaultTime              = time.Minute
	homeView    sessionState = iota
	settingsView
)

var (
	// Available spinners
	// spinners = []spinner.Spinner{
	// 	spinner.Line,
	// 	spinner.Dot,
	// 	spinner.MiniDot,
	// 	spinner.Jump,
	// 	spinner.Pulse,
	// 	spinner.Points,
	// 	spinner.Globe,
	// 	spinner.Moon,
	// 	spinner.Monkey,
	// }
	shell      string
	grep       string
	sed        string
	modelStyle = lipgloss.NewStyle().
			Width(15).
			Height(5).
			Align(lipgloss.Center, lipgloss.Center).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8"))
	focusedModelStyle = lipgloss.NewStyle().
				Width(15).
				Height(5).
				Align(lipgloss.Center, lipgloss.Center).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("69"))
	msgStyle = lipgloss.NewStyle().
			Width(30).
			Height(1).
			Align(lipgloss.Center, lipgloss.Center).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("78"))
	// spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	mainStyle = lipgloss.NewStyle().Height(20).Width(80).Padding(1, 1)
)

type Cursor struct {
	column int
	row    int
}

func (c Cursor) String() string {
	return fmt.Sprintf("{col:%d,row%d}", c.column, c.row)
}

type Maven struct {
	Version       string `json:"version"`
	ParentVersion string `json:"parentVerison"`
}

type Repo struct {
	Name     string `json:"name"`
	Branch   string `json:"branch"`
	Modified bool   `json:"modified"`
	Selected bool   `json:"selected"`
	Maven    Maven  `json:"maven"`
}

type model struct {
	config   Config
	settings SettingsModel
	home     HomeModel
	index    int
}

type Config struct {
	Repos         []Repo `json:"repos"`
	Branch        string `json:"branch"`
	Version       string `json:"version"`
	ParentVersion string `json:"parentVersion"`
	Prefix        string `json:"prefix"`
	state         sessionState
}

func (c Config) save() error {
	fmt.Println(c)
	err := saveConfig(c)
	if err != nil {
		fmt.Println("failed to save config", err)
	}
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	var updatedModel tea.Model
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.config.state {
		case settingsView:
			// cmds = append(cmds, m.settings.branch.Focus())
			updatedModel, cmd = m.settings.Update(msg)
			m.settings = updatedModel.(SettingsModel)
			m.config.state = m.settings.config.state
			cmds = append(cmds, cmd)
		default:
			updatedModel, cmd = m.home.Update(msg)
			m.home = updatedModel.(HomeModel)
			m.config.state = m.home.config.state
			cmds = append(cmds, cmd)
		}
	}
	// cmds = append(cmds, tea.ClearScreen)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var s string
	if m.config.state == settingsView {
		s += m.settings.View()
	} else {
		s += m.home.View()
	}
	return mainStyle.Render(s)
	// return s
}

func summarizeStatus(status string) string {
	if status == "" {
		return "Clean"
	}
	// Or count lines, etc.
	return fmt.Sprintf("%d changes", len(strings.Split(status, "\n"))-1)
}

func getConfig() (Config, bool) {
	var config Config

	_, err := os.ReadDir(".massgit")
	if err != nil {
		fmt.Println(".massgit does not exist", err)
		return config, false
	}

	bytes, err := os.ReadFile(".massgit/config.json")

	if err != nil {
		fmt.Println("config does not exist", err)
		return config, false
	}

	err = json.Unmarshal(bytes, &config)

	if err != nil {
		fmt.Println("failed to unmarshal config file", err)
		return config, false
	}

	return config, true
}

func createConfig(config Config) error {
	_, err := os.Stat(".massgit")
	if err == nil {
		config.Repos = []Repo{}
		config.Branch = "master"
		return saveConfig(config)
	}

	err = os.Mkdir(".massgit", 0644)

	if err != nil {
		fmt.Println("failed to create massgit cache")
		return err
	}

	config.Repos = []Repo{}
	config.Branch = "master"
	return saveConfig(config)
}

func saveConfig(config Config) error {
	bytes, err := json.Marshal(config)

	if err != nil {
		fmt.Println("failed to marshal config")
		return err
	}

	err = os.WriteFile(".massgit/config.json", bytes, 0644)

	if err != nil {
		fmt.Println("failed to write config")
		return err
	}

	return nil
}

func newModel() model {
	config, ok := getConfig()
	if !ok {
		err := createConfig(config)

		if err != nil {
			fmt.Println("config in bad state, try deleting '.massgit'")
			os.Exit(1)
		}
	}

	m := model{config: config}
	// for i, repo := range config.Repos {
	// 	if repo.Selected {
	// 		m.current = i
	// 		break
	// 	}
	// }
	// m.branch = textinput.New()
	// m.branch.Placeholder = "master"
	// m.branch.CharLimit = 40
	// m.branch.Width = 20
	m.settings = NewSettings(&config)
	m.home = NewHome(&config)

	// m.timer = timer.New(timeout)
	// m.spinner = spinner.New()
	return m
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
