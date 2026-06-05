package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(0, msg.Height-1))
		return m, nil
	case openFileResult:
		if msg.err != nil {
			m.setError(msg.err)
		} else {
			m.clearStatus()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h":
		m.loadPrev()
		return m, nil
	case "l":
		return m, m.handleSelect()
	case "enter":
		return m, m.handleEnter()
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
