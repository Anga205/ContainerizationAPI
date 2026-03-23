package executor

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func buildSandboxCommand(stdin string, ws sandboxWorkspace, cgroupFD int) *exec.Cmd {
	cmd := exec.Command(ws.runCommand[0], ws.runCommand[1:]...)
	sysProcAttr := &syscall.SysProcAttr{
		Cloneflags:  syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
		Setpgid:     true,
		UseCgroupFD: true,
		CgroupFD:    cgroupFD,
	}
	if ws.useChroot {
		sysProcAttr.Chroot = ws.dir
		cmd.Dir = "/"
	} else {
		cmd.Dir = ws.dir
	}
	cmd.SysProcAttr = sysProcAttr
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if len(ws.extraEnv) > 0 {
		cmd.Env = append(os.Environ(), ws.extraEnv...)
	}
	return cmd
}
