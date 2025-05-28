package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type settingsState uint

const (
	settingsCount int           = 5
	stageChanges  string        = "\nenter: stage\n"
	repoView      settingsState = iota
	reloadingView
	savingView
	branchView
	versionView
	parentVersionView
	prefixView
	colsView
)

type (
	errMsg error
)

type SettingsModel struct {
	branch        textinput.Model
	version       textinput.Model
	parentVersion textinput.Model
	prefix        textinput.Model
	cols          textinput.Model
	config        *Config
	cursor        Cursor
	state         settingsState
	err           error
	msg           string
}

func NewSettings(config *Config) SettingsModel {
	m := SettingsModel{
		branch:        textinput.New(),
		prefix:        textinput.New(),
		version:       textinput.New(),
		parentVersion: textinput.New(),
		cols:          textinput.New(),
		state:         repoView,
		config:        config,
	}

	m.branch.Placeholder = "master"
	m.branch.CharLimit = 40
	m.branch.Width = 20

	m.prefix.CharLimit = 40
	m.prefix.Width = 20

	m.version.Placeholder = "1.0.0-SNAPSHOT"
	m.version.CharLimit = 40
	m.version.Width = 20

	m.parentVersion.Placeholder = "1.0.0"
	m.parentVersion.CharLimit = 40
	m.parentVersion.Width = 20

	m.cols.Placeholder = "4"
	m.cols.CharLimit = 1
	m.cols.Width = 20

	return m
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func positiveMod(a, b int) int {
	return (a%b + b) % b
}

type MessageAccumulator struct {
	msg string
}

func updateRepo(repo *Repo, m *MessageAccumulator) {
	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()
		start1 := time.Now()
		branch, err := gitBranch(fmt.Sprintf("./%s", repo.Name))
		if err != nil {
			m.msg += fmt.Sprintf("failed to get branch for %s, err=%v\n",
				repo.Name, err)
		} else {
			repo.Branch = strings.TrimSpace(branch)
			m.msg += fmt.Sprintf("t1: %s: elapsed: %dms\n", repo.Name,
				time.Since(start1).Milliseconds())
		}
	}()

	go func() {
		defer wg.Done()
		start2 := time.Now()
		status, err := gitStatus(fmt.Sprintf("./%s", repo.Name))
		if err != nil {
			m.msg += fmt.Sprintf("failed to get status for %s, err=%v\n",
				repo.Name, err)
		} else {
			repo.Modified = !(status == "")
			m.msg += fmt.Sprintf("t2: %s: elapsed: %dms\n", repo.Name,
				time.Since(start2).Milliseconds())
		}
	}()

	go func() {
		defer wg.Done()
		start3 := time.Now()
		err := mvnVersion(fmt.Sprintf("./%s", repo.Name), repo)
		if err != nil {
			m.msg += fmt.Sprintf("failed to get mvn version for %s, err=%v\n",
				repo.Name, err)
		} else {
			m.msg += fmt.Sprintf("t3: %s: elapsed: %dms\n", repo.Name,
				time.Since(start3).Milliseconds())
		}
	}()

	wg.Wait()
}

func saveRepo(repo *Repo, config *Config, ma *MessageAccumulator) {
	var (
		repoPath string = fmt.Sprintf("./%s", repo.Name)
		switched bool
		err      error
	)

	if repo.Branch != config.Branch {
		// check if new branch exists
		start1 := time.Now()
		exists, _ := checkGitBranch(repoPath, config.Branch)

		if exists {
			switched, err = switchGitBranch(repoPath, config.Branch)
		} else {
			switched, err = createGitBranch(repoPath, config.Branch)
		}
		if err != nil {
			//todo log
			ma.msg += "failed to switch branch"
		}
		ma.msg += fmt.Sprintf("%s-git: elapsed: %dms\n", repo.Name, time.Since(start1).Milliseconds())
		if switched {
			repo.Branch = strings.TrimSpace(config.Branch)
		}
	}
	if repo.Maven.Version != config.Version {
		start2 := time.Now()
		err = updateMvnVersion(repoPath, config.Version, repo.Maven.Vln, repo)
		if err != nil {
			//todo log
			ma.msg += fmt.Sprintf("failed to update mvn version, err=%v\n", err)
		}
		ma.msg += fmt.Sprintf("%s-ver: elapsed: %dms\n", repo.Name, time.Since(start2).Milliseconds())
	}
	if repo.Maven.ParentVersion != config.ParentVersion {
		start3 := time.Now()
		err = updateMvnParentVersion(repoPath, config.ParentVersion, repo.Maven.Pvln, repo)
		if err != nil {
			//todo log
			ma.msg += fmt.Sprintf("failed to update mvn parentversion, err=%v\n", err)
		}
		ma.msg += fmt.Sprintf("%s-pver: elapsed: %dms\n", repo.Name, time.Since(start3).Milliseconds())
	}

	status, err := gitStatus(repoPath)
	if err != nil {
		fmt.Printf("failed to get status for %s, err=%v\n", repo.Name, err)
	} else {
		repo.Modified = !(status == "")
		// m.msg += fmt.Sprintf("t2: %s: elapsed: %dms\n", m.config.Repos[i].Name,
		// 	time.Since(start2).Milliseconds())
	}

}

func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == repoView {
				return m, tea.Quit
			}
		case "up", "k":
			if m.cursor.column == 0 {
				if m.cursor.row > 0 {
					m.cursor.row--
				}
			} else {
				m.cursor.row = positiveMod(m.cursor.row-1, settingsCount)
			}
		case "down", "j":
			if m.cursor.column == 0 && m.cursor.row < len(m.config.Repos)-1 {
				m.cursor.row++
			} else {
				m.cursor.row = (m.cursor.row + 1) % settingsCount
			}
		case "right", "l":
			m.cursor.column = (m.cursor.column + 1) % 2
			if m.cursor.column == 1 && m.cursor.row > settingsCount {
				m.cursor.row = settingsCount - 1
			} else if m.cursor.column == 0 && m.cursor.row > len(m.config.Repos) {
				m.cursor.row = len(m.config.Repos) - 1
			}
		case "left", "h":
			m.cursor.column = positiveMod(m.cursor.column-1, 2)

			// m.cursor.column = (m.cursor.column + 1) % 2
			if m.cursor.column == 1 && m.cursor.row > settingsCount {
				m.cursor.row = settingsCount - 1
			} else if m.cursor.column == 0 && m.cursor.row > len(m.config.Repos) {
				m.cursor.row = len(m.config.Repos) - 1
			}
		case "r":
			// m.msg = "reloading..."
			// m.state = repoViewReloading
			// return m, longRunningTask(&m)
			if m.state == repoView {
				m.state = reloadingView
				m.msg = "reloading..."
				return m, func() tea.Msg { return msg }
			} else if m.state == reloadingView {
				m.msg = ""
				repoNames, err := findGitRepos(".")
				for _, repoName := range repoNames {
					found := slices.ContainsFunc(m.config.Repos, func(repo Repo) bool {
						return repo.Name == repoName
					})
					if !found {
						m.config.Repos = append(m.config.Repos, Repo{Name: repoName, Selected: true})
					}
				}
				start := time.Now()
				if err != nil {
					m.msg = "failed to get git repos"
				} else {
					// repos := make([]Repo, 0, len(repoNames))
					ma := &MessageAccumulator{}
					// repoChan := make(chan Repo, len(m.config.Repos))
					var wg sync.WaitGroup

					for i := range m.config.Repos {
						wg.Add(1)
						go func() {
							defer wg.Done()
							updateRepo(&m.config.Repos[i], ma)
						}()
					}
					wg.Wait()

					sort.Slice(m.config.Repos, func(i, j int) bool {
						return m.config.Repos[i].Name < m.config.Repos[j].Name
					})
					// log.Println(ma.msg)
					// m.config.Repos = repos
					elapsed := time.Since(start)
					m.msg += fmt.Sprintf("reloaded in %dms", elapsed.Milliseconds())
				}
				m.state = repoView
			}
		case "s":
			if m.state == repoView {
				m.state = savingView
				m.msg = "saving..."
				return m, func() tea.Msg { return msg }
			} else if m.state == savingView {
				var (
					// repoPath string
					wg sync.WaitGroup
					// switched bool
					// err      error
				)
				start := time.Now()
				ma := &MessageAccumulator{}
				for i := range m.config.Repos {
					wg.Add(1)
					go func() {
						defer wg.Done()
						saveRepo(&m.config.Repos[i], m.config, ma)
					}()
				}
				wg.Wait()
				m.state = repoView
				m.msg = ""
				// fmt.Println(m.config.String())
				// fmt.Println(ma.msg)
				fmt.Printf("elapsed: %dms\n", time.Since(start).Milliseconds())
				// return m, nil
				go m.config.save()
				m.config.state = homeView
				m.msg = ""
				return m, tea.ClearScreen
			}
		case "enter", " ":
			switch m.state {
			case repoView:
				if m.cursor.column == 0 {
					m.config.Repos[m.cursor.row].Selected = !m.config.Repos[m.cursor.row].Selected
				} else {
					if m.cursor.row == 0 {
						m.branch.SetValue(m.config.Branch)
						m.state = branchView
						m.branch.Focus()
						return m, nil
					} else if m.cursor.row == 1 {
						m.version.SetValue(m.config.Version)
						m.state = versionView
						m.version.Focus()
						return m, nil
					} else if m.cursor.row == 2 {
						m.parentVersion.SetValue(m.config.ParentVersion)
						m.state = parentVersionView
						m.parentVersion.Focus()
						return m, nil
					} else if m.cursor.row == 3 {
						m.prefix.SetValue(m.config.Prefix)
						m.state = prefixView
						m.prefix.Focus()
						return m, nil
					} else {
						m.cols.SetValue(m.config.Cols)
						m.state = colsView
						m.cols.Focus()
						return m, nil
					}
				}
			case branchView:
				m.config.Branch = m.branch.Value()
				m.branch.Blur()
				m.state = repoView
			case versionView:
				m.config.Version = m.version.Value()
				m.version.Blur()
				m.state = repoView
			case parentVersionView:
				m.config.ParentVersion = m.parentVersion.Value()
				m.parentVersion.Blur()
				m.state = repoView
			case prefixView:
				m.config.Prefix = m.prefix.Value()
				m.prefix.Blur()
				m.state = repoView
			case colsView:
				m.config.Cols = m.cols.Value()
				m.cols.Blur()
				m.state = repoView
			}
		}
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.branch, cmd = m.branch.Update(msg)
	cmds = append(cmds, cmd)
	m.prefix, cmd = m.prefix.Update(msg)
	cmds = append(cmds, cmd)
	m.version, cmd = m.version.Update(msg)
	cmds = append(cmds, cmd)
	m.parentVersion, cmd = m.parentVersion.Update(msg)
	cmds = append(cmds, cmd)
	m.cols, cmd = m.cols.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func getCursor(cursor Cursor, row int, col int) string {
	if cursor.column == col && cursor.row == row {
		return selectedStyle.Render(">")
	}
	return " "
}

func (m SettingsModel) View() string {
	var s string = ""
	switch m.state {
	case branchView:
		s += fmt.Sprintf(
			"Branch name:\n\n%s\n\n",
			m.branch.View(),
		)
		s += helpStyle.Render(stageChanges)
	case prefixView:
		s += fmt.Sprintf(
			"Project name prefix to hide:\n\n%s\n\n",
			m.prefix.View(),
		)
		s += helpStyle.Render(stageChanges)
	case versionView:
		s += fmt.Sprintf(
			"Version:\n\n%s\n\n",
			m.version.View(),
		)
		s += helpStyle.Render(stageChanges)
	case parentVersionView:
		s += fmt.Sprintf(
			"Parent version:\n\n%s\n\n",
			m.parentVersion.View(),
		)
		s += helpStyle.Render(stageChanges)
	case colsView:
		s += fmt.Sprintf(
			"Number of columns to display:\n\n%s\n\n",
			m.cols.View(),
		)
		s += helpStyle.Render(stageChanges)
	default:
		// s += "\033[0J"
		var sub = make([]string, 0, len(m.config.Repos))
		for i, repo := range m.config.Repos {
			cursor := getCursor(m.cursor, i, 0)
			checked := " "
			if ok := repo.Selected; ok {
				checked = "x"
			}

			sub = append(sub, fmt.Sprintf("%s [%s] %s\n", cursor, checked, repo.Name))
		}
		var b string
		b += fmt.Sprintf("\t  %s branch: %s\n", getCursor(m.cursor, 0, 1), m.config.Branch)
		b += fmt.Sprintf("\t  %s ver: %s\n", getCursor(m.cursor, 1, 1), m.config.Version)
		b += fmt.Sprintf("\t  %s parent ver: %s\n", getCursor(m.cursor, 2, 1), m.config.ParentVersion)
		b += fmt.Sprintf("\t  %s hide prefix: %s\n", getCursor(m.cursor, 3, 1), m.config.Prefix)
		b += fmt.Sprintf("\t  %s num of cols: %s\n", getCursor(m.cursor, 4, 1), m.config.Cols)

		// b := fmt.Sprintf("\t  %s branch: %s\n\t  %s hide prefix: %s\n", getCursor(m.cursor, 0, 1), m.config.Branch, getCursor(m.cursor, 1, 1), m.config.Prefix)
		s += lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(sub, ""), b)
		if m.msg != "" {
			s += "\n"
			s += msgStyle.Render(m.msg)
		}
		s += fmt.Sprintf("\n%v", m.cursor)
		s += helpStyle.Render("\nenter: select • s: save • r: reload • q: exit\n")
	}

	return s
}
