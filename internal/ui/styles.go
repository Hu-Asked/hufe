package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type colorScheme struct {
	HeaderForeground          lipgloss.Color
	HeaderBackground          lipgloss.Color
	BoxBorder                 lipgloss.Color
	ListItemForeground        lipgloss.Color
	ListItemDimForeground     lipgloss.Color
	ListSelectedForeground    lipgloss.Color
	ListSelectedBackground    lipgloss.Color
	ListFilterMatchForeground lipgloss.Color
	PathForeground            lipgloss.Color
	KeyForeground             lipgloss.Color
	HintForeground            lipgloss.Color
	StatusForeground          lipgloss.Color
	StatusErrorForeground     lipgloss.Color
}

var colors = colorScheme{
	HeaderForeground:          lipgloss.Color("230"),
	HeaderBackground:          lipgloss.Color("62"),
	BoxBorder:                 lipgloss.Color("238"),
	ListItemForeground:        lipgloss.Color("252"),
	ListItemDimForeground:     lipgloss.Color("240"),
	ListSelectedForeground:    lipgloss.Color("229"),
	ListSelectedBackground:    lipgloss.Color("57"),
	ListFilterMatchForeground: lipgloss.Color("205"),
	PathForeground:            lipgloss.Color("75"),
	KeyForeground:             lipgloss.Color("205"),
	HintForeground:            lipgloss.Color("244"),
	StatusForeground:          lipgloss.Color("244"),
	StatusErrorForeground:     lipgloss.Color("196"),
}

var (
	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colors.HeaderForeground).
				Background(colors.HeaderBackground).
				Padding(0, 1)

	headerBarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colors.BoxBorder)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.BoxBorder)

	pathStyle = lipgloss.NewStyle().
			Foreground(colors.PathForeground).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(colors.KeyForeground).
			Bold(true)

	hintStyle = lipgloss.NewStyle().Foreground(colors.HintForeground)

	statusStyle = lipgloss.NewStyle().Foreground(colors.StatusForeground)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(colors.StatusErrorForeground).
				Bold(true)
)

func newList(items []list.Item) list.Model {
	delegate := relativeLineDelegate{
		styles: itemStyles(), 
	}

	l := list.New(items, delegate, 0, 0)

	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.Styles.Title = headerTitleStyle
	l.Styles.TitleBar = headerBarStyle

	return l
}

func itemStyles() list.DefaultItemStyles {
	styles := list.NewDefaultItemStyles()
	styles.NormalTitle = styles.NormalTitle.Foreground(colors.ListItemForeground)
	styles.DimmedTitle = styles.DimmedTitle.Foreground(colors.ListItemDimForeground)
	styles.FilterMatch = styles.FilterMatch.Foreground(colors.ListFilterMatchForeground).Bold(true)
	styles.SelectedTitle = styles.SelectedTitle.
		Foreground(colors.ListSelectedForeground).
		Background(colors.ListSelectedBackground).
		Bold(true)
	styles.SelectedDesc = styles.SelectedTitle

	return styles
}

func renderStatusLine(path string, status string, statusIsError bool, jumpMulti int) string {
	keyHints := strings.Join([]string{
		keyHint("Enter", "cd+quit"),
		keyHint("l", "open"),
		keyHint("h", "prev"),
		keyHint("q", "quit"),
	}, "  ")

	base := fmt.Sprintf("%s  |  %s  | %d", pathStyle.Render(path), keyHints, jumpMulti)
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
