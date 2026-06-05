package ui

import "github.com/charmbracelet/lipgloss"

func (m *Model) View() string {
	listContent := m.list.View();
	
	box := boxStyle.
		Width(m.boxWidth).
		Render(listContent)

	return lipgloss.JoinVertical(lipgloss.Left, box, m.statusLine());
}

func (m *Model) statusLine() string {
	return renderStatusLine(m.cwd, m.status, m.statusIsError, m.jumpMulti)
}
