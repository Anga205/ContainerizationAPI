package executor

import (
	"testing"
)

func TestDefaultRuntimeMountsDoNotExposeHostEtc(t *testing.T) {
	for _, mount := range defaultRuntimeMounts() {
		if mount == "/etc" {
			t.Fatal("runtime mounts must not expose the host /etc tree")
		}
	}
}

func TestJavaRuntimeMountsOnlyAllowlistedSecurityDirectory(t *testing.T) {
	ws, err := buildWorkspacePlan("/tmp/sandbox-dir", "java")
	if err != nil {
		t.Skipf("Java runtime unavailable: %v", err)
	}
	for _, mount := range ws.runtimeMounts {
		if mount == "/etc" {
			t.Fatal("Java runtime must not expose the host /etc tree")
		}
	}
}
