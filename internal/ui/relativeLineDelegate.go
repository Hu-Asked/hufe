package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

    if index == currentIdx {
        combined := numStr + text
        renderedLine = d.styles.SelectedTitle.Render(combined)
    } else {
        numStyle := lipgloss.NewStyle().Foreground(colors.HintForeground)
        
        numRendered := numStyle.Render(numStr)
        textRendered := d.styles.NormalTitle.Render(text)
        
        renderedLine = numRendered + textRendered
    }

    fmt.Fprint(w, renderedLine)
}
