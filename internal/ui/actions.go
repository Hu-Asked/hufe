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
	selectedItems := m.operationItems()
	if len(selectedItems) == 0 {
		m.setError(errors.New("Error: item does not exist"))
		return
	}

	paths := make([]string, 0, len(selectedItems))
	for _, selectedItem := range selectedItems {
		paths = append(paths, selectedItem.entry.Path)
	}
	m.pathsToCopy = paths
	m.pathToCopy = paths[0]
	m.stopSelectionMode()
	if len(paths) == 1 {
		m.setStatus(fmt.Sprintf("Copied %s", paths[0]), false)
	} else {
		m.setStatus(fmt.Sprintf("Copied %d items", len(paths)), false)
	}
}

func (m *Model) handlePaste() tea.Cmd {
	sources := m.copiedPaths()
	if len(sources) == 0 {
		m.setErrorMessage("nothing has been copied")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	progressChannel := make(chan pasteProgressMsg, 1)
	m.paste = &pasteState{
		source:       sources[0],
		sources:      sources,
		phase:        pastePhasePreparing,
		totalSources: len(sources),
		cancel:       cancel,
	}
	m.pasteProgressCh = progressChannel
	m.clearStatus()

	return tea.Batch(
		runPasteSourcesCmd(ctx, sources, m.cwd, progressChannel),
		waitForPasteProgressCmd(progressChannel),
	)
}

func (m *Model) copiedPaths() []string {
	if len(m.pathsToCopy) > 0 {
		return append([]string(nil), m.pathsToCopy...)
	}
	if m.pathToCopy != "" {
		return []string{m.pathToCopy}
	}
	return nil
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

func runPasteSourcesCmd(ctx context.Context, sources []string, destination string, progressChannel chan pasteProgressMsg) tea.Cmd {
	return func() tea.Msg {
		defer close(progressChannel)
		message := pasteFinishedMsg{}
		for index, source := range sources {
			result, err := fileops.Copy(ctx, source, destination, func(progress fileops.Progress) {
				progressMessage := pasteProgressMsg{
					phase:            pastePhase(progress.Phase),
					completedBytes:   progress.CompletedBytes,
					totalBytes:       progress.TotalBytes,
					completedItems:   progress.CompletedItems,
					totalItems:       progress.TotalItems,
					currentPath:      progress.CurrentPath,
					completedSources: index,
					totalSources:     len(sources),
				}
				select {
				case progressChannel <- progressMessage:
				default:
				}
			})
			if err != nil {
				message.err = err
				return message
			}
			message.result = result
			message.targets = append(message.targets, result.Target)
			select {
			case progressChannel <- pasteProgressMsg{
				phase:            pastePhaseCopying,
				completedSources: index + 1,
				totalSources:     len(sources),
				currentPath:      filepath.Base(source),
			}:
			default:
			}
		}
		return message
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
	selectedItems := m.operationItems()
	if len(selectedItems) == 0 {
		m.setErrorMessage("item does not exist")
		return
	}

	paths := make([]string, 0, len(selectedItems))
	trashDirectory := ""
	kind := "items"
	for _, selected := range selectedItems {
		if selected.entry.Name == ".." {
			m.setErrorMessage("the parent directory cannot be deleted")
			return
		}
		validatedTrash, err := fileops.ValidateTrashDestination(selected.entry.Path, os.Getenv("HUFE_TRASH_DIR"))
		if err != nil {
			m.setError(err)
			return
		}
		trashDirectory = validatedTrash
		info, err := os.Lstat(selected.entry.Path)
		if err != nil {
			m.setError(err)
			return
		}
		if len(selectedItems) == 1 {
			kind = "file"
			if info.Mode()&os.ModeSymlink != 0 {
				kind = "symbolic link"
			} else if info.IsDir() {
				kind = "directory and all of its contents"
			}
		}
		paths = append(paths, selected.entry.Path)
	}

	m.deletion = &deleteState{
		source:         paths[0],
		sources:        paths,
		trashDirectory: trashDirectory,
		kind:           kind,
		selectionIndex: m.list.Index(),
		phase:          deletePhaseConfirming,
		totalSources:   len(paths),
	}
	m.stopSelectionMode()
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
		runDeleteSourcesCmd(m.deletion.sourcesOrSource(), m.deletion.trashDirectory, progressChannel),
		waitForDeleteProgressCmd(progressChannel),
	)
}

func (state *deleteState) sourcesOrSource() []string {
	if len(state.sources) > 0 {
		return append([]string(nil), state.sources...)
	}
	if state.source != "" {
		return []string{state.source}
	}
	return nil
}

func runDeleteSourcesCmd(sources []string, trashDirectory string, progressChannel chan deleteProgressMsg) tea.Cmd {
	return func() tea.Msg {
		defer close(progressChannel)
		message := deleteFinishedMsg{}
		for index, source := range sources {
			result, err := fileops.MoveToTrash(context.Background(), source, trashDirectory, func(progress fileops.Progress) {
				progressMessage := deleteProgressMsg{
					completedBytes:   progress.CompletedBytes,
					totalBytes:       progress.TotalBytes,
					completedItems:   progress.CompletedItems,
					totalItems:       progress.TotalItems,
					currentPath:      progress.CurrentPath,
					completedSources: index,
					totalSources:     len(sources),
				}
				select {
				case progressChannel <- progressMessage:
				default:
				}
			})
			if err != nil {
				message.err = err
				return message
			}
			message.result = result
			message.results = append(message.results, result)
			select {
			case progressChannel <- deleteProgressMsg{
				completedSources: index + 1,
				totalSources:     len(sources),
				currentPath:      filepath.Base(source),
			}:
			default:
			}
		}
		return message
	}
}

func (m *Model) operationItems() []item {
	if !m.selectionMode {
		selected, ok := m.list.SelectedItem().(item)
		if !ok {
			return nil
		}
		return []item{selected}
	}

	start, end := m.selectionAnchor, m.list.Index()
	if start > end {
		start, end = end, start
	}
	items := make([]item, 0, end-start+1)
	for index := start; index <= end && index < len(m.list.Items()); index++ {
		if selected, ok := m.list.Items()[index].(item); ok {
			items = append(items, selected)
		}
	}
	return items
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
	m.selectionMode = false
	m.selectionAnchor = 0
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
	m.stopSelectionMode()
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
	m.stopSelectionMode()
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
