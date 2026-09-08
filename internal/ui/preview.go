package ui

import "hufe/internal/explorer"

func (m *Model) refreshPreview() {
	selected := m.list.SelectedItem()
	if selected == nil {
		m.clearPreview()
		return
	}

	selectedItem, ok := selected.(item)
	if !ok {
		m.clearPreview()
		return
	}

	entry := selectedItem.entry
	if m.previewPath == entry.Path {
		return
	}

	m.previewPath = entry.Path
	m.previewName = entry.Name
	m.previewIsDir = entry.IsDir
	m.previewEntries = nil
	m.previewErr = nil

	if !entry.IsDir {
		return
	}

	entries, err := explorer.ReadEntries(entry.Path)
	if err != nil {
		m.previewErr = err
		return
	}

	for _, child := range entries {
		if child.Name != ".." {
			m.previewEntries = append(m.previewEntries, child)
		}
	}
}

func (m *Model) clearPreview() {
	m.previewPath = ""
	m.previewName = ""
	m.previewIsDir = false
	m.previewEntries = nil
	m.previewErr = nil
}
