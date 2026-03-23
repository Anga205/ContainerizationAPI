package resourcemanager

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func (c *CgroupHandle) OpenFD() (int, error) {
	return unix.Open(c.path, unix.O_DIRECTORY|unix.O_RDONLY, 0)
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
	pids, err := readCgroupPIDs(c.path)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	return nil
}

func (c *CgroupHandle) Close() {
	_ = c.KillAll()
	_ = os.Remove(c.path)
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
