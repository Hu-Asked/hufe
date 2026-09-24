package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type colorScheme struct {
	HeaderForeground          lipgloss.Color
	HeaderBackground          lipgloss.Color
	BoxBorder                 lipgloss.Color
	ListItemForeground        lipgloss.Color
	ListItemDimForeground     lipgloss.Color
	ListCursorIndicator       lipgloss.Color
	ListSelectedForeground    lipgloss.Color
	ListSelectedBackground    lipgloss.Color
	ListFilterMatchForeground lipgloss.Color
	PathForeground            lipgloss.Color
	KeyForeground             lipgloss.Color
	HintForeground            lipgloss.Color
	StatusForeground          lipgloss.Color
	StatusErrorForeground     lipgloss.Color
}

var defaultColors = colorScheme{
	HeaderForeground:          lipgloss.Color("230"),
	HeaderBackground:          lipgloss.Color("62"),
	BoxBorder:                 lipgloss.Color("238"),
	ListItemForeground:        lipgloss.Color("252"),
	ListItemDimForeground:     lipgloss.Color("240"),
	ListCursorIndicator:       lipgloss.Color("75"),
	ListSelectedForeground:    lipgloss.Color("229"),
	ListSelectedBackground:    lipgloss.Color("57"),
	ListFilterMatchForeground: lipgloss.Color("205"),
	PathForeground:            lipgloss.Color("75"),
	KeyForeground:             lipgloss.Color("205"),
	HintForeground:            lipgloss.Color("244"),
	StatusForeground:          lipgloss.Color("244"),
	StatusErrorForeground:     lipgloss.Color("196"),
}

var colors = defaultColors

var (
	headerTitleStyle       lipgloss.Style
	headerBarStyle         lipgloss.Style
	boxStyle               lipgloss.Style
	pathStyle              lipgloss.Style
	keyStyle               lipgloss.Style
	helpKeyStyle           lipgloss.Style
	hintStyle              lipgloss.Style
	statusStyle            lipgloss.Style
	statusErrorStyle       lipgloss.Style
	modalStyle             lipgloss.Style
	pasteProgressDoneStyle lipgloss.Style
	pasteProgressLeftStyle lipgloss.Style
)

func init() {
	applyColorScheme(defaultColors)
}

func applyColorScheme(scheme colorScheme) {
	colors = scheme
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

	helpKeyStyle = keyStyle.Width(16)

	hintStyle = lipgloss.NewStyle().Foreground(colors.HintForeground)

	statusStyle = lipgloss.NewStyle().Foreground(colors.StatusForeground)

	statusErrorStyle = lipgloss.NewStyle().
		Foreground(colors.StatusErrorForeground).
		Bold(true)

	modalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colors.KeyForeground).
		Padding(1, 2)

	pasteProgressDoneStyle = lipgloss.NewStyle().
		Foreground(colors.KeyForeground)

	pasteProgressLeftStyle = lipgloss.NewStyle().
		Foreground(colors.ListItemDimForeground)
}

func styleTextInput(input *textinput.Model) {
	input.PromptStyle = lipgloss.NewStyle().Foreground(colors.KeyForeground)
	input.TextStyle = lipgloss.NewStyle().Foreground(colors.ListItemForeground)
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.HintForeground)
	input.CompletionStyle = lipgloss.NewStyle().Foreground(colors.ListItemDimForeground)
	input.Cursor.Style = lipgloss.NewStyle().Foreground(colors.HeaderForeground).Background(colors.HeaderBackground)
}

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
		Foreground(colors.PathForeground).
		BorderForeground(colors.ListCursorIndicator).
		Bold(true)
	styles.SelectedDesc = styles.SelectedTitle

	return styles
}

func renderStatusLine(path string, status string, statusIsError bool, jumpMulti int, selectionCount int) string {
	mode := fmt.Sprintf("%d", jumpMulti)
	if selectionCount > 0 {
		mode = keyStyle.Render(fmt.Sprintf("SELECT %d", selectionCount))
	}
	base := fmt.Sprintf("%s  |  %s  |  %s", pathStyle.Render(path), keyHint("Ctrl+H", "help"), mode)
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
