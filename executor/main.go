package executor

import (
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type executionTrace struct {
	steps []string
}

func newExecutionTrace() *executionTrace {
	return &executionTrace{steps: make([]string, 0, 16)}
}

func (t *executionTrace) ok(step string) {
	t.steps = append(t.steps, fmt.Sprintf("[OK] %s", step))
}

func (t *executionTrace) info(step, detail string) {
	t.steps = append(t.steps, fmt.Sprintf("[INFO] %s: %s", step, detail))
}

func (t *executionTrace) format() string {
	if len(t.steps) == 0 {
		return "(no trace entries)"
	}
	return strings.Join(t.steps, "\n")
}

func (t *executionTrace) fail(step string, err error) error {
	t.steps = append(t.steps, fmt.Sprintf("[FAILED] %s: %v", step, err))
	return fmt.Errorf("sandbox execution failed at step %q: %w\nexecution trace:\n%s", step, err, t.format())
}

func Execute(req models.Request) (models.Response, error) {
	trace := newExecutionTrace()
	trace.ok("request received by executor")

	ws, err := prepareWorkspace(req.Language, req.Code)
	if err != nil {
		return models.Response{}, trace.fail("prepare workspace", err)
	}
	trace.ok("prepared workspace")
	defer cleanupWorkspace(&ws)

	compileErr, ok := compileWorkspace(ws)
	if !ok {
		trace.info("compile workspace", "user code compilation failed; returning compiler stderr")
		return compileFailureResponse(compileErr), nil
	}
	trace.ok("compiled workspace")

	if err := ensureSandboxInitBinary(&ws); err != nil {
		return models.Response{}, trace.fail("ensure sandbox init binary", err)
	}
	trace.ok("installed sandbox init binary")

	cg, err := resourcemanager.NewCgroupV2(req.MemoryLimit)
	if err != nil {
		return models.Response{}, trace.fail("create cgroup v2", err)
	}
	trace.ok("created cgroup v2")
	defer cg.Close()

	if err := prepareRuntimeMounts(&ws); err != nil {
		return models.Response{}, trace.fail("prepare runtime mounts", err)
	}
	trace.ok("prepared runtime mounts")

	return executeSandboxedBinary(req, ws, cg, trace)
}

func executeSandboxedBinary(req models.Request, ws sandboxWorkspace, cg *resourcemanager.CgroupHandle, trace *executionTrace) (models.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.Timeout)
	defer cancel()
	trace.ok("created execution timeout context")

	cgroupFD, err := cg.OpenFD()
	if err != nil {
		return models.Response{}, trace.fail("open cgroup fd", err)
	}
	trace.ok("opened cgroup fd")
	defer syscall.Close(cgroupFD)

	cmd, stdoutBuf, stderrBuf, runErr := startSandboxWithFallback(req.Stdin, ws, cgroupFD, cg, trace)
	if runErr != nil {
		return models.Response{}, trace.fail("start sandbox process", runErr)
	}
	trace.ok("started sandbox process")

	start := time.Now()
	sampler := startMemorySampler(cg)
	trace.ok("started memory sampler")
	waitErr, timedOut := waitForSandbox(ctx, cmd, cg)
	observedPeak := sampler.stop()
	trace.ok("waited for sandbox completion")

	return finalizeResponse(cmd, stdoutBuf.String(), stderrBuf.String(), time.Since(start), observedPeak, timedOut, waitErr, cg)
}

func buildCommandBuffers(stdin string, ws sandboxWorkspace, cgroupFD int) (*exec.Cmd, *limitedBuffer, *limitedBuffer) {
	cmd := buildSandboxCommand(stdin, ws, cgroupFD)
	return buildCommandBuffersForCommand(cmd)
}

func buildCommandBuffersWithoutCgroupFD(stdin string, ws sandboxWorkspace) (*exec.Cmd, *limitedBuffer, *limitedBuffer) {
	cmd := buildSandboxCommandWithoutCgroupFD(stdin, ws)
	return buildCommandBuffersForCommand(cmd)
}

func buildCommandBuffersForCommand(cmd *exec.Cmd) (*exec.Cmd, *limitedBuffer, *limitedBuffer) {
	stdoutBuf := newLimitedBuffer(1 << 20)
	stderrBuf := newLimitedBuffer(1 << 20)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	return cmd, stdoutBuf, stderrBuf
}

func startSandboxWithFallback(stdin string, ws sandboxWorkspace, cgroupFD int, cg *resourcemanager.CgroupHandle, trace *executionTrace) (*exec.Cmd, *limitedBuffer, *limitedBuffer, error) {
	cmd, stdoutBuf, stderrBuf := buildCommandBuffers(stdin, ws, cgroupFD)
	trace.ok("built sandbox command with cgroup fd")
	err := startSandbox(cmd)
	if err == nil {
		trace.ok("sandbox start succeeded with cgroup fd")
		return cmd, stdoutBuf, stderrBuf, nil
	}
	trace.info("sandbox start with cgroup fd failed", err.Error())
	if !shouldFallbackWithoutCgroupFD(err) {
		return nil, nil, nil, err
	}

	trace.info("falling back", "retrying sandbox start without cgroup fd")
	cmd, stdoutBuf, stderrBuf = buildCommandBuffersWithoutCgroupFD(stdin, ws)
	trace.ok("built sandbox command without cgroup fd")
	if err := startSandbox(cmd); err != nil {
		return nil, nil, nil, err
	}
	trace.ok("sandbox start succeeded without cgroup fd")
	if err := cg.AddProcess(cmd.Process.Pid); err != nil {
		_ = hardKillProcessGroup(cmd.Process.Pid)
		_ = cmd.Wait()
		return nil, nil, nil, fmt.Errorf("failed to attach process to cgroup: %w", err)
	}
	trace.ok("attached process to cgroup manually")

	return cmd, stdoutBuf, stderrBuf, nil
}

func shouldFallbackWithoutCgroupFD(err error) bool {
	return errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.ENOSYS) ||
		errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.EPERM)
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
