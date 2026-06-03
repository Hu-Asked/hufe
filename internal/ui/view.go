package ui

func (m *Model) View() string {
	return m.list.View() + "\n" + m.statusLine()
}

func (m *Model) statusLine() string {
	return renderStatusLine(m.cwd, m.status, m.statusIsError)
}
