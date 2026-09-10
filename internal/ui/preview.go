package ui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"hufe/internal/explorer"
)

const maxPreviewBytes = 256 * 1024

var errUnsupportedPreview = errors.New("file is not previewable text")

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
	m.previewFileLines = nil
	m.previewTruncated = false
	m.previewUnsupported = false
	m.previewErr = nil

	if !entry.IsDir {
		lines, truncated, err := readFilePreview(entry.Path)
		if errors.Is(err, errUnsupportedPreview) {
			m.previewUnsupported = true
			return
		}
		if err != nil {
			m.previewErr = err
			return
		}
		m.previewFileLines = lines
		m.previewTruncated = truncated
		return
	}

	entries, err := explorer.ReadEntriesWithHidden(entry.Path, m.showHidden)
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
	m.previewFileLines = nil
	m.previewTruncated = false
	m.previewUnsupported = false
	m.previewErr = nil
}

func readFilePreview(path string) ([]string, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, errUnsupportedPreview
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxPreviewBytes+utf8.UTFMax))
	if err != nil {
		return nil, false, err
	}

	truncated := len(data) > maxPreviewBytes
	previewLength := min(len(data), maxPreviewBytes)
	validLength, ok := previewLength, utf8.Valid(data[:previewLength])
	if truncated {
		validLength, ok = validUTF8PrefixLength(data, previewLength)
	}
	if !ok || bytes.IndexByte(data[:validLength], 0) >= 0 {
		return nil, false, errUnsupportedPreview
	}

	text := sanitizePreviewText(string(data[:validLength]))
	if text == "" {
		return nil, truncated, nil
	}

	return strings.Split(text, "\n"), truncated, nil
}

func validUTF8PrefixLength(data []byte, length int) (int, bool) {
	for removed := 0; removed < utf8.UTFMax && length >= 0; removed++ {
		if utf8.Valid(data[:length]) {
			return length, true
		}
		length--
	}
	return 0, false
}

func sanitizePreviewText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return utf8.RuneError
	}, text)
}
