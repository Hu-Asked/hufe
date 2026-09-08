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
		model, cmd := m.handleKey(msg)
		m.refreshPreview()
		return model, cmd
	case tea.WindowSizeMsg:
		availableWidth := max(0, msg.Width-5)
		m.boxWidth = availableWidth / 2
		m.previewWidth = availableWidth - m.boxWidth
		m.previewHeight = max(0, msg.Height-3)
		m.list.SetSize(m.boxWidth, m.previewHeight)
		m.list.Styles.TitleBar = headerBarStyle.Width(m.boxWidth)
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
	m.refreshPreview()
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchMode {
		switch msg.String() {
		case "esc":
			m.cancelSearch()
			m.loadDir(m.cwd)
			return m, nil
		case "enter":
			cmd := m.handleSelect()
			return m, cmd
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		default:
			cmd := m.handleSearchInput(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case "/":
		m.initSearch(false)
		return m, nil
	case "?":
		m.initSearch(true)
		return m, nil
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if msg.String() == "0" && m.jumpMulti == 0 {
			m.setItem(0)
			return m, nil
		}
		m.jumpMulti = m.jumpMulti*10 + int(msg.String()[0]-'0')
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h":
		m.jumpMulti = 0
		m.loadPrev()
		return m, nil
	case "l":
		m.jumpMulti = 0
		return m, m.handleSelect()
	case "enter":
		return m, m.handleEnter()
	case "esc":
		m.jumpMulti = 0
		return m, nil
	case "k":
		steps := 1
		if m.jumpMulti > 0 {
			steps = m.jumpMulti
			m.jumpMulti = 0
		}
		target := m.list.Index() - steps
		target = max(0, target)
		m.setItem(target)
		return m, nil
	case "j":
		steps := 1
		if m.jumpMulti > 0 {
			steps = m.jumpMulti
			m.jumpMulti = 0
		}
		target := m.list.Index() + steps
		target = min(len(m.list.Items())-1, target)
		m.setItem(target)
		return m, nil
	case "y":
		m.handleCopy()
		return m, nil
	case "p":
		m.handlePaste()
		return m, nil
	default:
		m.jumpMulti = 0
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
