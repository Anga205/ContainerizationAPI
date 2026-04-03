package executor

import (
	"errors"
	"syscall"
	"testing"
)

func TestBindMountReadOnlyUsesRemountReadOnly(t *testing.T) {
	original := syscallMount
	t.Cleanup(func() { syscallMount = original })

	type mountCall struct {
		source string
		target string
		fstype string
		flags  uintptr
		data   string
	}
	calls := make([]mountCall, 0, 2)

	syscallMount = func(source, target, fstype string, flags uintptr, data string) error {
		calls = append(calls, mountCall{source: source, target: target, fstype: fstype, flags: flags, data: data})
		return nil
	}

	if err := bindMountReadOnly("/src", "/dst"); err != nil {
		t.Fatalf("bindMountReadOnly returned error: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("unexpected mount call count: got=%d want=2", len(calls))
	}

	if calls[0].source != "/src" || calls[0].target != "/dst" {
		t.Fatalf("unexpected first mount call: %+v", calls[0])
	}
	wantBindFlags := uintptr(syscall.MS_BIND | syscall.MS_REC)
	if calls[0].flags != wantBindFlags {
		t.Fatalf("unexpected first mount flags: got=%#x want=%#x", calls[0].flags, wantBindFlags)
	}

	if calls[1].source != "" || calls[1].target != "/dst" {
		t.Fatalf("unexpected second mount call: %+v", calls[1])
	}
	wantRemountFlags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_REC)
	if calls[1].flags != wantRemountFlags {
		t.Fatalf("unexpected second mount flags: got=%#x want=%#x", calls[1].flags, wantRemountFlags)
	}
}

func TestBindMountReadOnlyReturnsFirstError(t *testing.T) {
	original := syscallMount
	t.Cleanup(func() { syscallMount = original })

	wantErr := errors.New("bind failed")
	syscallMount = func(source, target, fstype string, flags uintptr, data string) error {
		return wantErr
	}

	err := bindMountReadOnly("/src", "/dst")
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: got=%v want=%v", err, wantErr)
	}
}

func TestPrepareRuntimeMountsMarksRootPrivateFirst(t *testing.T) {
	original := syscallMount
	t.Cleanup(func() { syscallMount = original })

	tempDir := t.TempDir()
	source := t.TempDir()

	type mountCall struct {
		source string
		target string
		flags  uintptr
	}
	calls := make([]mountCall, 0, 3)

	syscallMount = func(source, target, fstype string, flags uintptr, data string) error {
		calls = append(calls, mountCall{source: source, target: target, flags: flags})
		return nil
	}

	ws := sandboxWorkspace{
		dir:           tempDir,
		useChroot:     true,
		runtimeMounts: []string{source},
	}

	if err := prepareRuntimeMounts(&ws); err != nil {
		t.Fatalf("prepareRuntimeMounts returned error: %v", err)
	}
	if len(calls) < 1 {
		t.Fatal("expected at least one mount call")
	}

	if calls[0].source != "" || calls[0].target != "/" {
		t.Fatalf("first mount must mark root private: %+v", calls[0])
	}
	wantFlags := uintptr(syscall.MS_REC | syscall.MS_PRIVATE)
	if calls[0].flags != wantFlags {
		t.Fatalf("unexpected private-root mount flags: got=%#x want=%#x", calls[0].flags, wantFlags)
	}
}
