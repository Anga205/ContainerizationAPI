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
}
