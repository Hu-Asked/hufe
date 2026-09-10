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

	bodyHeight := max(0, m.previewHeight-lipgloss.Height(header))
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
	if m.previewErr != nil {
		kind := "file"
		if m.previewIsDir {
			kind = "directory"
		}
		return message(fmt.Sprintf("Unable to read %s: %v", kind, m.previewErr), statusErrorStyle)
	}
	if !m.previewIsDir {
		if m.previewUnsupported {
			return message("Binary or non-text file", hintStyle)
		}
		if len(m.previewFileLines) == 0 {
			return message("Empty file", hintStyle)
		}
		return m.previewTextLines(height)
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

func (m *Model) previewTextLines(height int) []string {
	showFooter := m.previewTruncated || len(m.previewFileLines) > height
	visible := min(len(m.previewFileLines), height)
	if showFooter && height > 0 {
		visible = min(visible, height-1)
	}

	lineStyle := lipgloss.NewStyle().
		Foreground(colors.ListItemForeground).
		MaxWidth(max(0, m.previewWidth))
	lines := make([]string, 0, height)
	for _, line := range m.previewFileLines[:visible] {
		line = strings.ReplaceAll(line, "\t", "    ")
		lines = append(lines, lineStyle.Render("  "+line))
	}

	if showFooter {
		message := fmt.Sprintf("  … %d more lines", len(m.previewFileLines)-visible)
		if m.previewTruncated {
			message = fmt.Sprintf("  … preview limited to %d KiB", maxPreviewBytes/1024)
		}
		lines = append(lines, hintStyle.MaxWidth(max(0, m.previewWidth)).Render(message))
	}

	return lines
}
