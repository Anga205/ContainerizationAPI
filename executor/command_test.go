package executor

import (
	"syscall"
	"testing"
)

func TestBuildSandboxCommandSetsIsolationFlags(t *testing.T) {
	ws := sandboxWorkspace{
		dir:        "/tmp/sandbox-dir",
		runCommand: []string{"/bin/echo", "ok"},
		useChroot:  true,
	}

	cmd := buildSandboxCommand("", ws, 7)
	attr := cmd.SysProcAttr
	if attr == nil {
		t.Fatal("SysProcAttr must be set")
	}

	want := uintptr(syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET)
	if attr.Cloneflags&want != want {
		t.Fatalf("missing expected clone flags: got=%#x want bits=%#x", attr.Cloneflags, want)
	}
	if !attr.Setpgid {
		t.Fatal("Setpgid must be enabled for process-group cleanup")
	}
	if !attr.UseCgroupFD {
		t.Fatal("UseCgroupFD must be enabled")
	}
	if attr.CgroupFD != 7 {
		t.Fatalf("unexpected cgroup fd: got=%d want=7", attr.CgroupFD)
	}
	if attr.Chroot != ws.dir {
		t.Fatalf("unexpected chroot: got=%q want=%q", attr.Chroot, ws.dir)
	}
	if cmd.Dir != "/" {
		t.Fatalf("unexpected command dir: got=%q want=%q", cmd.Dir, "/")
	}
}
