package opener

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

const textSampleSize = 8 * 1024

type LaunchMode uint8

const (
	Terminal LaunchMode = iota
	Detached
)

var browserExtensions = map[string]struct{}{
	".htm":   {},
	".html":  {},
	".mht":   {},
	".mhtml": {},
	".pdf":   {},
	".xhtml": {},
}

func Command(path string) (*exec.Cmd, LaunchMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, Terminal, fmt.Errorf("open %s: %w", filepath.Base(path), err)
	}
	if !info.Mode().IsRegular() {
		return nil, Terminal, fmt.Errorf("unsupported file type: %s", filepath.Base(path))
	}

	if _, ok := browserExtensions[strings.ToLower(filepath.Ext(path))]; ok {
		name, args, err := browserCommand()
		if err != nil {
			return nil, Detached, err
		}
		return buildCommand(name, args, path, Detached), Detached, nil
	}

	isText, err := textFile(path)
	if err != nil {
		return nil, Terminal, fmt.Errorf("inspect %s: %w", filepath.Base(path), err)
	}
	if !isText {
		return nil, Terminal, fmt.Errorf("unsupported file type: %s", filepath.Base(path))
	}

	name, args, err := editorCommand()
	if err != nil {
		return nil, Terminal, err
	}
	return buildCommand(name, args, path, Terminal), Terminal, nil
}

func textFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, textSampleSize))
	if err != nil {
		return false, err
	}
	return bytes.IndexByte(data, 0) < 0 && utf8.Valid(data), nil
}

func editorCommand() (string, []string, error) {
	for _, environmentVariable := range []string{"VISUAL", "EDITOR"} {
		if value := strings.TrimSpace(os.Getenv(environmentVariable)); value != "" {
			name, args, err := resolveCommand(value)
			if err != nil {
				return "", nil, fmt.Errorf("%s: %w", environmentVariable, err)
			}
			return name, args, nil
		}
	}
	return firstAvailable("editor", []commandCandidate{{name: "nvim"}, {name: "vi"}})
}

func browserCommand() (string, []string, error) {
	if value := strings.TrimSpace(os.Getenv("BROWSER")); value != "" {
		name, args, err := resolveCommand(value)
		if err != nil {
			return "", nil, fmt.Errorf("BROWSER: %w", err)
		}
		return name, args, nil
	}

	candidates := []commandCandidate{
		{name: "sensible-browser"},
		{name: "x-www-browser"},
		{name: "firefox"},
		{name: "google-chrome"},
		{name: "chromium"},
		{name: "chromium-browser"},
	}
	if runtime.GOOS == "darwin" {
		candidates = append([]commandCandidate{{name: "open", args: []string{"-a", "Safari"}}}, candidates...)
	} else if runtime.GOOS == "windows" {
		candidates = append([]commandCandidate{{name: "msedge"}, {name: "chrome"}, {name: "firefox"}}, candidates...)
	}
	return firstAvailable("browser", candidates)
}

type commandCandidate struct {
	name string
	args []string
}

func firstAvailable(kind string, candidates []commandCandidate) (string, []string, error) {
	for _, candidate := range candidates {
		name, err := exec.LookPath(candidate.name)
		if err == nil {
			return name, append([]string(nil), candidate.args...), nil
		}
	}
	return "", nil, fmt.Errorf("no %s command found", kind)
}

func resolveCommand(command string) (string, []string, error) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", nil, errors.New("open command is empty")
	}
	name, err := exec.LookPath(fields[0])
	if err != nil {
		return "", nil, fmt.Errorf("command %q not found", fields[0])
	}
	return name, fields[1:], nil
}

func buildCommand(name string, args []string, path string, mode LaunchMode) *exec.Cmd {
	args = append(append([]string(nil), args...), path)
	cmd := exec.Command(name, args...)
	if mode == Terminal {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}
