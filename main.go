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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	m, err := ui.NewModel(absPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if uiModel, ok := finalModel.(*ui.Model); ok {
		if dir := uiModel.ExitDir(); dir != "" {
			fmt.Fprint(os.Stdout, dir)
		}
	}
}
