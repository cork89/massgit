package main

import (
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HomeModel struct {
	config  *Config
	current int
	err     error
}

func NewHome(config *Config) HomeModel {
	return HomeModel{
		config: config,
	}
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			idx := m.current - m.config.Cols
			if !(len(m.config.VisibleRepos)%2 == 0) {
				idx--
			}
			m.current = m.config.VisibleRepos[positiveMod(idx, len(m.config.VisibleRepos))]

		case "down", "j":
			m.current = m.config.VisibleRepos[positiveMod(m.current+m.config.Cols, len(m.config.VisibleRepos))]
		case "tab", "right", "l":
			for i := range m.config.Repos {
				idx := positiveMod(m.current+1+i, len(m.config.Repos))
				if m.config.Repos[idx].Selected {
					m.current = idx
					break
				}
			}
		case "left", "h":
			for i := range m.config.Repos {
				idx := positiveMod(m.current-1-i, len(m.config.Repos))
				if m.config.Repos[idx].Selected {
					m.current = idx
					break
				}
			}
		case "s":
			m.config.state = settingsView
			return m, tea.ClearScreen
		case "c":
			var wg sync.WaitGroup
			for i := range m.config.Repos {
				wg.Add(1)
				var repoPath = fmt.Sprintf("./%s", m.config.Repos[i].Name)
				go func() {
					defer wg.Done()
					_, err := gitAdd(repoPath, "pom.xml")

					if err != nil {
						fmt.Println("failed to add pom.xml")
					}

					_, err = gitCommit(repoPath, "update pom version")

					if err != nil {
						fmt.Println("failed to commit pom.xml")
					}

					status, err := gitStatus(repoPath)
					if err != nil {
						fmt.Printf("failed to get status for %s, err=%v\n", m.config.Repos[i].Name, err)
					} else {
						m.config.Repos[i].Modified = !(status == "")
					}
				}()
			}
			wg.Wait()
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, cmd
}

func (m HomeModel) View() string {
	var s string = ""
	// if m.config.clearPending {
	// 	s += "\033[0J"
	// 	m.config.clearPending = false
	// }
	// s += fmt.Sprintf("%v - %s\n", m.config.Repos, m.config.Branch)
	// s += fmt.Sprintf("ver: %s\n", m.config.Version)
	// s += fmt.Sprintf("pver: %s\n", m.config.ParentVersion)
	s += fmt.Sprintf("%d\n", m.config.VisibleRepos)
	s += fmt.Sprintf("%d\n", m.current)
	sub := make([]string, 0, len(m.config.Repos))

	for i, repo := range m.config.Repos {
		if !repo.Selected {
			continue
		}
		trimmedRepo := strings.TrimPrefix(repo.Name, m.config.Prefix)
		if i == m.current {
			sub = append(sub, focusedModelStyle.Render(fmt.Sprintf("%s\n%s %s\nv: %s\npv: %s", trimmedRepo, getModifiedColor(repo.Modified), repo.Branch, repo.Maven.Version, repo.Maven.ParentVersion)))
		} else {
			sub = append(sub, modelStyle.Render(fmt.Sprintf("%s\n%s %s\nv: %s\npv: %s", trimmedRepo, getModifiedColor(repo.Modified), repo.Branch, repo.Maven.Version, repo.Maven.ParentVersion)))
		}
	}
	numCols := m.config.Cols

	if numCols == 0 {
		numCols = 4
	}

	sub2 := make([]string, 0, (len(sub)+numCols-1)/numCols)
	for i := range (len(sub) + numCols - 1) / numCols {
		start := i * numCols
		end := i*numCols + numCols
		if end > len(m.config.Repos) {
			end = len(m.config.Repos)
		}
		sub2 = append(sub2, lipgloss.JoinHorizontal(lipgloss.Top, sub[start:end]...))
	}

	s += lipgloss.JoinVertical(lipgloss.Top, sub2...)

	s += helpStyle.Render("\nhjkl mvmt • s: settings • c: commit changes • q: exit\n")

	// style := lipgloss.NewStyle().Height(20).Width(80).Padding(1, 2)
	return s
}

func getModifiedColor(modified bool) string {
	if modified {
		return "🟡"
	} else {
		return "🟢"
	}
}
