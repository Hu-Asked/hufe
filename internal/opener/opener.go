package opener

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

func Command(path string) (*exec.Cmd, error) {
	cmdName, cmdArgs, err := commandForOpen()
	if err != nil {
		return nil, err
	}

	cmdArgs = append(cmdArgs, path)
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func commandForOpen() (string, []string, error) {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor != "" {
		return splitCommand(editor)
	}

	pager := strings.TrimSpace(os.Getenv("PAGER"))
	if pager != "" {
		return splitCommand(pager)
	}

	return "less", nil, nil
}

func splitCommand(command string) (string, []string, error) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", nil, errors.New("open command is empty")
	}
	return fields[0], fields[1:], nil
}
