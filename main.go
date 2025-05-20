package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionState uint

const (
	defaultTime              = time.Minute
	timerView   sessionState = iota
	spinnerView
	updateSelectedView
)

var (
	// Available spinners
	spinners = []spinner.Spinner{
		spinner.Line,
		spinner.Dot,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		spinner.Globe,
		spinner.Moon,
		spinner.Monkey,
	}
	modelStyle = lipgloss.NewStyle().
			Width(15).
			Height(5).
			Align(lipgloss.Center, lipgloss.Center).
			BorderStyle(lipgloss.HiddenBorder())
	focusedModelStyle = lipgloss.NewStyle().
				Width(15).
				Height(5).
				Align(lipgloss.Center, lipgloss.Center).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("69"))
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type Repo struct {
	Name     string `json:"name"`
	Branch   string `json:"branch"`
	Selected bool   `json:"selected"`
}

type model struct {
	config Config
	cursor int
	// selected map[int]Repo
	current int
	state   sessionState
	timer   timer.Model
	spinner spinner.Model
	index   int
}

type Config struct {
	Repos []Repo `json:"repos"`
}

func (c Config) save() error {
	// fmt.Println(c)
	err := saveConfig(c)
	if err != nil {
		fmt.Println("failed to save config", err)
	}
	return err
}

func (m model) Init() tea.Cmd {
	config, ok := getConfig()
	if !ok {
		err := createConfig(config)

		if err != nil {
			fmt.Println("config in bad state, try deleting '.jagitui'")
			os.Exit(1)
		}
	}
	return tea.Batch(m.timer.Init(), m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// if m.state == timerView {
			// 	m.state = spinnerView
			// } else {
			// 	m.state = timerView
			// }
			for i := range m.config.Repos {
				idx := (m.current + i + 1) % len(m.config.Repos)
				if m.config.Repos[idx].Selected {
					m.current = idx
					break
				}
			}
			// m.current++
			// if m.current >= len(m.config.Repos) {
			// 	m.current = 0
			// }
		case "n":
			if m.state == timerView {
				m.timer = timer.New(defaultTime)
				cmds = append(cmds, m.timer.Init())
			} else {
				m.Next()
				m.resetSpinner()
				cmds = append(cmds, m.spinner.Tick)
			}
		case "u":
			if m.state != updateSelectedView {
				m.state = updateSelectedView
			}
		case "r":
			if m.state == updateSelectedView {
				repoNames, err := findGitRepos(".")
				if err != nil {
					fmt.Println("failed to get git repos", err)
				} else {
					repos := make([]Repo, 0, len(repoNames))
					for _, repo := range repoNames {
						status, err := gitStatus(fmt.Sprintf("./%s", repo))
						if err != nil {
							fmt.Printf("failed to get status for %s, err=%v\n", repo, err)
						} else {
							repos = append(repos, Repo{Name: repo, Branch: status, Selected: true})
						}
					}
					m.config.Repos = repos
				}
			}
		case "up", "k":
			if m.state == updateSelectedView {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.state == updateSelectedView {
				if m.cursor < len(m.config.Repos)-1 {
					m.cursor++
				}
			}
		case "enter", " ":
			if m.state == updateSelectedView {
				m.config.Repos[m.cursor].Selected = !m.config.Repos[m.cursor].Selected
			}
		case "s":
			if m.state == updateSelectedView {
				go m.config.save()
				m.state = timerView
			}
		}
		switch m.state {
		// update whichever model is focused
		case spinnerView:
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		default:
			m.timer, cmd = m.timer.Update(msg)
			cmds = append(cmds, cmd)
		}
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		// case timer.TickMsg:
		// 	m.timer, cmd = m.timer.Update(msg)
		// 	cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var s string
	model := m.currentFocusedModel()
	if m.state == updateSelectedView {
		for i, repo := range m.config.Repos {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			// Is this choice selected?
			checked := " "
			if ok := repo.Selected; ok {
				checked = "x"
			}

			// Render the row
			s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, repo.Name)
		}
		s += helpStyle.Render("\nenter: select • s: save selections • r: reload repos from disk • q: exit\n")
	} else {
		// s += lipgloss.JoinHorizontal(lipgloss.Top, focusedModelStyle.Render(fmt.Sprintf("%4s", m.timer.View())), modelStyle.Render(m.spinner.View()))
		s += fmt.Sprintf("%v\n", m.config.Repos)
		s += fmt.Sprintf("%d\n", m.current)
		sub := make([]string, 0, len(m.config.Repos))

		for i, repo := range m.config.Repos {
			if !repo.Selected {
				continue
			}
			if i == m.current {
				sub = append(sub, focusedModelStyle.Render(fmt.Sprintf("%s\n🔵 %s", repo.Name, repo.Branch)))
			} else {
				sub = append(sub, modelStyle.Render(fmt.Sprintf("%s\n🔵 %s", repo.Name, repo.Branch)))
			}
		}

		s += lipgloss.JoinHorizontal(lipgloss.Top, sub...)

		s += helpStyle.Render(fmt.Sprintf("\ntab: focus next • n: new %s • u: update repos • q: exit\n", model))
	}
	return s
}

func (m model) currentFocusedModel() string {
	if m.state == timerView {
		return "timer"
	}
	return "spinner"
}

func (m *model) Next() {
	if m.index == len(spinners)-1 {
		m.index = 0
	} else {
		m.index++
	}
}

func (m *model) resetSpinner() {
	m.spinner = spinner.New()
	m.spinner.Style = spinnerStyle
	m.spinner.Spinner = spinners[m.index]
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

	_, err := os.ReadDir(".jagitui")
	if err != nil {
		fmt.Println(".jagitui does not exist", err)
		return config, false
	}

	bytes, err := os.ReadFile(".jagitui/config.json")

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
	err := os.Mkdir(".jagitui", 0644)

	if err != nil {
		fmt.Println("failed to create jagitui cache")
		return err
	}

	config.Repos = []Repo{}
	return saveConfig(config)
}

func saveConfig(config Config) error {
	bytes, err := json.Marshal(config)

	if err != nil {
		fmt.Println("failed to marshal config")
		return err
	}

	err = os.WriteFile(".jagitui/config.json", bytes, 0644)

	if err != nil {
		fmt.Println("failed to write config")
		return err
	}

	return nil
}

func findGitRepos(parentDir string) ([]string, error) {
	var repos []string
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			gitPath := filepath.Join(parentDir, entry.Name(), ".git")
			if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
				repos = append(repos, filepath.Join(parentDir, entry.Name()))
			}
		}
	}
	return repos, nil
}

func gitStatus(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func newModel(timeout time.Duration) model {
	config, _ := getConfig()
	m := model{state: timerView, config: config}
	for i, repo := range config.Repos {
		if repo.Selected {
			m.current = i
			break
		}
	}
	m.timer = timer.New(timeout)
	m.spinner = spinner.New()
	return m
}

func main() {
	p := tea.NewProgram(newModel(defaultTime))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
