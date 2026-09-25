package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"hufe/internal/explorer"
)

type Model struct {
	list            list.Model
	cwd             string
	status          string
	statusIsError   bool
	exitDir         string
	boxWidth        int
	windowWidth     int
	windowHeight    int
	jumpMulti       int
	pathToCopy      string
	pathsToCopy     []string
	history         []selectionHistoryEntry
	showHidden      bool
	showHelp        bool
	selectionMode   bool
	selectionAnchor int
	pywalPath       string
	pywalSeen       bool
	pywalInfo       os.FileInfo

	previewWidth       int
	previewHeight      int
	previewPath        string
	previewName        string
	previewIsDir       bool
	previewEntries     []explorer.Entry
	previewFileLines   []string
	previewTruncated   bool
	previewUnsupported bool
	previewErr         error
	previewImage       bool
	previewImageFile   string
	previewImageTemp   bool
	previewImageCancel context.CancelFunc
	previewImageCh     chan imagePreviewMsg
	previewImageSeq    uint64
	kittyGraphics      bool
	kittyImageID       uint32

	searchInput     textinput.Model
	searchMode      bool
	recursiveSearch bool
	allEntries      []explorer.Entry
	searchFiles     []FileEntry

	paste           *pasteState
	pasteProgressCh <-chan pasteProgressMsg
	deletion        *deleteState
	deleteProgress  <-chan deleteProgressMsg
	rename          *renameState
	creation        *creationState
}

type renameState struct {
	source string
	input  textinput.Model
	err    error
}

type creationState struct {
	input textinput.Model
	err   error
}

type pasteState struct {
	source           string
	sources          []string
	phase            pastePhase
	completedBytes   int64
	totalBytes       int64
	completedItems   int
	totalItems       int
	currentPath      string
	completedSources int
	totalSources     int
	cancelling       bool
	cancel           context.CancelFunc
}

type deleteState struct {
	source           string
	sources          []string
	trashDirectory   string
	kind             string
	selectionIndex   int
	phase            deletePhase
	completedBytes   int64
	totalBytes       int64
	completedItems   int
	totalItems       int
	currentPath      string
	completedSources int
	totalSources     int
}

type selectionHistoryEntry struct {
	directory string
	index     int
}

func NewModel(startDir string) (*Model, error) {
	entries, err := explorer.ReadEntriesWithHidden(startDir, false)
	if err != nil {
		return nil, err
	}

	applyColorScheme(defaultColors)
	l := newList(itemsFromEntries(entries))
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 156
	ti.Width = 20
	styleTextInput(&ti)

	m := &Model{
		list:           l,
		cwd:            startDir,
		searchInput:    ti,
		pywalPath:      pywalColorsPath(),
		previewImageCh: make(chan imagePreviewMsg, 16),
		kittyGraphics:  os.Getenv("TERM") == "xterm-kitty",
		kittyImageID:   newKittyImageID(),
	}
	m.refreshPywalScheme()
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
	m.updateSelectionRange()
}

func (m *Model) startSelectionMode() {
	if len(m.list.Items()) == 0 {
		return
	}
	m.selectionMode = true
	m.selectionAnchor = m.list.Index()
	m.updateSelectionRange()
}

func (m *Model) stopSelectionMode() {
	m.selectionMode = false
	m.selectionAnchor = 0
	m.updateSelectionRange()
}

func (m *Model) updateSelectionRange() {
	start, end := m.selectionAnchor, m.list.Index()
	if start > end {
		start, end = end, start
	}
	for index, listItem := range m.list.Items() {
		entryItem, ok := listItem.(item)
		if !ok {
			continue
		}
		selected := m.selectionMode && index >= start && index <= end
		if entryItem.rangeSelected == selected {
			continue
		}
		entryItem.rangeSelected = selected
		m.list.SetItem(index, entryItem)
	}
}
