package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type sandboxWorkspace struct {
	dir        string
	sourcePath string
	binaryPath string
}

func prepareWorkspace(code string) (sandboxWorkspace, error) {
	dir, err := os.MkdirTemp("", "codesandbox-c-*")
	if err != nil {
		return sandboxWorkspace{}, fmt.Errorf("failed to create sandbox dir: %w", err)
	}
	ws := sandboxWorkspace{dir: dir}
	ws.sourcePath = filepath.Join(dir, "main.c")
	ws.binaryPath = filepath.Join(dir, "program")
	if err := os.WriteFile(ws.sourcePath, []byte(code), 0o600); err != nil {
		return sandboxWorkspace{}, fmt.Errorf("failed to write source: %w", err)
	}
	return ws, nil
}

func compileWorkspace(ws sandboxWorkspace) (string, bool) {
	cmd := exec.Command("gcc", "-O2", "-pipe", "-std=c11", "-static", "-s", "-o", ws.binaryPath, ws.sourcePath)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String(), false
	}
	return "", true
}
