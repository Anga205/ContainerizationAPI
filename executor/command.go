package executor

import (
	"os/exec"
	"reflect"
	"strings"
	"syscall"
)

func buildSandboxCommand(stdin string, ws sandboxWorkspace, cgroupFD int) *exec.Cmd {
	return buildSandboxCommandWithCgroupMode(stdin, ws, cgroupFD, true)
}

func buildSandboxCommandWithoutCgroupFD(stdin string, ws sandboxWorkspace) *exec.Cmd {
	return buildSandboxCommandWithCgroupMode(stdin, ws, 0, false)
}

func buildSandboxCommandWithCgroupMode(stdin string, ws sandboxWorkspace, cgroupFD int, useCgroupFD bool) *exec.Cmd {
	command := ws.runCommand
	if ws.sandboxInitExecPath != "" {
		command = append([]string{ws.sandboxInitExecPath}, ws.runCommand...)
	}

	cmd := exec.Command(command[0], command[1:]...)
	sysProcAttr := &syscall.SysProcAttr{
		Cloneflags:                 syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
		Setpgid:                    true,
		UseCgroupFD:                useCgroupFD,
		CgroupFD:                   cgroupFD,
		Credential:                 &syscall.Credential{Uid: 0, Gid: 0},
		UidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: 65534, Size: 1}},
		GidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: 65534, Size: 1}},
		GidMappingsEnableSetgroups: false,
	}
	setNoNewPrivsIfSupported(sysProcAttr)
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
	baseEnv := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/",
		"LANG=C.UTF-8",
	}
	cmd.Env = append([]string{}, baseEnv...)
	if len(ws.extraEnv) > 0 {
		cmd.Env = append(cmd.Env, ws.extraEnv...)
	}
	return cmd
}

func setNoNewPrivsIfSupported(attr *syscall.SysProcAttr) {
	field := reflect.ValueOf(attr).Elem().FieldByName("NoNewPrivs")
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.Bool {
		field.SetBool(true)
	}
}
