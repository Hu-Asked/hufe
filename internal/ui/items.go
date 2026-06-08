package ui

import (
	"os"

	"github.com/charmbracelet/bubbles/list"
	"hufe/internal/explorer"
)

type item struct {
	entry explorer.Entry
}

func (i item) Title() string {
	if i.entry.IsDir && i.entry.Name != ".." {
		return i.entry.Name + string(os.PathSeparator)
	}
	return i.entry.Name
}

func (i item) Description() string {
	return ""
}

func (i item) FilterValue() string {
	if i.entry.IsDir && i.entry.Name != ".." {
		return i.entry.Name + string(os.PathSeparator)
	}
	return i.entry.Name
}

func itemsFromEntries(entries []explorer.Entry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, item{entry: entry})
	}
	return items
}
