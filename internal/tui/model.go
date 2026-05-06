package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

type Model struct {
	Input    textinput.Model
	Progress progress.Model
	Percent  float64
	Done     bool
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "S3 URI (s3://bucket/key)"
	ti.Focus()
	p := progress.New(progress.WithDefaultGradient())
	return Model{Input: ti, Progress: p}
}

func (m Model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd { return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) }) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			m.Done = true
			return m, tea.Quit
		}
	case tickMsg:
		if m.Percent < 1.0 {
			m.Percent += 0.03
			if m.Percent > 1.0 {
				m.Percent = 1.0
			}
		}
		return m, tick()
	}
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return fmt.Sprintf("%s\n%s\n\nDownload target: %s\n\nPress Enter to start, q to quit.\n",
		"Progress:", m.Progress.ViewAs(m.Percent), m.Input.View())
}
