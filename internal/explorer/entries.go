package explorer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Name  string
	Path  string
	IsDir bool
}

func ReadEntries(dir string) ([]Entry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []Entry
	var files []Entry

	for _, item := range items {
		name := item.Name()
		entry := Entry{
			Name:  name,
			Path:  filepath.Join(dir, name),
			IsDir: item.IsDir(),
		}
		if entry.IsDir {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	sortEntries := func(entries []Entry) {
		sort.Slice(entries, func(i, j int) bool {
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})
	}

	sortEntries(dirs)
	sortEntries(files)

	entries := make([]Entry, 0, len(dirs)+len(files)+1)
	parent := filepath.Dir(dir)
	if parent != dir {
		entries = append(entries, Entry{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}

	entries = append(entries, dirs...)
	entries = append(entries, files...)

	return entries, nil
}
