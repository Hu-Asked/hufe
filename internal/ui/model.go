package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"hufe/internal/explorer"
)

type Model struct {
	list          list.Model
	cwd           string
	status        string
	statusIsError bool
	exitDir       string
	boxWidth	  int
	jumpMulti	  int
	pathToCopy    string
}

func NewModel(startDir string) (*Model, error) {
	entries, err := explorer.ReadEntries(startDir)
	if err != nil {
		return nil, err
	}

	l := newList(itemsFromEntries(entries))
	m := &Model{
		list: l,
		cwd:  startDir,
	}
	m.updateTitle()

	return m, nil
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
