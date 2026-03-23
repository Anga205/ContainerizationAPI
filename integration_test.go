package main_test

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/routes"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type simpleExecuteRequest struct {
	Language  string   `json:"language"`
	Code      string   `json:"code"`
	Timeout   uint     `json:"timeout,omitempty"`
	MaxMemory uint     `json:"max_memory,omitempty"`
	Inputs    []string `json:"inputs,omitempty"`
}

type simpleExecuteResponse struct {
	Output     string `json:"output"`
	Error      string `json:"error"`
	MemoryUsed string `json:"memory_used"`
	CPUTime    string `json:"cpu_time"`
}

var (
	testServer *httptest.Server
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func TestMain(m *testing.M) {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "integration tests must run as root (sudo go test -v)")
		os.Exit(1)
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		fmt.Fprintf(os.Stderr, "gcc is required for integration tests: %v\n", err)
		os.Exit(1)
	}

	config.Config.Globals.ENABLE_QUEUE = false

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	routes.Setup(router)
	testServer = httptest.NewServer(router)

	exitCode := m.Run()
	testServer.Close()
	os.Exit(exitCode)
}

func TestContainerizationAPISecurityIntegration(t *testing.T) {
	if config.Config.Globals.ENABLE_QUEUE {
		t.Fatal("ENABLE_QUEUE must be false for integration tests")
	}

	baseURL := testServer.URL
	apiPort := mustExtractPort(t, baseURL)

	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "file privacy across request IDs", run: func(t *testing.T) {
			const secret = "SecretData123"
			writePayload := `
#include <stdio.h>
#include <sys/stat.h>

int main() {
    mkdir("/root", 0700);
    FILE *f = fopen("/root/test.txt", "w");
    if (!f) {
        perror("fopen write");
        return 1;
    }
    fprintf(f, "SecretData123");
    fclose(f);

    char buf[64] = {0};
    f = fopen("/root/test.txt", "r");
    if (!f) {
        perror("fopen read");
        return 1;
    }
    fgets(buf, sizeof(buf), f);
    fclose(f);
    printf("%s", buf);
    return 0;
}
`
			respA := callSimpleExecute(t, baseURL, simpleExecuteRequest{Language: "c", Code: writePayload, Timeout: 4, MaxMemory: 32768})
			if !strings.Contains(respA.Output, secret) {
				t.Fatalf("step A did not return secret in stdout; stdout=%q stderr=%q", respA.Output, respA.Error)
			}

			readPayload := `
#include <stdio.h>

int main() {
    FILE *f = fopen("/root/test.txt", "r");
    if (!f) {
        perror("fopen");
        return 1;
    }
    char buf[64] = {0};
    fgets(buf, sizeof(buf), f);
    fclose(f);
    printf("%s", buf);
    return 0;
}
`
			respB := callSimpleExecute(t, baseURL, simpleExecuteRequest{Language: "c", Code: readPayload, Timeout: 4, MaxMemory: 32768})
			if !strings.Contains(strings.ToLower(respB.Error), "no such file or directory") {
				t.Fatalf("step B unexpectedly accessed file from another sandbox; stdout=%q stderr=%q", respB.Output, respB.Error)
			}
		}},
		{name: "disk spammer is terminated and data is reclaimed", run: func(t *testing.T) {
			beforeFree := mustFreeBytes(t, os.TempDir())
			beforeCount := countSandboxTempDirs(t)

			spammerPayload := `
#include <stdio.h>
#include <string.h>

int main() {
    static char buf[1024 * 1024];
    memset(buf, 'A', sizeof(buf));

    FILE *f = fopen("/disk_spam.bin", "w");
    if (!f) {
        perror("fopen");
        return 1;
    }

    while (1) {
        if (fwrite(buf, 1, sizeof(buf), f) != sizeof(buf)) {
            perror("fwrite");
            fflush(f);
            return 1;
        }
        fflush(f);
    }
}
`
			resp := callSimpleExecute(t, baseURL, simpleExecuteRequest{Language: "c", Code: spammerPayload, Timeout: 2, MaxMemory: 32768})
			stderrLower := strings.ToLower(resp.Error)
			if !strings.Contains(stderrLower, "execution timed out") && !strings.Contains(stderrLower, "memory limit exceeded") {
				t.Fatalf("disk spammer was not terminated as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
			}

			time.Sleep(400 * time.Millisecond)
			afterFree := mustFreeBytes(t, os.TempDir())
			afterCount := countSandboxTempDirs(t)

			const maxAllowedResidualBytes = uint64(20 * 1024 * 1024)
			if beforeFree > afterFree && (beforeFree-afterFree) > maxAllowedResidualBytes {
				t.Fatalf("sandbox disk garbage appears to persist: free space dropped by %d bytes", beforeFree-afterFree)
			}
			if afterCount > beforeCount {
				t.Fatalf("sandbox temp directories leaked: before=%d after=%d", beforeCount, afterCount)
			}
		}},
		{name: "fork bomb does not poison subsequent requests", run: func(t *testing.T) {
			forkBombPayload := `
#include <stdio.h>
#include <unistd.h>

int main() {
    while (1) {
        pid_t p = fork();
        if (p < 0) {
            perror("fork");
            return 0;
        }
        if (p == 0) {
            for (;;) {
                pause();
            }
        }
    }
}
`
			_ = callSimpleExecute(t, baseURL, simpleExecuteRequest{Language: "c", Code: forkBombPayload, Timeout: 2, MaxMemory: 32768})

			helloPayload := `
#include <stdio.h>
int main() {
    printf("Hello World\\n");
    return 0;
}
`
			helloResp := callSimpleExecute(t, baseURL, simpleExecuteRequest{Language: "c", Code: helloPayload, Timeout: 4, MaxMemory: 32768})
			if !strings.Contains(helloResp.Output, "Hello World") {
				t.Fatalf("follow-up request failed after fork bomb; stdout=%q stderr=%q", helloResp.Output, helloResp.Error)
			}
			if strings.Contains(strings.ToLower(helloResp.Error), "resource temporarily unavailable") {
				t.Fatalf("follow-up request indicates host PID exhaustion; stderr=%q", helloResp.Error)
			}
		}},
		{name: "network namespace blocks localhost bridge", run: func(t *testing.T) {
			netPayload := fmt.Sprintf(`
#include <arpa/inet.h>
#include <stdio.h>
#include <string.h>
#include <sys/socket.h>

int main() {
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) {
        perror("socket");
        return 1;
    }

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(%d);
    inet_pton(AF_INET, "127.0.0.1", &addr.sin_addr);

    if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) == 0) {
        printf("connected");
        return 0;
    }

    perror("connect");
    return 1;
}
`, apiPort)
			resp := callSimpleExecute(t, baseURL, simpleExecuteRequest{Language: "c", Code: netPayload, Timeout: 3, MaxMemory: 32768})
			combined := strings.ToLower(resp.Output + "\n" + resp.Error)
			if strings.Contains(combined, "connected") {
				t.Fatalf("localhost bridge unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
			}
			if !containsAny(combined, []string{"connection refused", "network is unreachable", "no route to host"}) {
				t.Fatalf("network isolation did not produce expected connect error; stdout=%q stderr=%q", resp.Output, resp.Error)
			}
		}},
		{name: "memory hard limit triggers oom kill", run: func(t *testing.T) {
			memoryBombPayload := `
#define _GNU_SOURCE
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <unistd.h>
#include <sys/types.h>

int main() {
    const size_t chunk = 1024 * 1024;

    for (int i = 0; i < 16; i++) {
        pid_t p = fork();
        if (p == 0) {
            while (1) {
                void *ptr = mmap(NULL, chunk, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
                memset(ptr, 0xAB, chunk);
            }
        }
    }

    while (1) {
        pause();
    }
}
`
			resp := callSimpleExecute(t, baseURL, simpleExecuteRequest{Language: "c", Code: memoryBombPayload, Timeout: 3, MaxMemory: 16384})
			if !containsAny(strings.ToLower(resp.Error), []string{"memory limit exceeded", "killed", "execution timed out"}) {
				t.Fatalf("memory bomb did not terminate as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
			}
			execDur, err := time.ParseDuration(resp.CPUTime)
			if err != nil {
				t.Fatalf("failed to parse cpu_time %q: %v", resp.CPUTime, err)
			}
			if execDur > 500*time.Millisecond {
				t.Fatalf("memory enforcement was too slow: cpu_time=%s stderr=%q", resp.CPUTime, resp.Error)
			}
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func callSimpleExecute(t *testing.T, baseURL string, req simpleExecuteRequest) simpleExecuteResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/simple-execute", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	rawBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected HTTP status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	var parsed simpleExecuteResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		t.Fatalf("failed to decode response JSON: %v; body=%q", err, strings.TrimSpace(string(rawBody)))
	}
	return parsed
}

func mustExtractPort(t *testing.T, serverURL string) int {
	t.Helper()
	hostPort := strings.TrimPrefix(serverURL, "http://")
	_, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("failed to parse test server URL %q: %v", serverURL, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse server port %q: %v", portStr, err)
	}
	return port
}

func mustFreeBytes(t *testing.T, path string) uint64 {
	t.Helper()
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		t.Fatalf("statfs failed for %s: %v", path, err)
	}
	return stat.Bavail * uint64(stat.Bsize)
}

func countSandboxTempDirs(t *testing.T) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "codesandbox-c-*"))
	if err != nil {
		t.Fatalf("failed to glob sandbox temp dirs: %v", err)
	}
	return len(matches)
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
