package ui

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"hufe/internal/explorer"
)

type Model struct {
	list          list.Model
	cwd           string
	status        string
	statusIsError bool
	exitDir       string
	boxWidth      int
	jumpMulti     int
	pathToCopy    string
	history       []selectionHistoryEntry

	previewWidth   int
	previewHeight  int
	previewPath    string
	previewName    string
	previewIsDir   bool
	previewEntries []explorer.Entry
	previewErr     error

	searchInput     textinput.Model
	searchMode      bool
	recursiveSearch bool
	allEntries      []explorer.Entry
	searchFiles     []FileEntry
}

type selectionHistoryEntry struct {
	directory string
	index     int
}

func NewModel(startDir string) (*Model, error) {
	entries, err := explorer.ReadEntries(startDir)
	if err != nil {
		return nil, err
	}

	l := newList(itemsFromEntries(entries))
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 156
	ti.Width = 20

	m := &Model{
		list:        l,
		cwd:         startDir,
		searchInput: ti,
	}
	m.updateTitle()
	m.refreshPreview()

	return m, nil
}

func (m *Model) pushHistory(directory string, index int) {
	m.history = append(m.history, selectionHistoryEntry{
		directory: filepath.Clean(directory),
		index:     index,
	})
}

func (m *Model) popHistory(directory string) (int, bool) {
	if len(m.history) == 0 {
		return 0, false
	}

	index := len(m.history) - 1
	element := m.history[index]
	if element.directory != filepath.Clean(directory) {
		return 0, false
	}

	m.history = m.history[:index]
	return element.index, true
}

func (m *Model) updateTitle() {
	base := m.cwd
	if base == "" {
		base = m.cwd
	}
	m.list.Title = base
}

func (m *Model) ExitDir() string {
	return m.exitDir
}

func (m *Model) setStatus(message string, isError bool) {
	m.status = message
	m.statusIsError = isError
}

func (m *Model) clearStatus() {
	m.status = ""
	m.statusIsError = false
}

func (m *Model) setError(err error) {
	m.setStatus(fmt.Sprintf("Error: %s", err), true)
}

func (m *Model) setErrorMessage(message string) {
	m.setStatus(fmt.Sprintf("Error: %s", message), true)
}

func (m *Model) setItem(index int) {
	itemCount := len(m.list.Items())
	if itemCount == 0 {
		return
	}
	index = max(0, min(index, itemCount-1))
	m.list.Select(index)
}
