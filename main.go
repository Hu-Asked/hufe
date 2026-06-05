package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"hufe/internal/ui"
)

func main() {
	startPath := "."
	if len(os.Args) > 1 {
		startPath = os.Args[1]
	}

	absPath, err := filepath.Abs(startPath)
	if err != nil {
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	m, err := ui.NewModel(absPath)
	if err != nil {
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if dir := m.ExitDir(); dir != "" {
		if exportFile := os.Getenv("HUFE_OUTPUT"); exportFile != "" {
			_ = os.WriteFile(exportFile, []byte(dir), 0644)
		} else {
			fmt.Println(dir)
		}
	}
}
