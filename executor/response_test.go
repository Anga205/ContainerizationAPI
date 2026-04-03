package executor

import (
	"os/exec"
	"strings"
	"testing"
)

func TestAppendExitDiagnosticAddsSeccompHintForSigsys(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "kill -SYS $$")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected process to fail with SIGSYS")
	}

	got := appendExitDiagnostic("", err)
	if !strings.Contains(strings.ToLower(got), "sigsys") {
		t.Fatalf("expected SIGSYS in diagnostic, got: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "seccomp") {
		t.Fatalf("expected seccomp hint in diagnostic, got: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "bad system call") {
		t.Fatalf("expected bad system call phrase in diagnostic, got: %q", got)
	}
}

func TestAppendExitDiagnosticReportsExitCode(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 7")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected process to exit with non-zero status")
	}

	got := appendExitDiagnostic("", err)
	if !strings.Contains(got, "non-zero status 7") {
		t.Fatalf("expected exit status details, got: %q", got)
	}
}

func TestAppendExitDiagnosticPreservesExistingStderr(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 9")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected process to exit with non-zero status")
	}

	got := appendExitDiagnostic("runtime error", err)
	if !strings.HasPrefix(got, "runtime error\n") {
		t.Fatalf("expected existing stderr to be preserved, got: %q", got)
	}
	if !strings.Contains(got, "non-zero status 9") {
		t.Fatalf("expected appended exit status details, got: %q", got)
	}
}
