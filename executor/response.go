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
	if runErr != nil && isExitError(runErr) && strings.TrimSpace(stderr) == "" {
		stderr = runErr.Error()
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

func bytesToKB(bytes uint64) uint {
	return uint(math.Ceil(float64(bytes) / 1024.0))
}
