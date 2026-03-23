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

const sampleCodeDir = "sample_code_for_tests"

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

type integrationHarness struct {
	baseURL string
	apiPort int
}

var (
	testServer *httptest.Server
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func TestMain(m *testing.M) {
	enforceRootAndCompiler()
	config.Config.Globals.ENABLE_QUEUE = false

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	routes.Setup(router)
	testServer = httptest.NewServer(router)

	exitCode := m.Run()
	testServer.Close()
	os.Exit(exitCode)
}

func enforceRootAndCompiler() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "integration tests must run as root (sudo go test -v)")
		os.Exit(1)
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		fmt.Fprintf(os.Stderr, "gcc is required for integration tests: %v\n", err)
		os.Exit(1)
	}
}

func TestContainerizationAPISecurityIntegration(t *testing.T) {
	if config.Config.Globals.ENABLE_QUEUE {
		t.Fatal("ENABLE_QUEUE must be false for integration tests")
	}

	h := integrationHarness{baseURL: testServer.URL}
	h.apiPort = mustExtractPort(t, h.baseURL)

	cases := []struct {
		name string
		run  func(*testing.T, integrationHarness)
	}{
		{name: "file privacy across request IDs", run: testFilesystemIsolation},
		{name: "disk spammer is terminated and data is reclaimed", run: testDiskCleanup},
		{name: "fork bomb does not poison subsequent requests", run: testForkBombContainment},
		{name: "network namespace blocks localhost bridge", run: testNetworkIsolation},
		{name: "memory hard limit triggers oom kill", run: testMemoryHardLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { tc.run(t, h) })
	}
}

func testFilesystemIsolation(t *testing.T, h integrationHarness) {
	writeCode := mustLoadSampleCode(t, "file_privacy_write_read.c", nil)
	writeResp := callSimpleExecute(t, h.baseURL, buildCRequest(writeCode, 4, 32768))
	if !strings.Contains(writeResp.Output, "SecretData123") {
		t.Fatalf("step A did not return secret in stdout; stdout=%q stderr=%q", writeResp.Output, writeResp.Error)
	}

	readCode := mustLoadSampleCode(t, "file_privacy_read_only.c", nil)
	readResp := callSimpleExecute(t, h.baseURL, buildCRequest(readCode, 4, 32768))
	if !strings.Contains(strings.ToLower(readResp.Error), "no such file or directory") {
		t.Fatalf("step B unexpectedly accessed file from another sandbox; stdout=%q stderr=%q", readResp.Output, readResp.Error)
	}
}

func testDiskCleanup(t *testing.T, h integrationHarness) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := mustLoadSampleCode(t, "disk_spammer.c", nil)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 2, 32768))
	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "memory limit exceeded"}) {
		t.Fatalf("disk spammer was not terminated as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
}

func assertDiskReclaimed(t *testing.T, beforeFree, afterFree uint64) {
	const maxResidual = uint64(20 * 1024 * 1024)
	if beforeFree > afterFree && (beforeFree-afterFree) > maxResidual {
		t.Fatalf("sandbox disk garbage appears to persist: free space dropped by %d bytes", beforeFree-afterFree)
	}
}

func assertNoSandboxLeak(t *testing.T, beforeCount, afterCount int) {
	if afterCount > beforeCount {
		t.Fatalf("sandbox temp directories leaked: before=%d after=%d", beforeCount, afterCount)
	}
}

func testForkBombContainment(t *testing.T, h integrationHarness) {
	bombCode := mustLoadSampleCode(t, "fork_bomb.c", nil)
	_ = callSimpleExecute(t, h.baseURL, buildCRequest(bombCode, 2, 32768))

	helloCode := mustLoadSampleCode(t, "hello_world.c", nil)
	helloResp := callSimpleExecute(t, h.baseURL, buildCRequest(helloCode, 4, 32768))
	if !strings.Contains(helloResp.Output, "Hello World") {
		t.Fatalf("follow-up request failed after fork bomb; stdout=%q stderr=%q", helloResp.Output, helloResp.Error)
	}
	if strings.Contains(strings.ToLower(helloResp.Error), "resource temporarily unavailable") {
		t.Fatalf("follow-up request indicates host PID exhaustion; stderr=%q", helloResp.Error)
	}
}

func testNetworkIsolation(t *testing.T, h integrationHarness) {
	replacements := map[string]string{"__API_PORT__": strconv.Itoa(h.apiPort)}
	code := mustLoadSampleCode(t, "network_localhost_bridge.c", replacements)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 3, 32768))

	combined := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combined, "connected") {
		t.Fatalf("localhost bridge unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combined, []string{"connection refused", "network is unreachable", "no route to host"}) {
		t.Fatalf("network isolation did not produce expected connect error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func testMemoryHardLimit(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "memory_bomb.c", nil)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 3, 16384))
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
}

func mustLoadSampleCode(t *testing.T, fileName string, replacements map[string]string) string {
	t.Helper()
	path := filepath.Join(sampleCodeDir, fileName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample code %s: %v", path, err)
	}

	code := string(content)
	for oldValue, newValue := range replacements {
		code = strings.ReplaceAll(code, oldValue, newValue)
	}
	return code
}

func buildCRequest(code string, timeoutSec, maxMemoryKB uint) simpleExecuteRequest {
	return simpleExecuteRequest{Language: "c", Code: code, Timeout: timeoutSec, MaxMemory: maxMemoryKB}
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
	return decodeSimpleResponse(t, httpResp)
}

func decodeSimpleResponse(t *testing.T, httpResp *http.Response) simpleExecuteResponse {
	t.Helper()
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
