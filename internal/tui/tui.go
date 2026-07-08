package tui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/ninj4dkill4/octx/internal/config"
	"github.com/ninj4dkill4/octx/internal/switcher"
)

type model struct {
	projects []config.Project
	cursor   int
	opts     switcher.Options
	pickOnly bool
	picked   *Selection
	result   *switcher.Result
	err      error
	quitting bool
}

type Selection struct {
	Project *config.Project
	Clear   bool
}

var (
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
)

func Run(cfg config.Config, opts switcher.Options) (*switcher.Result, error) {
	fm, err := run(cfg, opts, false, nil)
	if err != nil {
		return nil, err
	}
	return fm.result, nil
}

func Pick(cfg config.Config, output io.Writer) (*Selection, error) {
	fm, err := run(cfg, switcher.Options{}, true, output)
	if err != nil {
		return nil, err
	}
	return fm.picked, nil
}

func run(cfg config.Config, opts switcher.Options, pickOnly bool, output io.Writer) (model, error) {
	m := model{
		projects: cfg.Projects,
		opts:     opts,
		pickOnly: pickOnly,
	}

	programOptions := []tea.ProgramOption{}
	if output != nil {
		lipgloss.SetColorProfile(termenv.ANSI256)
		programOptions = append(programOptions, tea.WithOutput(output))
	}

	finalModel, err := tea.NewProgram(m, programOptions...).Run()
	if err != nil {
		return model{}, err
	}

	fm, ok := finalModel.(model)
	if !ok {
		return model{}, nil
	}
	if fm.err != nil {
		return model{}, fm.err
	}
	return fm, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.projects) {
				m.cursor++
			}
		case "enter":
			if m.cursor == 0 {
				if m.pickOnly {
					m.picked = &Selection{Clear: true}
					return m, tea.Quit
				}
				if _, err := switcher.Clear(m.opts); err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.result = &switcher.Result{}
				return m, tea.Quit
			}
			project := m.projects[m.cursor-1]
			if m.pickOnly {
				m.picked = &Selection{Project: &project}
				return m, tea.Quit
			}
			result, err := switcher.Switch(project.Code, m.opts)
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			m.result = &result
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.result != nil || m.picked != nil || m.quitting {
		return ""
	}
	if m.err != nil {
		return m.err.Error()
	}

	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render("Project Context Switcher"))
	fmt.Fprintln(&b, promptStyle.Render("?")+" Choose a profile")

	clearName := "unset profiles"
	clearCursor := " "
	if m.cursor == 0 {
		clearCursor = cursorStyle.Render("›")
		clearName = selectedStyle.Render(clearName)
	}
	fmt.Fprintf(&b, "%s %s\n", clearCursor, clearName)

	for i, project := range m.projects {
		cursor := " "
		name := project.Code
		if i+1 == m.cursor {
			cursor = cursorStyle.Render("›")
			name = selectedStyle.Render(name)
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, name)
	}

	if len(m.projects) == 0 {
		fmt.Fprintln(&b, "  No projects configured")
	}

	return b.String()
}
