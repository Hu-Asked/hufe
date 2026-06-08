package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/epilande/go-devicons"
)

type relativeLineDelegate struct {
	styles list.DefaultItemStyles
}

func (d relativeLineDelegate) Height() int { return 1 }

func (d relativeLineDelegate) Spacing() int { return 0 }

func (d relativeLineDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d relativeLineDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if listItem == nil {
		return
	}

	currentIdx := m.Index()
	relative := index - currentIdx
	if relative < 0 {
		relative = -relative
	}

	numStr := fmt.Sprintf("%2d ", relative)

	text := listItem.FilterValue()

	var renderedLine string

	itemIcon := devicons.IconForPath(listItem.(item).entry.Path)
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(itemIcon.Color))

	totalWidth := m.Width()
    iconWidth := lipgloss.Width(itemIcon.Icon)

    if index == currentIdx {
        selectedBg := d.styles.SelectedTitle.GetBackground()
        selectedFg := d.styles.SelectedTitle.GetForeground()

        numRendered := d.styles.SelectedTitle.Render(numStr)
        numWidth := lipgloss.Width(numRendered)

        textStyle := lipgloss.NewStyle().
            Background(selectedBg).
            Foreground(selectedFg).
            MaxWidth(totalWidth - numWidth - iconWidth).
            Inline(true)
        
        textRendered := textStyle.Render(text)
        textWidth := lipgloss.Width(textRendered)
        itemIconRendered := iconStyle.Background(selectedBg).Render(itemIcon.Icon)
        actualIconWidth := lipgloss.Width(itemIconRendered)

        paddingWidth := totalWidth - numWidth - textWidth - actualIconWidth
		paddingWidth = max(0, paddingWidth)
        spaces := strings.Repeat(" ", paddingWidth)
        spacesRendered := lipgloss.NewStyle().Background(selectedBg).Render(spaces)

        renderedLine = numRendered + textRendered + spacesRendered + itemIconRendered
    } else {
        numRendered := lipgloss.NewStyle().Foreground(colors.HintForeground).Render(numStr)
        numWidth := lipgloss.Width(numRendered)

        textStyle := d.styles.NormalTitle.MaxWidth(totalWidth - numWidth - iconWidth).Inline(true)
        textRendered := textStyle.Render(text)
        textWidth := lipgloss.Width(textRendered)

        itemIconRendered := iconStyle.Render(itemIcon.Icon)
        actualIconWidth := lipgloss.Width(itemIconRendered) + 1

        paddingWidth := totalWidth - numWidth - textWidth - actualIconWidth
		paddingWidth = max(0, paddingWidth)
        spaces := strings.Repeat(" ", paddingWidth)

        renderedLine = numRendered + textRendered + spaces + itemIconRendered
    }

	fmt.Fprint(w, renderedLine)
}
