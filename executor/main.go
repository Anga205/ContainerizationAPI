package executor

import (
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func Execute(req models.Request) (models.Response, error) {
	if req.Language != "c" {
		return models.Response{}, fmt.Errorf("unsupported language: %s", req.Language)
	}
	return executeCProgram(req)
}

func executeCProgram(req models.Request) (models.Response, error) {
	ws, err := prepareWorkspace(req.Code)
	if err != nil {
		return models.Response{}, err
	}
	defer func() { _ = osRemoveAll(ws.dir) }()

	compileErr, ok := compileWorkspace(ws)
	if !ok {
		return compileFailureResponse(compileErr), nil
	}

	cg, err := resourcemanager.NewCgroupV2(req.MemoryLimit)
	if err != nil {
		return models.Response{}, err
	}
	defer cg.Close()

	return executeSandboxedBinary(req, ws.dir, cg)
}

func executeSandboxedBinary(req models.Request, sandboxDir string, cg *resourcemanager.CgroupHandle) (models.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.Timeout)
	defer cancel()

	cgroupFD, err := cg.OpenFD()
	if err != nil {
		return models.Response{}, fmt.Errorf("failed to open cgroup fd: %w", err)
	}
	defer syscall.Close(cgroupFD)

	cmd, stdoutBuf, stderrBuf := buildCommandBuffers(req.Stdin, sandboxDir, cgroupFD)
	runErr := startSandbox(cmd)
	if runErr != nil {
		return models.Response{}, runErr
	}

	start := time.Now()
	sampler := startMemorySampler(cg)
	waitErr, timedOut := waitForSandbox(ctx, cmd, cg)
	observedPeak := sampler.stop()

	return finalizeResponse(cmd, stdoutBuf.String(), stderrBuf.String(), time.Since(start), observedPeak, timedOut, waitErr, cg)
}

func buildCommandBuffers(stdin, sandboxDir string, cgroupFD int) (*exec.Cmd, *strings.Builder, *strings.Builder) {
	cmd := buildSandboxCommand(stdin, sandboxDir, cgroupFD)
	stdoutBuf := &strings.Builder{}
	stderrBuf := &strings.Builder{}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	return cmd, stdoutBuf, stderrBuf
}

func startSandbox(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start sandboxed process: %w", err)
	}
	return nil
}

func compileFailureResponse(stderr string) models.Response {
	return models.Response{Stdout: "", Stderr: stderr, ExecutionTime: 0, MemoryUsed: 0}
}

func osRemoveAll(path string) error {
	return os.RemoveAll(path)
}
