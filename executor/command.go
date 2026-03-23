package executor

import (
	"os/exec"
	"strings"
	"syscall"
)

func buildSandboxCommand(stdin, sandboxDir string, cgroupFD int) *exec.Cmd {
	cmd := exec.Command("/program")
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
		Chroot:      sandboxDir,
		Setpgid:     true,
		UseCgroupFD: true,
		CgroupFD:    cgroupFD,
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	return cmd
}
