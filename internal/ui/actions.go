package ui

import (
	"path/filepath"

	"hufe/internal/explorer"
	"hufe/internal/opener"

	tea "github.com/charmbracelet/bubbletea"
)

type openFileResult struct {
	err error
}

func (m *Model) handleEnter() tea.Cmd {
	selected := m.list.SelectedItem()
	if selected == nil {
		return nil
	}

	selectedItem, ok := selected.(item)
	if !ok {
		return nil
	}

	entry := selectedItem.entry
	if !entry.IsDir {
		m.setErrorMessage("not a directory")
		return nil
	}

	m.exitDir = entry.Path
	return tea.Quit
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
			m.setError(err)
		}
		return nil
	}

	return m.openFileCmd(entry.Path)
}

func (m *Model) loadPrev() {
	parent := filepath.Dir(m.cwd)
	if parent == m.cwd {
		return
	}
	if err := m.loadDir(parent); err != nil {
		m.setError(err)
	}
}

func (m *Model) loadDir(path string) error {
	entries, err := explorer.ReadEntries(path)
	if err != nil {
		return err
	}

	m.cwd = path
	m.updateTitle()
	m.clearStatus()
	m.list.SetItems(itemsFromEntries(entries))
	m.list.Select(0)

	return nil
}

func (m *Model) openFileCmd(path string) tea.Cmd {
	cmd, err := opener.Command(path)
	if err != nil {
		m.setError(err)
		return nil
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return openFileResult{err: err}
	})
}
