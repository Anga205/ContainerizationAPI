package executor

import (
	"CodeSandboxAPI/models"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var cgroupWriteMu sync.Mutex

func Execute(req models.Request) (models.Response, error) {
	if req.Language != "c" {
		return models.Response{}, fmt.Errorf("unsupported language: %s", req.Language)
	}

	tmpDir, err := os.MkdirTemp("", "codesandbox-c-*")
	if err != nil {
		return models.Response{}, fmt.Errorf("failed to create sandbox dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	sourcePath := filepath.Join(tmpDir, "main.c")
	binaryPath := filepath.Join(tmpDir, "program")
	if err := os.WriteFile(sourcePath, []byte(req.Code), 0o600); err != nil {
		return models.Response{}, fmt.Errorf("failed to write source: %w", err)
	}

	compileCmd := exec.Command("gcc", "-O0", "-pipe", "-std=c11", "-static", "-s", "-o", binaryPath, sourcePath)
	var compileStderr strings.Builder
	compileCmd.Stderr = &compileStderr
	if err := compileCmd.Run(); err != nil {
		return models.Response{
			Stdout:        "",
			Stderr:        compileStderr.String(),
			ExecutionTime: 0,
			MemoryUsed:    0,
		}, nil
	}

	cg, err := createCgroupV2(req.MemoryLimit)
	if err != nil {
		return models.Response{}, err
	}
	defer cg.Close()

	ctx, cancel := context.WithTimeout(context.Background(), req.Timeout)
	defer cancel()

	runCmd := exec.Command("/program")
	runCmd.Dir = "/"
	runCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
		Chroot:     tmpDir,
		Setpgid:    true,
	}
	if len(req.Stdin) > 0 {
		runCmd.Stdin = strings.NewReader(strings.Join(req.Stdin, "\n"))
	}

	var stdoutBuf strings.Builder
	var stderrBuf strings.Builder
	runCmd.Stdout = &stdoutBuf
	runCmd.Stderr = &stderrBuf

	start := time.Now()
	if err := runCmd.Start(); err != nil {
		return models.Response{}, fmt.Errorf("failed to start sandboxed process: %w", err)
	}

	if err := applyProcessRlimits(runCmd.Process.Pid, req.MemoryLimit); err != nil {
		_ = hardKillProcessGroup(runCmd.Process.Pid)
		_, _ = runCmd.Process.Wait()
		return models.Response{}, fmt.Errorf("failed to apply rlimits: %w", err)
	}

	if err := cg.AddPID(runCmd.Process.Pid); err != nil {
		_ = hardKillProcessGroup(runCmd.Process.Pid)
		_, _ = runCmd.Process.Wait()
		return models.Response{}, fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	// Track observed working-set usage while the process runs.
	var observedPeakBytes uint64
	sampleCtx, stopSampling := context.WithCancel(context.Background())
	var samplerWG sync.WaitGroup
	samplerWG.Add(1)
	go func() {
		defer samplerWG.Done()
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-sampleCtx.Done():
				return
			case <-ticker.C:
				currentBytes, readErr := cg.ReadMemoryCurrentBytes()
				if readErr != nil {
					continue
				}
				for {
					prev := atomic.LoadUint64(&observedPeakBytes)
					if currentBytes <= prev {
						break
					}
					if atomic.CompareAndSwapUint64(&observedPeakBytes, prev, currentBytes) {
						break
					}
				}
			}
		}
	}()

	timedOut := false
	done := make(chan error, 1)
	go func() {
		done <- runCmd.Wait()
	}()

	select {
	case err = <-done:
		// Process finished by itself.
	case <-ctx.Done():
		timedOut = true
		_ = hardKillProcessGroup(runCmd.Process.Pid)
		_ = cg.KillAll()
		err = <-done
	}
	stopSampling()
	samplerWG.Wait()
	runDuration := time.Since(start)

	peakBytes := cg.ReadMemoryPeakBytes()
	if sampled := atomic.LoadUint64(&observedPeakBytes); sampled > peakBytes {
		peakBytes = sampled
	}
	peakKB := bytesToKB(peakBytes)

	if ps := runCmd.ProcessState; ps != nil {
		if usage, ok := ps.SysUsage().(*syscall.Rusage); ok {
			if rssKB := uint(usage.Maxrss); rssKB > peakKB {
				peakKB = rssKB
			}
		}
	}

	memoryEvents := cg.ReadMemoryEvents()
	memoryLimitTriggered := memoryEvents.OOM > 0 || memoryEvents.OOMKill > 0 || memoryEvents.Max > 0

	if timedOut {
		if stderrBuf.Len() > 0 && !strings.HasSuffix(stderrBuf.String(), "\n") {
			stderrBuf.WriteByte('\n')
		}
		stderrBuf.WriteString("Execution Timed Out")
		return models.Response{
			Stdout:        stdoutBuf.String(),
			Stderr:        stderrBuf.String(),
			ExecutionTime: runDuration,
			MemoryUsed:    peakKB,
		}, nil
	}

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return models.Response{}, fmt.Errorf("execution failed: %w", err)
		}
	}

	if memoryLimitTriggered {
		if stderrBuf.Len() > 0 && !strings.HasSuffix(stderrBuf.String(), "\n") {
			stderrBuf.WriteByte('\n')
		}
		stderrBuf.WriteString("Memory limit exceeded")
	}

	return models.Response{
		Stdout:        stdoutBuf.String(),
		Stderr:        stderrBuf.String(),
		ExecutionTime: runDuration,
		MemoryUsed:    peakKB,
	}, nil
}

type cgroupHandle struct {
	path string
}

type memoryEvents struct {
	Low     uint64
	High    uint64
	Max     uint64
	OOM     uint64
	OOMKill uint64
}

func createCgroupV2(memoryLimitKB uint) (*cgroupHandle, error) {
	base := "/sys/fs/cgroup"
	name := fmt.Sprintf("codesandbox-%d-%d", os.Getpid(), time.Now().UnixNano())
	path := filepath.Join(base, name)

	cgroupWriteMu.Lock()
	defer cgroupWriteMu.Unlock()

	if err := os.Mkdir(path, 0o755); err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("cannot create cgroup (need CAP_SYS_ADMIN/root): %w", err)
		}
		return nil, fmt.Errorf("failed to create cgroup: %w", err)
	}

	h := &cgroupHandle{path: path}
	if err := os.WriteFile(filepath.Join(path, "cpu.max"), []byte("100000 100000\n"), 0o644); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to set cpu.max: %w", err)
	}

	memLimitBytes := "max\n"
	if memoryLimitKB > 0 {
		memLimitBytes = fmt.Sprintf("%d\n", uint64(memoryLimitKB)*1024)
	}
	if err := os.WriteFile(filepath.Join(path, "memory.max"), []byte(memLimitBytes), 0o644); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to set memory.max: %w", err)
	}

	if err := os.WriteFile(filepath.Join(path, "memory.high"), []byte(memLimitBytes), 0o644); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to set memory.high: %w", err)
	}

	// Enforce resident-memory-only accounting so workloads cannot bypass
	// the limit by paging anonymous memory out to swap.
	if err := os.WriteFile(filepath.Join(path, "memory.swap.max"), []byte("0\n"), 0o644); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to set memory.swap.max: %w", err)
	}

	// Kill the entire cgroup as one unit on OOM instead of leaving survivors.
	if err := os.WriteFile(filepath.Join(path, "memory.oom.group"), []byte("1\n"), 0o644); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to set memory.oom.group: %w", err)
	}

	if err := os.WriteFile(filepath.Join(path, "pids.max"), []byte("64\n"), 0o644); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to set pids.max: %w", err)
	}

	return h, nil
}

func (c *cgroupHandle) AddPID(pid int) error {
	return os.WriteFile(filepath.Join(c.path, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o644)
}

func (c *cgroupHandle) ReadMemoryPeakBytes() uint64 {
	raw, err := os.ReadFile(filepath.Join(c.path, "memory.peak"))
	if err != nil {
		return 0
	}
	val := strings.TrimSpace(string(raw))
	bytesUsed, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0
	}
	return bytesUsed
}

func (c *cgroupHandle) ReadMemoryCurrentBytes() (uint64, error) {
	raw, err := os.ReadFile(filepath.Join(c.path, "memory.current"))
	if err != nil {
		return 0, err
	}
	val := strings.TrimSpace(string(raw))
	bytesUsed, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return bytesUsed, nil
}

func (c *cgroupHandle) KillAll() error {
	procs, err := os.ReadFile(filepath.Join(c.path, "cgroup.procs"))
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(procs)), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		pid, convErr := strconv.Atoi(l)
		if convErr != nil {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	return nil
}

func (c *cgroupHandle) ReadMemoryEvents() memoryEvents {
	raw, err := os.ReadFile(filepath.Join(c.path, "memory.events"))
	if err != nil {
		return memoryEvents{}
	}
	events := memoryEvents{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		v, convErr := strconv.ParseUint(fields[1], 10, 64)
		if convErr != nil {
			continue
		}
		switch fields[0] {
		case "low":
			events.Low = v
		case "high":
			events.High = v
		case "max":
			events.Max = v
		case "oom":
			events.OOM = v
		case "oom_kill":
			events.OOMKill = v
		}
	}
	return events
}

func (c *cgroupHandle) Close() {
	_ = c.KillAll()
	_ = os.Remove(c.path)
}

func bytesToKB(bytes uint64) uint {
	return uint(math.Ceil(float64(bytes) / 1024.0))
}

func applyProcessRlimits(pid int, memoryLimitKB uint) error {
	if memoryLimitKB == 0 {
		return nil
	}
	limitBytes := uint64(memoryLimitKB) * 1024
	rlim := &unix.Rlimit{Cur: limitBytes, Max: limitBytes}
	if err := unix.Prlimit(pid, unix.RLIMIT_AS, rlim, nil); err != nil {
		return err
	}
	if err := unix.Prlimit(pid, unix.RLIMIT_DATA, rlim, nil); err != nil {
		return err
	}
	return nil
}

func hardKillProcessGroup(pid int) error {
	// Negative PID targets the whole process group created with Setpgid.
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
