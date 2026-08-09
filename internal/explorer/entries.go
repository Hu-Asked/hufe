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

func ReadEntriesRecursive(dir string) ([]Entry, error) {
	var dirs []Entry
	var files []Entry

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we cannot access
		}
		if path == dir {
			return nil
		}
		
		rel, errRel := filepath.Rel(dir, path)
		if errRel != nil {
			rel = filepath.Base(path)
		}

		entry := Entry{
			Name:  rel,
			Path:  path,
			IsDir: d.IsDir(),
		}

		if entry.IsDir {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sortEntries := func(entries []Entry) {
		sort.Slice(entries, func(i, j int) bool {
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})
	}

	sortEntries(dirs)
	sortEntries(files)

	var entries []Entry
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
