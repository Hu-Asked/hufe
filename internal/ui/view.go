package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/epilande/go-devicons"
)

func (m *Model) View() string {
	base := m.baseView()
	if m.paste == nil {
		return base
	}

	width := m.windowWidth
	height := m.windowHeight
	if width <= 0 {
		width = lipgloss.Width(base)
	}
	if height <= 0 {
		height = lipgloss.Height(base)
	}
	return overlayCentered(base, m.pasteView(), width, height)
}

func (m *Model) baseView() string {
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

func (m *Model) pasteView() string {
	availableWidth := max(16, m.windowWidth-8)
	contentWidth := min(56, availableWidth-6)
	if contentWidth < 10 {
		contentWidth = 10
	}

	title := "Pasting " + filepath.Base(m.paste.source)
	phase := "Preparing files…"
	if m.paste.cancelling {
		phase = "Cancelling…"
	} else if m.paste.phase == pastePhaseCopying {
		phase = "Copying files…"
	}

	ratio := pasteRatio(m.paste)
	barWidth := max(8, contentWidth-8)
	filled := min(barWidth, max(0, int(ratio*float64(barWidth))))
	bar := pasteProgressDoneStyle.Render(strings.Repeat("█", filled)) +
		pasteProgressLeftStyle.Render(strings.Repeat("░", barWidth-filled))
	progressLine := fmt.Sprintf("%s %3.0f%%", bar, ratio*100)

	detail := "Scanning source"
	if m.paste.phase == pastePhaseCopying {
		detail = fmt.Sprintf("%s / %s   %d / %d items",
			formatBytes(m.paste.completedBytes),
			formatBytes(m.paste.totalBytes),
			m.paste.completedItems,
			m.paste.totalItems,
		)
	}
	current := m.paste.currentPath
	if current == "" || current == "." {
		current = filepath.Base(m.paste.source)
	}
	current = ansi.Truncate(current, contentWidth, "…")

	body := strings.Join([]string{
		headerTitleStyle.Render(title),
		phase,
		progressLine,
		detail,
		hintStyle.Render(current),
		keyHint("Esc", "cancel"),
	}, "\n")
	return pasteModalStyle.Width(contentWidth).Render(body)
}

func pasteRatio(state *pasteState) float64 {
	if state.phase == pastePhasePreparing {
		return 0
	}
	if state.totalBytes > 0 {
		return min(1, float64(state.completedBytes)/float64(state.totalBytes))
	}
	if state.totalItems > 0 {
		return min(1, float64(state.completedItems)/float64(state.totalItems))
	}
	return 0
}

func formatBytes(size int64) string {
	const unit = int64(1024)
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	divisor := unit
	unitName := "KiB"
	for _, next := range []string{"MiB", "GiB", "TiB"} {
		if size < divisor*unit {
			break
		}
		divisor *= unit
		unitName = next
	}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(divisor), unitName)
}

func overlayCentered(background, foreground string, width, height int) string {
	if width <= 0 || height <= 0 {
		return foreground
	}

	foregroundLines := strings.Split(foreground, "\n")
	foregroundWidth := min(width, lipgloss.Width(foreground))
	if len(foregroundLines) > height {
		foregroundLines = foregroundLines[:height]
	}
	left := max(0, (width-foregroundWidth)/2)
	top := max(0, (height-len(foregroundLines))/2)
	backgroundLines := strings.Split(background, "\n")
	result := make([]string, height)

	for row := range height {
		backgroundLine := ""
		if row < len(backgroundLines) {
			backgroundLine = backgroundLines[row]
		}
		if row < top || row >= top+len(foregroundLines) {
			result[row] = padLine(ansi.Cut(backgroundLine, 0, width), width)
			continue
		}

		foregroundLine := padLine(ansi.Cut(foregroundLines[row-top], 0, foregroundWidth), foregroundWidth)
		leftPart := padLine(ansi.Cut(backgroundLine, 0, left), left)
		rightStart := left + foregroundWidth
		rightPart := ansi.Cut(backgroundLine, rightStart, width)
		result[row] = padLine(leftPart+foregroundLine+rightPart, width)
	}

	return strings.Join(result, "\n")
}

func padLine(line string, width int) string {
	missing := width - lipgloss.Width(line)
	if missing <= 0 {
		return line
	}
	return line + strings.Repeat(" ", missing)
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
