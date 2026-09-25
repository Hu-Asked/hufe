package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"hufe/internal/fileops"
)

type pastePhase uint8

const (
	pastePhasePreparing pastePhase = iota
	pastePhaseCopying
)

type pasteProgressMsg struct {
	phase            pastePhase
	completedBytes   int64
	totalBytes       int64
	completedItems   int
	totalItems       int
	currentPath      string
	completedSources int
	totalSources     int
}

type pasteFinishedMsg struct {
	result  fileops.Result
	targets []string
	err     error
}

type deletePhase uint8

const (
	deletePhaseConfirming deletePhase = iota
	deletePhaseMoving
)

type deleteProgressMsg struct {
	completedBytes   int64
	totalBytes       int64
	completedItems   int
	totalItems       int
	currentPath      string
	completedSources int
	totalSources     int
}

type deleteFinishedMsg struct {
	result  fileops.Result
	results []fileops.Result
	err     error
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(nextPywalTick(), waitForImagePreview(m.previewImageCh))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imagePreviewMsg:
		m.acceptImagePreview(msg)
		return m, waitForImagePreview(m.previewImageCh)
	case pywalTickMsg:
		m.refreshPywalScheme()
		return m, nextPywalTick()
	case tea.KeyMsg:
		model, cmd := m.handleKey(msg)
		m.refreshPreview()
		return model, cmd
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		availableWidth := max(0, msg.Width-5)
		m.boxWidth = availableWidth / 2
		m.previewWidth = availableWidth - m.boxWidth
		m.previewHeight = max(0, msg.Height-3)
		m.list.SetSize(m.boxWidth, m.previewHeight)
		m.list.Styles.TitleBar = headerBarStyle.Width(m.boxWidth)
		return m, nil
	case pasteProgressMsg:
		if m.paste == nil {
			return m, nil
		}
		m.paste.phase = msg.phase
		m.paste.completedBytes = msg.completedBytes
		m.paste.totalBytes = msg.totalBytes
		m.paste.completedItems = msg.completedItems
		m.paste.totalItems = msg.totalItems
		m.paste.currentPath = msg.currentPath
		m.paste.completedSources = msg.completedSources
		m.paste.totalSources = msg.totalSources
		return m, waitForPasteProgressCmd(m.pasteProgressCh)
	case pasteFinishedMsg:
		return m, m.finishPaste(msg)
	case deleteProgressMsg:
		if m.deletion == nil || m.deletion.phase != deletePhaseMoving {
			return m, nil
		}
		m.deletion.completedBytes = msg.completedBytes
		m.deletion.totalBytes = msg.totalBytes
		m.deletion.completedItems = msg.completedItems
		m.deletion.totalItems = msg.totalItems
		m.deletion.currentPath = msg.currentPath
		m.deletion.completedSources = msg.completedSources
		m.deletion.totalSources = msg.totalSources
		return m, waitForDeleteProgressCmd(m.deleteProgress)
	case deleteFinishedMsg:
		return m, m.finishDelete(msg)
	case openFileResult:
		if msg.err != nil {
			m.setError(msg.err)
		} else {
			m.clearStatus()
		}
		return m, nil
	}
	if m.rename != nil {
		var cmd tea.Cmd
		m.rename.input, cmd = m.rename.input.Update(msg)
		return m, cmd
	}
	if m.creation != nil {
		var cmd tea.Cmd
		m.creation.input, cmd = m.creation.input.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshPreview()
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.paste != nil {
		switch msg.String() {
		case "esc", "ctrl+c":
			if !m.paste.cancelling {
				m.paste.cancelling = true
				m.paste.cancel()
			}
		}
		return m, nil
	}
	if m.deletion != nil {
		if m.deletion.phase == deletePhaseConfirming {
			if msg.String() == "d" {
				return m, m.confirmDelete()
			}
			m.deletion = nil
			m.deleteProgress = nil
		}
		return m, nil
	}
	if m.rename != nil {
		switch msg.String() {
		case "esc":
			m.cancelRename()
			return m, nil
		case "enter":
			m.confirmRename()
			return m, nil
		default:
			var cmd tea.Cmd
			m.rename.input, cmd = m.rename.input.Update(msg)
			m.rename.err = nil
			return m, cmd
		}
	}
	if m.creation != nil {
		switch msg.String() {
		case "esc":
			m.cancelCreation()
			return m, nil
		case "enter":
			m.confirmCreation()
			return m, nil
		default:
			var cmd tea.Cmd
			m.creation.input, cmd = m.creation.input.Update(msg)
			m.creation.err = nil
			return m, cmd
		}
	}

	if m.showHelp {
		switch msg.String() {
		case "ctrl+h", "esc":
			m.showHelp = false
		}
		return m, nil
	}

	if msg.String() == "ctrl+h" {
		m.jumpMulti = 0
		m.showHelp = true
		return m, nil
	}

	if msg.String() == "tab" {
		m.jumpMulti = 0
		m.toggleHidden()
		return m, nil
	}

	if m.searchMode {
		switch msg.String() {
		case "esc":
			m.cancelSearch()
			m.loadDir(m.cwd)
			return m, nil
		case "enter":
			cmd := m.handleSelect()
			return m, cmd
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		default:
			cmd := m.handleSearchInput(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case "/":
		m.initSearch(false)
		return m, nil
	case "?":
		m.initSearch(true)
		return m, nil
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if msg.String() == "0" && m.jumpMulti == 0 {
			m.setItem(0)
			return m, nil
		}
		m.jumpMulti = m.jumpMulti*10 + int(msg.String()[0]-'0')
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h":
		m.jumpMulti = 0
		m.loadPrev()
		return m, nil
	case "l":
		m.jumpMulti = 0
		return m, m.handleSelect()
	case "o":
		m.jumpMulti = 0
		return m, m.handleOpen()
	case "enter":
		return m, m.handleEnter()
	case "v":
		m.jumpMulti = 0
		m.startSelectionMode()
		return m, nil
	case "esc":
		m.jumpMulti = 0
		m.stopSelectionMode()
		return m, nil
	case "k":
		steps := 1
		if m.jumpMulti > 0 {
			steps = m.jumpMulti
			m.jumpMulti = 0
		}
		target := m.list.Index() - steps
		target = max(0, target)
		m.setItem(target)
		return m, nil
	case "j":
		steps := 1
		if m.jumpMulti > 0 {
			steps = m.jumpMulti
			m.jumpMulti = 0
		}
		target := m.list.Index() + steps
		target = min(len(m.list.Items())-1, target)
		m.setItem(target)
		return m, nil
	case "y":
		m.handleCopy()
		return m, nil
	case "p":
		return m, m.handlePaste()
	case "d":
		m.handleDelete()
		return m, nil
	case "r":
		if m.selectionMode {
			return m, nil
		}
		return m, m.handleRename()
	case "n":
		m.jumpMulti = 0
		return m, m.handleCreate()
	default:
		m.jumpMulti = 0
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.updateSelectionRange()
	return m, cmd
}

func (m *Model) finishDelete(message deleteFinishedMsg) tea.Cmd {
	state := m.deletion
	if state == nil {
		return nil
	}
	m.deletion = nil
	m.deleteProgress = nil

	deletedSources := state.sourcesOrSource()
	if message.err != nil {
		deletedSources = deletedSources[:min(len(deletedSources), len(message.results))]
	}
	m.pruneCopiedPaths(deletedSources)
	if err := m.loadDir(m.cwd); err != nil {
		m.setError(err)
		return nil
	}
	m.setItem(state.selectionIndex)
	m.previewPath = ""
	m.refreshPreview()
	if message.err != nil {
		m.setError(message.err)
		return nil
	}
	if len(state.sourcesOrSource()) == 1 {
		m.setStatus(fmt.Sprintf("Moved %s to %s", filepath.Base(state.source), message.result.Target), false)
	} else {
		m.setStatus(fmt.Sprintf("Moved %d items to %s", len(state.sourcesOrSource()), state.trashDirectory), false)
	}
	return nil
}

func (m *Model) pruneCopiedPaths(deletedSources []string) {
	paths := m.copiedPaths()
	kept := paths[:0]
	for _, copiedPath := range paths {
		deleted := false
		for _, source := range deletedSources {
			if pathContainsSelection(source, copiedPath) {
				deleted = true
				break
			}
		}
		if !deleted {
			_, err := os.Lstat(copiedPath)
			deleted = errors.Is(err, os.ErrNotExist)
		}
		if !deleted {
			kept = append(kept, copiedPath)
		}
	}
	m.pathsToCopy = append([]string(nil), kept...)
	m.pathToCopy = ""
	if len(m.pathsToCopy) > 0 {
		m.pathToCopy = m.pathsToCopy[0]
	}
}

func pathContainsSelection(parent, child string) bool {
	if child == "" {
		return false
	}
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
}

func (m *Model) finishPaste(message pasteFinishedMsg) tea.Cmd {
	state := m.paste
	if state == nil {
		return nil
	}
	if state.cancel != nil {
		state.cancel()
	}
	m.paste = nil
	m.pasteProgressCh = nil

	targets := message.targets
	if len(targets) == 0 && message.result.Target != "" {
		targets = []string{message.result.Target}
	}
	if len(targets) > 0 {
		if err := m.loadDir(m.cwd); err != nil {
			m.setError(err)
			return nil
		}
		m.selectPath(targets[0])
		m.previewPath = ""
		m.refreshPreview()
	}
	if message.err != nil {
		if state.cancelling || errors.Is(message.err, context.Canceled) {
			m.setStatus("Paste cancelled", false)
		} else {
			m.setError(message.err)
		}
		return nil
	}

	if len(targets) == 0 {
		return nil
	}
	if len(state.sources) > 1 {
		m.setStatus(fmt.Sprintf("Pasted %d items", len(targets)), false)
	} else {
		m.setStatus(fmt.Sprintf("Pasted %s", filepath.Base(targets[0])), false)
	}
	return nil
}
