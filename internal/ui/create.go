package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleCreate() tea.Cmd {
	input := textinput.New()
	styleTextInput(&input)
	input.Prompt = ""
	input.CharLimit = 255
	input.Width = max(10, min(50, m.windowWidth-14))
	m.creation = &creationState{input: input}
	m.clearStatus()
	return m.creation.input.Focus()
}

func (m *Model) cancelCreation() {
	if m.creation == nil {
		return
	}
	m.creation.input.Blur()
	m.creation = nil
}

func (m *Model) confirmCreation() {
	if m.creation == nil {
		return
	}

	entered := m.creation.input.Value()
	isDirectory := strings.HasSuffix(entered, "/")
	name := entered
	if isDirectory {
		name = strings.TrimSuffix(name, "/")
	}
	if err := validateRenameName(name); err != nil {
		m.creation.err = err
		return
	}
	if strings.ContainsRune(name, '/') {
		m.creation.err = errors.New("name cannot contain a path separator")
		return
	}

	target := filepath.Join(m.cwd, name)
	var err error
	if isDirectory {
		err = os.Mkdir(target, 0o777)
	} else {
		var file *os.File
		file, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if err == nil {
			err = file.Close()
		}
	}
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			m.creation.err = fmt.Errorf("%q already exists", name)
		} else {
			m.creation.err = err
		}
		return
	}

	m.cancelCreation()
	if err := m.loadDir(m.cwd); err != nil {
		m.setError(err)
		return
	}
	m.selectPath(target)
	kind := "file"
	if isDirectory {
		kind = "directory"
	}
	m.setStatus(fmt.Sprintf("Created %s %s", kind, entered), false)
}
