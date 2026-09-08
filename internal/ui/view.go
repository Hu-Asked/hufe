package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/epilande/go-devicons"
)

func (m *Model) View() string {
	listBox := boxStyle.
		Width(m.boxWidth).
		Render(m.list.View())
	previewBox := boxStyle.
		Width(m.previewWidth).
		Height(m.previewHeight).
		Render(m.previewView())
	panes := lipgloss.JoinHorizontal(lipgloss.Top, listBox, " ", previewBox)

	var bottom string
	if m.searchMode {
		bottom = m.searchInput.View()
	} else {
		bottom = m.statusLine()
	}

	return lipgloss.JoinVertical(lipgloss.Left, panes, bottom)
}

func (m *Model) statusLine() string {
	return renderStatusLine(m.cwd, m.status, m.statusIsError, m.jumpMulti)
}

func (m *Model) previewView() string {
	title := "Preview"
	if m.previewName != "" {
		title += ": " + m.previewName
	}
	header := headerBarStyle.
		Width(m.previewWidth).
		Render(headerTitleStyle.Render(title))

	bodyHeight := max(0, m.previewHeight-1)
	if bodyHeight == 0 {
		return header
	}

	lines := m.previewLines(bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(lines, "\n"))
}

func (m *Model) previewLines(height int) []string {
	message := func(text string, style lipgloss.Style) []string {
		return []string{style.MaxWidth(max(0, m.previewWidth-2)).Render("  " + text)}
	}

	if m.list.SelectedItem() == nil {
		return message("No selection", hintStyle)
	}
	if !m.previewIsDir {
		return message("Select a directory to preview its contents", hintStyle)
	}
	if m.previewErr != nil {
		return message(fmt.Sprintf("Unable to read directory: %v", m.previewErr), statusErrorStyle)
	}
	if len(m.previewEntries) == 0 {
		return message("Empty directory", hintStyle)
	}

	visible := min(len(m.previewEntries), height)
	if len(m.previewEntries) > height && height > 0 {
		visible = max(0, height-1)
	}

	lines := make([]string, 0, height)
	for _, entry := range m.previewEntries[:visible] {
		name := entry.Name
		if entry.IsDir {
			name += string(os.PathSeparator)
		}
		icon := devicons.IconForPath(entry.Path)
		iconText := lipgloss.NewStyle().Foreground(lipgloss.Color(icon.Color)).Render(icon.Icon)
		line := fmt.Sprintf("  %s %s", iconText, name)
		lines = append(lines, lipgloss.NewStyle().
			Foreground(colors.ListItemForeground).
			MaxWidth(m.previewWidth).
			Render(line))
	}

	if visible < len(m.previewEntries) {
		remaining := len(m.previewEntries) - visible
		lines = append(lines, hintStyle.Render(fmt.Sprintf("  … %d more", remaining)))
	}

	return lines
}
