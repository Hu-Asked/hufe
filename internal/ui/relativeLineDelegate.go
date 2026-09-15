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

	entryItem, ok := listItem.(item)
	if !ok {
		return
	}
	text := listItem.FilterValue()
	itemIcon := devicons.IconForPath(entryItem.entry.Path)

	numberStyle := lipgloss.NewStyle().Foreground(colors.HintForeground)
	textStyle := d.styles.NormalTitle
	backgroundStyle := lipgloss.NewStyle()
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(itemIcon.Color))
	if index == currentIdx {
		numberStyle = d.styles.SelectedTitle
		textStyle = lipgloss.NewStyle().
			Foreground(d.styles.SelectedTitle.GetForeground()).
			Bold(true)
	}
	if entryItem.rangeSelected {
		numberStyle = numberStyle.
			Foreground(colors.ListSelectedForeground).
			Background(colors.ListSelectedBackground)
		textStyle = textStyle.
			Foreground(colors.ListSelectedForeground).
			Background(colors.ListSelectedBackground)
		backgroundStyle = backgroundStyle.Background(colors.ListSelectedBackground)
		iconStyle = iconStyle.Background(colors.ListSelectedBackground)
	}

	totalWidth := m.Width()
	numberRendered := numberStyle.Render(numStr)
	numberWidth := lipgloss.Width(numberRendered)
	iconWidth := lipgloss.Width(itemIcon.Icon) + 1
	textRendered := textStyle.
		MaxWidth(max(0, totalWidth-numberWidth-iconWidth)).
		Inline(true).
		Render(text)
	paddingWidth := max(0, totalWidth-numberWidth-lipgloss.Width(textRendered)-iconWidth)
	padding := backgroundStyle.Render(strings.Repeat(" ", paddingWidth))
	iconRendered := iconStyle.Render(itemIcon.Icon)

	fmt.Fprint(w, numberRendered+textRendered+padding+iconRendered)
}
