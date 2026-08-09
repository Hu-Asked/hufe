package ui

import "github.com/charmbracelet/lipgloss"

func (m *Model) View() string {
	listContent := m.list.View();
	
	box := boxStyle.
		Width(m.boxWidth).
		Render(listContent)

	var bottom string
	if m.searchMode {
		bottom = m.searchInput.View()
	} else {
		bottom = m.statusLine()
	}

	return lipgloss.JoinVertical(lipgloss.Left, box, bottom)
}

func (m *Model) statusLine() string {
	return renderStatusLine(m.cwd, m.status, m.statusIsError, m.jumpMulti)
}
