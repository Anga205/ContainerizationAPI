package executor

import (
	"strings"
	"testing"
)

func TestSandboxInitSourceMountsProcfsBeforeSeccomp(t *testing.T) {
	if !strings.Contains(sandboxInitSource, "mount(\"proc\", \"/proc\", \"proc\"") {
		t.Fatal("sandbox init source must attempt to mount procfs before installing seccomp")
	}
	if !strings.Contains(sandboxInitSource, "#include <sys/mount.h>") {
		t.Fatal("sandbox init source must include sys/mount.h for procfs mounting")
	}
	if strings.Contains(sandboxInitSource, "mount_procfs_if_possible") {
		t.Fatal("procfs setup must not fail open")
	}
	if !strings.Contains(sandboxInitSource, "drop_capabilities") || !strings.Contains(sandboxInitSource, "SYS_capset") {
		t.Fatal("sandbox init must clear capabilities before executing user code")
	}
	if !strings.Contains(sandboxInitSource, "__NR_bpf") || !strings.Contains(sandboxInitSource, "__NR_unshare") {
		t.Fatal("seccomp policy must deny namespace and kernel attack syscalls")
	}
}
