package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"hufe/internal/explorer"
	"hufe/internal/opener"
)

type item struct {
	entry explorer.Entry
}

func (i item) Title() string {
	if i.entry.IsDir && i.entry.Name != ".." {
		return i.entry.Name + string(os.PathSeparator)
	}
	return i.entry.Name
}

func (i item) Description() string {
	return ""
}

func (i item) FilterValue() string {
	return i.entry.Name
}

type openFileResult struct {
	err error
}

type Model struct {
	list   list.Model
	cwd    string
	status string
}

func NewModel(startDir string) (*Model, error) {
	entries, err := explorer.ReadEntries(startDir)
	if err != nil {
		return nil, err
	}

	l := list.New(itemsFromEntries(entries), list.NewDefaultDelegate(), 0, 0)
	l.Title = startDir
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)

	return &Model{
		list:   l,
		cwd:    startDir,
		status: "",
	}, nil
}

func itemsFromEntries(entries []explorer.Entry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, item{entry: entry})
	}
	return items
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "backspace", "h":
			m.goUp()
			return m, nil
		case "enter":
			return m, m.handleSelect()
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(0, msg.Height-1))
		return m, nil
	case openFileResult:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err)
		} else {
			m.status = ""
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	return m.list.View() + "\n" + m.statusLine()
}

func (m *Model) statusLine() string {
	base := fmt.Sprintf("%s  |  Enter: open  Backspace/h: up  q: quit", m.cwd)
	if m.status != "" {
		return fmt.Sprintf("%s  |  %s", base, m.status)
	}
	return base
}

func (m *Model) handleSelect() tea.Cmd {
	selected := m.list.SelectedItem()
	if selected == nil {
		return nil
	}

	selectedItem, ok := selected.(item)
	if !ok {
		return nil
	}

	entry := selectedItem.entry
	if entry.IsDir {
		if err := m.loadDir(entry.Path); err != nil {
			m.status = fmt.Sprintf("Error: %s", err)
		}
		return nil
	}

	return m.openFileCmd(entry.Path)
}

func (m *Model) goUp() {
	parent := filepath.Dir(m.cwd)
	if parent == m.cwd {
		return
	}
	if err := m.loadDir(parent); err != nil {
		m.status = fmt.Sprintf("Error: %s", err)
	}
}

func (m *Model) loadDir(path string) error {
	entries, err := explorer.ReadEntries(path)
	if err != nil {
		return err
	}

	m.cwd = path
	m.status = ""
	m.list.Title = path
	m.list.SetItems(itemsFromEntries(entries))
	m.list.Select(0)

	return nil
}

func (m *Model) openFileCmd(path string) tea.Cmd {
	cmd, err := opener.Command(path)
	if err != nil {
		m.status = fmt.Sprintf("Error: %s", err)
		return nil
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return openFileResult{err: err}
	})
}

