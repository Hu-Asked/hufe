package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	titleBarStyle = lipgloss.NewStyle().Padding(0, 0, 1, 1)

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	hintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
)

func newList(items []list.Item) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.Styles = itemStyles()

	l := list.New(items, delegate, 0, 0)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.Styles.Title = titleStyle
	l.Styles.TitleBar = titleBarStyle

	return l
}

func itemStyles() list.DefaultItemStyles {
	styles := list.NewDefaultItemStyles()
	styles.NormalTitle = styles.NormalTitle.Foreground(lipgloss.Color("252"))
	styles.DimmedTitle = styles.DimmedTitle.Foreground(lipgloss.Color("240"))
	styles.FilterMatch = styles.FilterMatch.Foreground(lipgloss.Color("205")).Bold(true)
	styles.SelectedTitle = styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	styles.SelectedDesc = styles.SelectedTitle

	return styles
}

func renderStatusLine(path string, status string, statusIsError bool) string {
	keyHints := strings.Join([]string{
		keyHint("Enter", "cd+quit"),
		keyHint("l", "open"),
		keyHint("Backspace/h", "up"),
		keyHint("q", "quit"),
	}, "  ")

	base := fmt.Sprintf("%s  |  %s", pathStyle.Render(path), keyHints)
	if status == "" {
		return base
	}
	if statusIsError {
		return fmt.Sprintf("%s  |  %s", base, statusErrorStyle.Render(status))
	}
	return fmt.Sprintf("%s  |  %s", base, statusStyle.Render(status))
}

func keyHint(key string, hint string) string {
	return fmt.Sprintf("%s %s", keyStyle.Render(key), hintStyle.Render(hint))
}
