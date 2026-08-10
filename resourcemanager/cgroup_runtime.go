package resourcemanager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func (c *CgroupHandle) OpenFD() (int, error) {
	return unix.Open(c.path, unix.O_DIRECTORY|unix.O_RDONLY, 0)
}

func (c *CgroupHandle) AddProcess(pid int) error {
	cgroupWriteMu.Lock()
	defer cgroupWriteMu.Unlock()

	procsPath := filepath.Join(c.path, "cgroup.procs")
	if err := os.WriteFile(procsPath, []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to attach pid %d to cgroup: %w", pid, err)
	}
	return nil
}

func (c *CgroupHandle) ReadMemoryPeakBytes() uint64 {
	return readUintFromFile(filepath.Join(c.path, "memory.peak"))
}

func (c *CgroupHandle) ReadMemoryCurrentBytes() (uint64, error) {
	return readUintWithError(filepath.Join(c.path, "memory.current"))
}

func (c *CgroupHandle) ReadMemoryEvents() MemoryEvents {
	content, err := os.ReadFile(filepath.Join(c.path, "memory.events"))
	if err != nil {
		return MemoryEvents{}
	}
	return parseMemoryEvents(string(content))
}

func (c *CgroupHandle) KillAll() error {
	if err := os.WriteFile(filepath.Join(c.path, "cgroup.kill"), []byte("1\n"), 0o644); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to kill cgroup processes: %w", err)
	}

	var killErr error
	pids, err := readCgroupPIDs(c.path)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			killErr = errors.Join(killErr, fmt.Errorf("kill pid %d: %w", pid, err))
		}
	}
	return killErr
}

func (c *CgroupHandle) Close() {
	_ = c.KillAll()
	deadline := time.Now().Add(2 * time.Second)
	for {
		pids, err := readCgroupPIDs(c.path)
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err == nil && len(pids) == 0 {
			if err := os.Remove(c.path); err == nil || errors.Is(err, os.ErrNotExist) {
				return
			}
		}
		if time.Now().After(deadline) {
			return
		}
		_ = c.KillAll()
		time.Sleep(10 * time.Millisecond)
	}
}

func readCgroupPIDs(path string) ([]int, error) {
	data, err := os.ReadFile(filepath.Join(path, "cgroup.procs"))
	if err != nil {
		return nil, err
	}
	return parsePIDLines(string(data)), nil
}

func parsePIDLines(content string) []int {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	result := make([]int, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		pid, err := strconv.Atoi(line)
		if err == nil {
			result = append(result, pid)
		}
	}
	return result
}
