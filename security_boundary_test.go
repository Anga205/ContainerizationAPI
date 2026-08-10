package main_test

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSecurityBoundaryNamespacesAndInheritedResources(t *testing.T) {
	if testServer == nil {
		t.Fatal("security integration server is not running")
	}

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("read host hostname: %v", err)
	}
	sentinelDir := t.TempDir()
	sentinelPath := filepath.Join(sentinelDir, "inherited-fd-sentinel")
	if err := os.WriteFile(sentinelPath, []byte("must remain unchanged"), 0o600); err != nil {
		t.Fatalf("create sentinel: %v", err)
	}
	sentinel, err := os.Open(sentinelPath)
	if err != nil {
		t.Fatalf("open sentinel: %v", err)
	}
	defer sentinel.Close()
	before := mustFileSHA256(t, sentinelPath)

	socketPath := filepath.Join(sentinelDir, "control.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("create sentinel socket: %v", err)
	}
	defer listener.Close()

	key := int(time.Now().UnixNano() & 0x3fffffff)
	tmpName := fmt.Sprintf("security-boundary-%d", os.Getpid())
	code := fmt.Sprintf(`#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <stdio.h>
#include <string.h>
#include <sys/ipc.h>
#include <sys/msg.h>
#include <sys/sem.h>
#include <sys/shm.h>
#include <sys/mman.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

static void result(const char *name, int ok) { printf("%%s:%%s\n", name, ok ? "succeeded" : strerror(errno)); }
static int unix_connect(const char *path) {
    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd < 0) return -1;
    struct sockaddr_un address = {0};
    address.sun_family = AF_UNIX;
    snprintf(address.sun_path, sizeof(address.sun_path), "%%s", path);
    int result = connect(fd, (struct sockaddr *)&address, sizeof(address));
    close(fd);
    return result;
}
int main(void) {
    int key = %d;
    result("sysv-shm", shmget(key, 4096, IPC_CREAT | IPC_EXCL | 0600) >= 0);
    result("sysv-sem", semget(key, 1, IPC_CREAT | IPC_EXCL | 0600) >= 0);
    result("sysv-msg", msgget(key, IPC_CREAT | IPC_EXCL | 0600) >= 0);
	int posix = shm_open("/sandbox-boundary-%d", O_CREAT | O_EXCL | O_RDWR, 0600);
	result("posix-shm", posix >= 0);
	if (posix >= 0) { close(posix); shm_unlink("/sandbox-boundary-%d"); }
    char hostname[256] = {0};
    gethostname(hostname, sizeof(hostname) - 1);
    printf("hostname:%%s\n", hostname);
    result("hostname-change", sethostname("sandbox-attacker", 16) == 0);
    result("domain-change", setdomainname("sandbox-domain", 14) == 0);
    char proc[128];
    snprintf(proc, sizeof(proc), "/proc/%d/cmdline", %d);
    int hostProc = open(proc, O_RDONLY);
    result("host-proc", hostProc >= 0);
    if (hostProc >= 0) close(hostProc);
    result("host-signal", kill(%d, 0) == 0);
    result("sys-kernel", access("/sys/kernel", R_OK) == 0);
    result("sys-firmware", access("/sys/firmware", R_OK) == 0);
    int cgroup = open("/sys/fs/cgroup/cgroup.procs", O_WRONLY);
    result("sys-cgroup-write", cgroup >= 0);
    if (cgroup >= 0) close(cgroup);
    for (int fd = 3; fd < 64; fd++) {
        char path[64], target[256];
        snprintf(path, sizeof(path), "/proc/self/fd/%%d", fd);
        ssize_t size = readlink(path, target, sizeof(target) - 1);
        if (size > 0) { target[size] = 0; if (strstr(target, "inherited-fd-sentinel")) printf("inherited-fd:%%s\n", target); }
    }
    result("unix-socket", unix_connect("%s") == 0);
    int tmp = open("/tmp/%s", O_CREAT | O_WRONLY | O_EXCL, 0600);
    result("tmp-create", tmp >= 0);
    if (tmp >= 0) close(tmp);
    int varTmp = open("/var/tmp/%s", O_CREAT | O_WRONLY | O_EXCL, 0600);
    result("var-tmp-create", varTmp >= 0);
    if (varTmp >= 0) close(varTmp);
    return 0;
}
`, key, key, key, os.Getpid(), os.Getpid(), os.Getpid(), socketPath, tmpName, tmpName)

	request := buildCRequest(code, 4, 32768)
	responses := make([]simpleExecuteResponse, 0, 6)
	responses = append(responses, callSimpleExecute(t, testServer.URL, request))
	responses = append(responses, callSimpleExecute(t, testServer.URL, request))
	responses = append(responses, runSecurityBoundaryRequests(t, request, 4)...)

	for _, response := range responses {
		combined := strings.ToLower(response.Output + "\n" + response.Error)
		for _, marker := range []string{"sysv-shm:succeeded", "sysv-sem:succeeded", "sysv-msg:succeeded"} {
			if !strings.Contains(combined, marker) {
				t.Fatalf("sandbox could not create its private IPC object (%s): stdout=%q stderr=%q", marker, response.Output, response.Error)
			}
		}
		for _, marker := range []string{
			"hostname-change:succeeded", "domain-change:succeeded", "host-proc:succeeded",
			"host-signal:succeeded", "sys-kernel:succeeded", "sys-firmware:succeeded",
			"sys-cgroup-write:succeeded", "inherited-fd:", "unix-socket:succeeded",
		} {
			if strings.Contains(combined, marker) {
				t.Fatalf("security boundary crossed (%s): stdout=%q stderr=%q", marker, response.Output, response.Error)
			}
		}
		if response.Output == "" {
			t.Fatalf("security probe produced no output: stderr=%q", response.Error)
		}
	}
	if got := mustFileSHA256(t, sentinelPath); got != before {
		t.Fatalf("host sentinel changed: before=%s after=%s", before, got)
	}
	if got, err := os.Hostname(); err != nil || got != hostname {
		t.Fatalf("host hostname changed: before=%q after=%q err=%v", hostname, got, err)
	}
	for _, path := range []string{filepath.Join("/tmp", tmpName), filepath.Join("/var/tmp", tmpName)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("sandbox temporary path escaped to host: path=%s err=%v", path, err)
		}
	}
}

func runSecurityBoundaryRequests(t *testing.T, request simpleExecuteRequest, count int) []simpleExecuteResponse {
	t.Helper()
	results := make(chan simpleExecuteResponse, count)
	errors := make(chan error, count)
	for i := 0; i < count; i++ {
		go func() {
			response, err := callSimpleExecuteRaw(testServer.URL, request)
			if err != nil {
				errors <- err
				return
			}
			results <- response
		}()
	}
	responses := make([]simpleExecuteResponse, 0, count)
	for i := 0; i < count; i++ {
		select {
		case err := <-errors:
			t.Fatalf("concurrent security request failed: %v", err)
		case response := <-results:
			responses = append(responses, response)
		}
	}
	return responses
}
