package resourcemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func NewCgroupV2(memoryLimitKB uint) (*CgroupHandle, error) {
	path, err := createCgroupPath()
	if err != nil {
		return nil, err
	}

	handle := &CgroupHandle{path: path}
	if err := configureCgroupFiles(handle.path, memoryLimitKB); err != nil {
		handle.Close()
		return nil, err
	}
	return handle, nil
}

func createCgroupPath() (string, error) {
	name := fmt.Sprintf("codesandbox-%d-%d", os.Getpid(), time.Now().UnixNano())
	path := filepath.Join("/sys/fs/cgroup", name)

	cgroupWriteMu.Lock()
	defer cgroupWriteMu.Unlock()

	if err := os.Mkdir(path, 0o755); err != nil {
		return "", wrapCreateError(err)
	}
	return path, nil
}

func wrapCreateError(err error) error {
	if os.IsPermission(err) {
		return fmt.Errorf("cannot create cgroup (need CAP_SYS_ADMIN/root): %w", err)
	}
	return fmt.Errorf("failed to create cgroup: %w", err)
}

func configureCgroupFiles(path string, memoryLimitKB uint) error {
	for name, content := range cgroupControlFiles(memoryLimitKB) {
		if err := writeCgroupControl(path, name, content); err != nil {
			return err
		}
	}
	return nil
}

func writeCgroupControl(path, name, content string) error {
	filePath := filepath.Join(path, name)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to set %s: %w", name, err)
	}
	return nil
}
