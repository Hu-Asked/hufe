package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"hufe/internal/explorer"
	"hufe/internal/fileops"

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

func (m *Model) handlePaste() tea.Cmd {
	if m.pathToCopy == "" {
		m.setErrorMessage("nothing has been copied")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	progressChannel := make(chan pasteProgressMsg, 1)
	m.paste = &pasteState{
		source: m.pathToCopy,
		phase:  pastePhasePreparing,
		cancel: cancel,
	}
	m.pasteProgressCh = progressChannel
	m.clearStatus()

	return tea.Batch(
		runPasteCmd(ctx, m.pathToCopy, m.cwd, progressChannel),
		waitForPasteProgressCmd(progressChannel),
	)
}

func runPasteCmd(ctx context.Context, source, destination string, progressChannel chan pasteProgressMsg) tea.Cmd {
	return func() tea.Msg {
		defer close(progressChannel)
		result, err := fileops.Copy(ctx, source, destination, func(progress fileops.Progress) {
			message := pasteProgressMsg{
				phase:          pastePhase(progress.Phase),
				completedBytes: progress.CompletedBytes,
				totalBytes:     progress.TotalBytes,
				completedItems: progress.CompletedItems,
				totalItems:     progress.TotalItems,
				currentPath:    progress.CurrentPath,
			}
			select {
			case progressChannel <- message:
			default:
			}
		})
		return pasteFinishedMsg{result: result, err: err}
	}
}

func waitForPasteProgressCmd(progressChannel <-chan pasteProgressMsg) tea.Cmd {
	return func() tea.Msg {
		message, ok := <-progressChannel
		if !ok {
			return nil
		}
		return message
	}
}

func (m *Model) handleDelete() {
	selected, ok := m.list.SelectedItem().(item)
	if !ok {
		m.setErrorMessage("item does not exist")
		return
	}
	if selected.entry.Name == ".." {
		m.setErrorMessage("the parent directory cannot be deleted")
		return
	}

	trashDirectory, err := fileops.ValidateTrashDestination(selected.entry.Path, os.Getenv("HUFE_TRASH_DIR"))
	if err != nil {
		m.setError(err)
		return
	}
	info, err := os.Lstat(selected.entry.Path)
	if err != nil {
		m.setError(err)
		return
	}
	kind := "file"
	if info.Mode()&os.ModeSymlink != 0 {
		kind = "symbolic link"
	} else if info.IsDir() {
		kind = "directory and all of its contents"
	}

	m.deletion = &deleteState{
		source:         selected.entry.Path,
		trashDirectory: trashDirectory,
		kind:           kind,
		selectionIndex: m.list.Index(),
		phase:          deletePhaseConfirming,
	}
	m.clearStatus()
}

func (m *Model) confirmDelete() tea.Cmd {
	if m.deletion == nil || m.deletion.phase != deletePhaseConfirming {
		return nil
	}
	m.deletion.phase = deletePhaseMoving
	progressChannel := make(chan deleteProgressMsg, 1)
	m.deleteProgress = progressChannel

	return tea.Batch(
		runDeleteCmd(m.deletion.source, m.deletion.trashDirectory, progressChannel),
		waitForDeleteProgressCmd(progressChannel),
	)
}

func runDeleteCmd(source, trashDirectory string, progressChannel chan deleteProgressMsg) tea.Cmd {
	return func() tea.Msg {
		defer close(progressChannel)
		result, err := fileops.MoveToTrash(context.Background(), source, trashDirectory, func(progress fileops.Progress) {
			message := deleteProgressMsg{
				completedBytes: progress.CompletedBytes,
				totalBytes:     progress.TotalBytes,
				completedItems: progress.CompletedItems,
				totalItems:     progress.TotalItems,
				currentPath:    progress.CurrentPath,
			}
			select {
			case progressChannel <- message:
			default:
			}
		})
		return deleteFinishedMsg{result: result, err: err}
	}
}

func waitForDeleteProgressCmd(progressChannel <-chan deleteProgressMsg) tea.Cmd {
	return func() tea.Msg {
		message, ok := <-progressChannel
		if !ok {
			return nil
		}
		return message
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
		currentParent := filepath.Clean(filepath.Dir(m.cwd))
		selectedDir := filepath.Clean(entry.Path)
		if selectedDir == currentParent {
			return nil
		}

		previousDir := m.cwd
		previousIndex := m.list.Index()
		if err := m.loadDir(entry.Path); err != nil {
			m.setError(err)
		} else {
			m.pushHistory(previousDir, previousIndex)
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

	previousDir := m.cwd
	restoreIndex, hasHistory := m.popHistory(parent)
	if err := m.loadDir(parent); err != nil {
		if hasHistory {
			m.pushHistory(parent, restoreIndex)
		}
		m.setError(err)
		return
	}

	if hasHistory {
		m.setItem(restoreIndex)
	} else {
		m.selectPath(previousDir)
	}
	m.previewPath = ""
	m.refreshPreview()
}

func (m *Model) loadDir(path string) error {
	entries, err := explorer.ReadEntriesWithHidden(path, m.showHidden)
	if err != nil {
		return err
	}

	m.cwd = path
	m.updateTitle()
	m.clearStatus()
	m.list.SetItems(itemsFromEntries(entries))
	m.setItem(0)
	m.previewPath = ""
	m.refreshPreview()

	if m.searchMode {
		m.cancelSearch()
	}

	return nil
}

func (m *Model) selectPath(path string) bool {
	wanted := filepath.Clean(path)
	for index, listItem := range m.list.Items() {
		entryItem, ok := listItem.(item)
		if ok && filepath.Clean(entryItem.entry.Path) == wanted {
			m.setItem(index)
			return true
		}
	}
	return false
}

func (m *Model) initSearch(recursive bool) {
	m.searchMode = true
	m.recursiveSearch = recursive
	m.searchInput.Focus()
	m.searchInput.SetValue("")

	var entries []explorer.Entry
	var err error
	if recursive {
		entries, err = explorer.ReadEntriesRecursiveWithHidden(m.cwd, m.showHidden)
	} else {
		entries, err = explorer.ReadEntriesWithHidden(m.cwd, m.showHidden)
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

func (m *Model) toggleHidden() {
	showHidden := !m.showHidden
	var entries []explorer.Entry
	var err error
	if m.searchMode && m.recursiveSearch {
		entries, err = explorer.ReadEntriesRecursiveWithHidden(m.cwd, showHidden)
	} else {
		entries, err = explorer.ReadEntriesWithHidden(m.cwd, showHidden)
	}
	if err != nil {
		m.setError(err)
		return
	}

	selectedPath := ""
	if selected, ok := m.list.SelectedItem().(item); ok {
		selectedPath = selected.entry.Path
	}
	m.showHidden = showHidden

	if m.searchMode {
		m.allEntries = entries
		paths := make([]string, len(entries))
		for i, entry := range entries {
			paths[i] = entry.Path
		}
		m.searchFiles = PreProcessPaths(paths)
		m.applySearch(m.searchInput.Value())
	} else {
		m.list.SetItems(itemsFromEntries(entries))
	}

	if selectedPath == "" || !m.selectPath(selectedPath) {
		m.setItem(0)
	}
	m.clearPreview()
	m.refreshPreview()

	if m.showHidden {
		m.setStatus("Hidden files shown", false)
	} else {
		m.setStatus("Hidden files hidden", false)
	}
}

func (m *Model) handleSearchInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	m.applySearch(m.searchInput.Value())
	return cmd
}

func (m *Model) applySearch(query string) {
	if query == "" {
		m.list.SetItems(itemsFromEntries(m.allEntries))
		return
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
	m.setItem(0)
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
