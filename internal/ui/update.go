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
		m.boxWidth = msg.Width / 2
		listHeight := msg.Height - 2
		listWidth := m.boxWidth - 1
		m.list.SetSize(listWidth, max(0, listHeight - 1))
		m.list.Styles.TitleBar = headerBarStyle.Width(listWidth);
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
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if msg.String() == "0" && m.jumpMulti == 0 {
			m.list.Select(0);
			return m, nil
		}
		m.jumpMulti = m.jumpMulti * 10 + int(msg.String()[0] - '0')
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h":
		m.loadPrev()
		return m, nil
	case "l":
		return m, m.handleSelect()
	case "enter":
		return m, m.handleEnter()
	case "esc":
		m.jumpMulti = 0;
		return m, nil
	case "k":
		steps := 1
		if m.jumpMulti > 0 {
			steps = m.jumpMulti
			m.jumpMulti = 0
		}
		target := m.list.Index() - steps
		target = max(0, target)
		m.list.Select(target)
		return m, nil
	case "j":
		steps := 1
		if m.jumpMulti > 0 {
			steps = m.jumpMulti
			m.jumpMulti = 0
		}
		target := m.list.Index() + steps
		target = min(len(m.list.Items()) - 1, target)
		m.list.Select(target)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
