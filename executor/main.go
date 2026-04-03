package executor

import (
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func Execute(req models.Request) (models.Response, error) {
	ws, err := prepareWorkspace(req.Language, req.Code)
	if err != nil {
		return models.Response{}, err
	}
	defer cleanupWorkspace(&ws)

	compileErr, ok := compileWorkspace(ws)
	if !ok {
		return compileFailureResponse(compileErr), nil
	}
	if err := ensureSandboxInitBinary(&ws); err != nil {
		return models.Response{}, err
	}

	cg, err := resourcemanager.NewCgroupV2(req.MemoryLimit)
	if err != nil {
		return models.Response{}, err
	}
	defer cg.Close()

	if err := prepareRuntimeMounts(&ws); err != nil {
		return models.Response{}, err
	}

	return executeSandboxedBinary(req, ws, cg)
}

func executeSandboxedBinary(req models.Request, ws sandboxWorkspace, cg *resourcemanager.CgroupHandle) (models.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.Timeout)
	defer cancel()

	cgroupFD, err := cg.OpenFD()
	if err != nil {
		return models.Response{}, fmt.Errorf("failed to open cgroup fd: %w", err)
	}
	defer syscall.Close(cgroupFD)

	cmd, stdoutBuf, stderrBuf := buildCommandBuffers(req.Stdin, ws, cgroupFD)
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

func buildCommandBuffers(stdin string, ws sandboxWorkspace, cgroupFD int) (*exec.Cmd, *limitedBuffer, *limitedBuffer) {
	cmd := buildSandboxCommand(stdin, ws, cgroupFD)
	stdoutBuf := newLimitedBuffer(1 << 20)
	stderrBuf := newLimitedBuffer(1 << 20)
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

func cleanupWorkspace(ws *sandboxWorkspace) {
	cleanupRuntimeMounts(ws)
	_ = osRemoveAll(ws.dir)
}
