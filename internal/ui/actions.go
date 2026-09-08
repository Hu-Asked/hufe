package ui

import (
	"errors"
	"fmt"
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
	m.previewPath = ""
	m.refreshPreview()
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
		currentParent := filepath.Clean(filepath.Dir(m.cwd))
		selectedDir := filepath.Clean(entry.Path)
		if selectedDir == currentParent {
			return nil
		}

		if err := m.loadDir(entry.Path); err != nil {
			m.setError(err)
		}
		return nil
	} else if m.searchMode {
		if err := m.loadDir(filepath.Dir(entry.Path)); err != nil {
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
	m.previewPath = ""
	m.refreshPreview()

	if m.searchMode {
		m.cancelSearch()
	}

	return nil
}

func (m *Model) initSearch(recursive bool) {
	m.searchMode = true
	m.recursiveSearch = recursive
	m.searchInput.Focus()
	m.searchInput.SetValue("")

	var entries []explorer.Entry
	var err error
	if recursive {
		entries, err = explorer.ReadEntriesRecursive(m.cwd)
	} else {
		entries, err = explorer.ReadEntries(m.cwd)
	}

	if err != nil {
		m.setError(err)
		m.cancelSearch()
		return
	}

	m.allEntries = entries
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	m.searchFiles = PreProcessPaths(paths)
	m.list.SetItems(itemsFromEntries(entries))
}

func (m *Model) handleSearchInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	query := m.searchInput.Value()
	if query == "" {
		m.list.SetItems(itemsFromEntries(m.allEntries))
		return cmd
	}

	results := FuzzyFind(query, m.searchFiles)

	pathMap := make(map[string]explorer.Entry)
	for _, e := range m.allEntries {
		pathMap[e.Path] = e
	}

	var filtered []explorer.Entry
	for _, r := range results {
		if entry, ok := pathMap[r.Path]; ok {
			filtered = append(filtered, entry)
		}
	}

	m.list.SetItems(itemsFromEntries(filtered))
	m.list.Select(0)
	return cmd
}

func (m *Model) cancelSearch() {
	m.searchMode = false
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.searchFiles = nil
	m.allEntries = nil
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
