package ui

import (
	"fmt"
	"errors"
	"os"
	"path/filepath"

	"hufe/internal/explorer"

	tea "github.com/charmbracelet/bubbletea"
)

type openFileResult struct {
	err error
}

func (m *Model) handleCopy() {
	selected := m.list.SelectedItem()
	if selected == nil {
		m.setError(errors.New("Error: item does not exist"))
		return
	}

	selectedItem, ok := selected.(item)
	if !ok {
		m.setError(errors.New("Error: item not ok"))
		return
	}
	
	entry := selectedItem.entry
	m.pathToCopy = entry.Path
	m.setStatus(fmt.Sprintf("Copied %s", m.pathToCopy), false)
}

func (m *Model) handlePaste() {
	selected := m.list.SelectedItem()
	if m.pathToCopy == "" {
		return
	}
	if selected == nil {
		m.setError(errors.New("Error: destination does not exist"))
		return
	}
	finalTarget := filepath.Join(m.cwd, filepath.Base(m.pathToCopy))
	err := os.CopyFS(finalTarget, os.DirFS(m.pathToCopy))
	if err != nil {
		m.setError(err)
		return
	}
	m.updateTitle()
	m.clearStatus()
	entries, err := explorer.ReadEntries(m.cwd)
	m.list.SetItems(itemsFromEntries(entries))
	if err != nil {
		return
	}
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
	m.exitDir = entry.Path
	if entry.Name == ".." {
		m.exitDir = m.cwd
	}
	if !entry.IsDir {
		m.exitDir = filepath.Dir(entry.Path)
	}

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
		currentParent := filepath.Clean(filepath.Dir(m.cwd));
		selectedDir := filepath.Clean(entry.Path)
		if selectedDir == currentParent {
			return nil
		}

		if err := m.loadDir(entry.Path); err != nil {
			m.setError(err)
		}
		return nil
	}

	return nil // m.openFileCmd(entry.Path)
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

// func (m *Model) openFileCmd(path string) tea.Cmd {
// 	cmd, err := opener.Command(path)
// 	if err != nil {
// 		m.setError(err)
// 		return nil
// 	}
//
// 	return tea.ExecProcess(cmd, func(err error) tea.Msg {
// 		return openFileResult{err: err}
// 	})
// }

// func (m *Model) copyTo (pathToCopy string, targetDirectory string) tea.Cmd {
//
// }
