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

	want := uintptr(syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER)
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
	if attr.Credential == nil {
		t.Fatal("Credential must be set")
	}
	if attr.Credential.Uid != 0 || attr.Credential.Gid != 0 {
		t.Fatalf("unexpected credential: uid=%d gid=%d", attr.Credential.Uid, attr.Credential.Gid)
	}
	if len(attr.UidMappings) != 1 {
		t.Fatalf("unexpected uid mapping count: got=%d want=1", len(attr.UidMappings))
	}
	if got := attr.UidMappings[0]; got.ContainerID != 0 || got.HostID != 65534 || got.Size != 1 {
		t.Fatalf("unexpected uid mapping: %+v", got)
	}
	if len(attr.GidMappings) != 1 {
		t.Fatalf("unexpected gid mapping count: got=%d want=1", len(attr.GidMappings))
	}
	if got := attr.GidMappings[0]; got.ContainerID != 0 || got.HostID != 65534 || got.Size != 1 {
		t.Fatalf("unexpected gid mapping: %+v", got)
	}
	if attr.GidMappingsEnableSetgroups {
		t.Fatal("GidMappingsEnableSetgroups must be false for namespace setup")
	}
	if attr.Chroot != ws.dir {
		t.Fatalf("unexpected chroot: got=%q want=%q", attr.Chroot, ws.dir)
	}
	if cmd.Dir != "/" {
		t.Fatalf("unexpected command dir: got=%q want=%q", cmd.Dir, "/")
	}
}

func TestBuildSandboxCommandUsesMinimalEnvironment(t *testing.T) {
	ws := sandboxWorkspace{
		dir:        "/tmp/sandbox-dir",
		runCommand: []string{"/bin/echo", "ok"},
		extraEnv:   []string{"JAVA_HOME=/usr/lib/jvm/default-java"},
	}

	cmd := buildSandboxCommand("", ws, 5)

	want := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/",
		"LANG=C.UTF-8",
		"JAVA_HOME=/usr/lib/jvm/default-java",
	}
	if len(cmd.Env) != len(want) {
		t.Fatalf("unexpected env count: got=%d want=%d env=%v", len(cmd.Env), len(want), cmd.Env)
	}
	for i := range want {
		if cmd.Env[i] != want[i] {
			t.Fatalf("unexpected env var at %d: got=%q want=%q", i, cmd.Env[i], want[i])
		}
	}
}
