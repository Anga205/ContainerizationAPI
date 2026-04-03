package executor

import (
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func finalizeResponse(
	cmd *exec.Cmd,
	stdout, stderr string,
	duration time.Duration,
	observedPeak uint64,
	timedOut bool,
	runErr error,
	cg *resourcemanager.CgroupHandle,
) (models.Response, error) {
	peakKB := determinePeakKB(cmd, cg, observedPeak)
	limitHit := memoryLimitTriggered(cg)
	stderr = appendMemoryLimitError(stderr, limitHit)
	if timedOut && !limitHit {
		stderr = appendTimeoutError(stderr)
	}
	if runErr != nil && isExitError(runErr) {
		stderr = appendExitDiagnostic(stderr, runErr)
	}
	if runErr != nil && !isExitError(runErr) {
		return models.Response{}, fmt.Errorf("execution failed: %w", runErr)
	}
	return models.Response{Stdout: stdout, Stderr: stderr, ExecutionTime: duration, MemoryUsed: peakKB}, nil
}

func determinePeakKB(cmd *exec.Cmd, cg *resourcemanager.CgroupHandle, observed uint64) uint {
	peak := cg.ReadMemoryPeakBytes()
	if observed > peak {
		peak = observed
	}
	peakKB := bytesToKB(peak)
	if rss := maxRSSKB(cmd); rss > peakKB {
		return rss
	}
	return peakKB
}

func maxRSSKB(cmd *exec.Cmd) uint {
	state := cmd.ProcessState
	if state == nil {
		return 0
	}
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	return uint(usage.Maxrss)
}

func memoryLimitTriggered(cg *resourcemanager.CgroupHandle) bool {
	events := cg.ReadMemoryEvents()
	return events.OOM > 0 || events.OOMKill > 0 || events.Max > 0
}

func appendMemoryLimitError(stderr string, triggered bool) string {
	if !triggered {
		return stderr
	}
	return appendErrorLine(stderr, "Memory limit exceeded")
}

func appendTimeoutError(stderr string) string {
	return appendErrorLine(stderr, "Execution Timed Out")
}

func appendErrorLine(stderr, message string) string {
	if stderr == "" {
		return message
	}
	if strings.HasSuffix(stderr, "\n") {
		return stderr + message
	}
	return stderr + "\n" + message
}

func isExitError(err error) bool {
	_, ok := err.(*exec.ExitError)
	return ok
}

func appendExitDiagnostic(stderr string, err error) string {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return appendErrorLine(stderr, err.Error())
	}

	detail := formatExitDiagnostic(exitErr)
	if detail == "" {
		detail = exitErr.Error()
	}
	return appendErrorLine(stderr, detail)
}

func formatExitDiagnostic(exitErr *exec.ExitError) string {
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return exitErr.Error()
	}

	if status.Signaled() {
		sig := status.Signal()
		base := fmt.Sprintf("Process terminated by signal %s (%d): %s", signalName(sig), int(sig), sig.String())
		if sig == syscall.SIGSYS {
			return base + "; likely blocked by sandbox seccomp policy"
		}
		return base
	}

	if status.Exited() {
		code := status.ExitStatus()
		if code != 0 {
			return fmt.Sprintf("Process exited with non-zero status %d", code)
		}
	}

	return exitErr.Error()
}

func signalName(sig syscall.Signal) string {
	switch sig {
	case syscall.SIGKILL:
		return "SIGKILL"
	case syscall.SIGSEGV:
		return "SIGSEGV"
	case syscall.SIGABRT:
		return "SIGABRT"
	case syscall.SIGSYS:
		return "SIGSYS"
	case syscall.SIGXCPU:
		return "SIGXCPU"
	case syscall.SIGTERM:
		return "SIGTERM"
	default:
		return fmt.Sprintf("SIG%d", int(sig))
	}
}

func bytesToKB(bytes uint64) uint {
	return uint(math.Ceil(float64(bytes) / 1024.0))
}
