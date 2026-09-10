package explorer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Entry struct {
	Name  string
	Path  string
	IsDir bool
}

func ReadEntries(dir string) ([]Entry, error) {
	return ReadEntriesWithHidden(dir, true)
}

func ReadEntriesWithHidden(dir string, showHidden bool) ([]Entry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []Entry
	var files []Entry

	for _, item := range items {
		name := item.Name()
		if !showHidden && isHidden(name) {
			continue
		}
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

	sortEntries(dirs, true)
	sortEntries(files, false)

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
	return ReadEntriesRecursiveWithHidden(dir, true)
}

func ReadEntriesRecursiveWithHidden(dir string, showHidden bool) ([]Entry, error) {
	var dirs []Entry
	var files []Entry

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we cannot access
		}
		if path == dir {
			return nil
		}
		if !showHidden && isHidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
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

	sortEntries(dirs, true)
	sortEntries(files, false)

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

func sortEntries(entries []Entry, capitalizedFirst bool) {
	sort.Slice(entries, func(i, j int) bool {
		if capitalizedFirst {
			iCapitalized := startsWithCapital(entries[i].Name)
			jCapitalized := startsWithCapital(entries[j].Name)
			if iCapitalized != jCapitalized {
				return iCapitalized
			}
		}

		iName := strings.ToLower(entries[i].Name)
		jName := strings.ToLower(entries[j].Name)
		if iName != jName {
			return iName < jName
		}
		return entries[i].Name < entries[j].Name
	})
}

func startsWithCapital(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(first)
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".") && name != ".."
}
