package main_test

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/routes"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
		{name: "io flood is bounded and returns before timeout", run: testIOFloodResilience},
		{name: "signal trap cannot survive forced timeout", run: testSignalTrapTimeout},
		{name: "orphan grandchild is reaped after request exits", run: testOrphanReaping},
		{name: "inode bomb does not poison host temp filesystem", run: testInodeExhaustion},
		{name: "privileged reboot syscall is denied", run: testPrivilegedSyscallDenied},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { tc.run(t, h) })
	}
}

func TestContainerizationAPISecurityIntegrationPython3(t *testing.T) {
	if config.Config.Globals.ENABLE_QUEUE {
		t.Fatal("ENABLE_QUEUE must be false for integration tests")
	}

	h := integrationHarness{baseURL: testServer.URL}
	h.apiPort = mustExtractPort(t, h.baseURL)

	cases := []struct {
		name string
		run  func(*testing.T, integrationHarness)
	}{
		{name: "file privacy across request IDs", run: testFilesystemIsolationPython3},
		{name: "disk spammer is terminated and data is reclaimed", run: testDiskCleanupPython3},
		{name: "fork bomb does not poison subsequent requests", run: testForkBombContainmentPython3},
		{name: "network namespace blocks localhost bridge", run: testNetworkIsolationPython3},
		{name: "memory hard limit triggers oom kill", run: testMemoryHardLimitPython3},
		{name: "io flood is bounded and returns before timeout", run: testIOFloodResiliencePython3},
		{name: "signal trap cannot survive forced timeout", run: testSignalTrapTimeoutPython3},
		{name: "orphan grandchild is reaped after request exits", run: testOrphanReapingPython3},
		{name: "inode bomb does not poison host temp filesystem", run: testInodeExhaustionPython3},
		{name: "privileged reboot syscall is denied", run: testPrivilegedSyscallDeniedPython3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { tc.run(t, h) })
	}
}

func TestContainerizationAPISecurityIntegrationCpp(t *testing.T) {
	if config.Config.Globals.ENABLE_QUEUE {
		t.Fatal("ENABLE_QUEUE must be false for integration tests")
	}

	h := integrationHarness{baseURL: testServer.URL}
	h.apiPort = mustExtractPort(t, h.baseURL)
	runLanguageMirrorSuite(t, h, "cpp", "cpp")
}

func TestContainerizationAPISecurityIntegrationJava(t *testing.T) {
	if config.Config.Globals.ENABLE_QUEUE {
		t.Fatal("ENABLE_QUEUE must be false for integration tests")
	}

	h := integrationHarness{baseURL: testServer.URL}
	h.apiPort = mustExtractPort(t, h.baseURL)
	runLanguageMirrorSuite(t, h, "java", "java")
}

func runLanguageMirrorSuite(t *testing.T, h integrationHarness, language, ext string) {
	t.Helper()

	cases := []struct {
		name string
		run  func(*testing.T, integrationHarness)
	}{
		{name: "file privacy across request IDs", run: func(t *testing.T, h integrationHarness) {
			testFilesystemIsolationForLanguage(t, h, language, "file_privacy_write_read."+ext, "file_privacy_read_only."+ext)
		}},
		{name: "disk spammer is terminated and data is reclaimed", run: func(t *testing.T, h integrationHarness) {
			testDiskCleanupForLanguage(t, h, language, "disk_spammer."+ext)
		}},
		{name: "fork bomb does not poison subsequent requests", run: func(t *testing.T, h integrationHarness) {
			testForkBombContainmentForLanguage(t, h, language, "fork_bomb."+ext, "hello_world."+ext)
		}},
		{name: "network namespace blocks localhost bridge", run: func(t *testing.T, h integrationHarness) {
			testNetworkIsolationForLanguage(t, h, language, "network_localhost_bridge."+ext)
		}},
		{name: "memory hard limit triggers oom kill", run: func(t *testing.T, h integrationHarness) {
			testMemoryHardLimitForLanguage(t, h, language, "memory_bomb."+ext)
		}},
		{name: "io flood is bounded and returns before timeout", run: func(t *testing.T, h integrationHarness) {
			testIOFloodResilienceForLanguage(t, h, language, "io_spam."+ext)
		}},
		{name: "signal trap cannot survive forced timeout", run: func(t *testing.T, h integrationHarness) {
			testSignalTrapTimeoutForLanguage(t, h, language, "signal_trap."+ext)
		}},
		{name: "orphan grandchild is reaped after request exits", run: func(t *testing.T, h integrationHarness) {
			name := "orphanmakergc"
			if language == "python3" {
				name = "orphanpygc"
			}
			if language == "java" {
				name = "orphanjavagc"
			}
			testOrphanReapingForLanguage(t, h, language, "orphan_maker."+ext, name)
		}},
		{name: "inode bomb does not poison host temp filesystem", run: func(t *testing.T, h integrationHarness) {
			testInodeExhaustionForLanguage(t, h, language, "inode_bomb."+ext)
		}},
		{name: "privileged reboot syscall is denied", run: func(t *testing.T, h integrationHarness) {
			testPrivilegedSyscallDeniedForLanguage(t, h, language, "try_reboot."+ext)
		}},
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

func testIOFloodResilience(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "io_spam.c", nil)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 1, 32768))

	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "killed", "memory limit exceeded"}) {
		t.Fatalf("io flood did not terminate with expected error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	if strings.TrimSpace(resp.Error) == "" {
		t.Fatalf("io flood returned empty stderr; expected bounded but non-empty error output")
	}

	const maxErrorBytes = (1 << 20) + 4096
	if len(resp.Error) > maxErrorBytes {
		t.Fatalf("stderr exceeded resilience cap: got=%d bytes cap=%d", len(resp.Error), maxErrorBytes)
	}

	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "io flood request")
}

func testSignalTrapTimeout(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "signal_trap.c", nil)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 1, 32768))

	if !strings.Contains(strings.ToLower(resp.Error), "execution timed out") {
		t.Fatalf("signal trap did not timeout as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	assertDurationWindow(t, resp.CPUTime, 900*time.Millisecond, 2500*time.Millisecond, "signal trap timeout")
}

func testOrphanReaping(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "orphan_maker.c", nil)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 2, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "orphan maker request")

	time.Sleep(400 * time.Millisecond)
	if cnt := countProcessesByComm(t, "orphanmakergc"); cnt > 0 {
		t.Fatalf("orphan grandchild leaked to host after request completion: count=%d stderr=%q", cnt, resp.Error)
	}
}

func testInodeExhaustion(t *testing.T, h integrationHarness) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := mustLoadSampleCode(t, "inode_bomb.c", nil)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 4, 32768))

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if !containsAny(combinedLower, []string{"inode bomb completed", "no space left", "disk quota exceeded"}) {
		t.Fatalf("inode exhaustion test produced unexpected result; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
	mustCreateAndDeleteTempFile(t)
}

func testPrivilegedSyscallDenied(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "try_reboot.c", nil)
	resp := callSimpleExecute(t, h.baseURL, buildCRequest(code, 3, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "privileged reboot syscall")

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "reboot succeeded unexpectedly") {
		t.Fatalf("privileged reboot syscall unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combinedLower, []string{"operation not permitted", "permission denied", "bad system call", "killed", "hangup"}) {
		t.Fatalf("expected privileged syscall denial signal was not observed; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func testFilesystemIsolationPython3(t *testing.T, h integrationHarness) {
	writeCode := mustLoadSampleCode(t, "file_privacy_write_read.py", nil)
	writeResp := callSimpleExecute(t, h.baseURL, buildPython3Request(writeCode, 4, 32768))
	if !strings.Contains(writeResp.Output, "SecretData123") {
		t.Fatalf("step A did not return secret in stdout; stdout=%q stderr=%q", writeResp.Output, writeResp.Error)
	}

	readCode := mustLoadSampleCode(t, "file_privacy_read_only.py", nil)
	readResp := callSimpleExecute(t, h.baseURL, buildPython3Request(readCode, 4, 32768))
	if !strings.Contains(strings.ToLower(readResp.Error), "no such file or directory") {
		t.Fatalf("step B unexpectedly accessed file from another sandbox; stdout=%q stderr=%q", readResp.Output, readResp.Error)
	}
}

func testDiskCleanupPython3(t *testing.T, h integrationHarness) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := mustLoadSampleCode(t, "disk_spammer.py", nil)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 2, 32768))
	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "memory limit exceeded"}) {
		t.Fatalf("disk spammer was not terminated as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
}

func testForkBombContainmentPython3(t *testing.T, h integrationHarness) {
	bombCode := mustLoadSampleCode(t, "fork_bomb.py", nil)
	_ = callSimpleExecute(t, h.baseURL, buildPython3Request(bombCode, 2, 32768))

	helloCode := mustLoadSampleCode(t, "hello_world.py", nil)
	helloResp := callSimpleExecute(t, h.baseURL, buildPython3Request(helloCode, 4, 32768))
	if !strings.Contains(helloResp.Output, "Hello World") {
		t.Fatalf("follow-up request failed after fork bomb; stdout=%q stderr=%q", helloResp.Output, helloResp.Error)
	}
	if strings.Contains(strings.ToLower(helloResp.Error), "resource temporarily unavailable") {
		t.Fatalf("follow-up request indicates host PID exhaustion; stderr=%q", helloResp.Error)
	}
}

func testNetworkIsolationPython3(t *testing.T, h integrationHarness) {
	replacements := map[string]string{"__API_PORT__": strconv.Itoa(h.apiPort)}
	code := mustLoadSampleCode(t, "network_localhost_bridge.py", replacements)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 3, 32768))

	combined := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combined, "connected") {
		t.Fatalf("localhost bridge unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combined, []string{"connection refused", "network is unreachable", "no route to host"}) {
		t.Fatalf("network isolation did not produce expected connect error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func testMemoryHardLimitPython3(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "memory_bomb.py", nil)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 3, 16384))
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

func testIOFloodResiliencePython3(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "io_spam.py", nil)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 1, 32768))

	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "killed", "memory limit exceeded"}) {
		t.Fatalf("io flood did not terminate with expected error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	if strings.TrimSpace(resp.Error) == "" {
		t.Fatalf("io flood returned empty stderr; expected bounded but non-empty error output")
	}

	const maxErrorBytes = (1 << 20) + 4096
	if len(resp.Error) > maxErrorBytes {
		t.Fatalf("stderr exceeded resilience cap: got=%d bytes cap=%d", len(resp.Error), maxErrorBytes)
	}

	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "io flood request")
}

func testSignalTrapTimeoutPython3(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "signal_trap.py", nil)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 1, 32768))

	if !strings.Contains(strings.ToLower(resp.Error), "execution timed out") {
		t.Fatalf("signal trap did not timeout as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	assertDurationWindow(t, resp.CPUTime, 900*time.Millisecond, 2500*time.Millisecond, "signal trap timeout")
}

func testOrphanReapingPython3(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "orphan_maker.py", nil)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 2, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "orphan maker request")

	time.Sleep(400 * time.Millisecond)
	if cnt := countProcessesByComm(t, "orphanpygc"); cnt > 0 {
		t.Fatalf("orphan grandchild leaked to host after request completion: count=%d stderr=%q", cnt, resp.Error)
	}
}

func testInodeExhaustionPython3(t *testing.T, h integrationHarness) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := mustLoadSampleCode(t, "inode_bomb.py", nil)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 4, 32768))

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if !containsAny(combinedLower, []string{"inode bomb completed", "no space left", "disk quota exceeded"}) {
		t.Fatalf("inode exhaustion test produced unexpected result; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
	mustCreateAndDeleteTempFile(t)
}

func testPrivilegedSyscallDeniedPython3(t *testing.T, h integrationHarness) {
	code := mustLoadSampleCode(t, "try_reboot.py", nil)
	resp := callSimpleExecute(t, h.baseURL, buildPython3Request(code, 3, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "privileged reboot syscall")

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "reboot succeeded unexpectedly") {
		t.Fatalf("privileged reboot syscall unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combinedLower, []string{"operation not permitted", "permission denied", "bad system call", "killed", "hangup"}) {
		t.Fatalf("expected privileged syscall denial signal was not observed; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func testFilesystemIsolationForLanguage(t *testing.T, h integrationHarness, language, writeFile, readFile string) {
	writeCode := mustLoadSampleCode(t, writeFile, nil)
	writeResp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, writeCode, 4, 32768))
	if !strings.Contains(writeResp.Output, "SecretData123") {
		t.Fatalf("step A did not return secret in stdout; stdout=%q stderr=%q", writeResp.Output, writeResp.Error)
	}

	readCode := mustLoadSampleCode(t, readFile, nil)
	readResp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, readCode, 4, 32768))
	if !strings.Contains(strings.ToLower(readResp.Error), "no such file or directory") {
		t.Fatalf("step B unexpectedly accessed file from another sandbox; stdout=%q stderr=%q", readResp.Output, readResp.Error)
	}
}

func testDiskCleanupForLanguage(t *testing.T, h integrationHarness, language, payload string) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := mustLoadSampleCode(t, payload, nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 2, 32768))
	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "memory limit exceeded"}) {
		t.Fatalf("disk spammer was not terminated as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
}

func testForkBombContainmentForLanguage(t *testing.T, h integrationHarness, language, bombFile, helloFile string) {
	bombCode := mustLoadSampleCode(t, bombFile, nil)
	_ = callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, bombCode, 2, 32768))

	helloCode := mustLoadSampleCode(t, helloFile, nil)
	helloResp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, helloCode, 4, 32768))
	if !strings.Contains(helloResp.Output, "Hello World") {
		t.Fatalf("follow-up request failed after fork bomb; stdout=%q stderr=%q", helloResp.Output, helloResp.Error)
	}
	if strings.Contains(strings.ToLower(helloResp.Error), "resource temporarily unavailable") {
		t.Fatalf("follow-up request indicates host PID exhaustion; stderr=%q", helloResp.Error)
	}
}

func testNetworkIsolationForLanguage(t *testing.T, h integrationHarness, language, payload string) {
	replacements := map[string]string{"__API_PORT__": strconv.Itoa(h.apiPort)}
	code := mustLoadSampleCode(t, payload, replacements)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 3, 32768))

	combined := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combined, "connected") {
		t.Fatalf("localhost bridge unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combined, []string{"connection refused", "network is unreachable", "no route to host"}) {
		t.Fatalf("network isolation did not produce expected connect error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
}

func testMemoryHardLimitForLanguage(t *testing.T, h integrationHarness, language, payload string) {
	code := mustLoadSampleCode(t, payload, nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 3, 16384))
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

func testIOFloodResilienceForLanguage(t *testing.T, h integrationHarness, language, payload string) {
	code := mustLoadSampleCode(t, payload, nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 1, 32768))

	stderrLower := strings.ToLower(resp.Error)
	if !containsAny(stderrLower, []string{"execution timed out", "killed", "memory limit exceeded"}) {
		t.Fatalf("io flood did not terminate with expected error; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	if strings.TrimSpace(resp.Error) == "" {
		t.Fatalf("io flood returned empty stderr; expected bounded but non-empty error output")
	}

	const maxErrorBytes = (1 << 20) + 4096
	if len(resp.Error) > maxErrorBytes {
		t.Fatalf("stderr exceeded resilience cap: got=%d bytes cap=%d", len(resp.Error), maxErrorBytes)
	}

	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "io flood request")
}

func testSignalTrapTimeoutForLanguage(t *testing.T, h integrationHarness, language, payload string) {
	code := mustLoadSampleCode(t, payload, nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 1, 32768))

	if !strings.Contains(strings.ToLower(resp.Error), "execution timed out") {
		t.Fatalf("signal trap did not timeout as expected; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	assertDurationWindow(t, resp.CPUTime, 900*time.Millisecond, 2500*time.Millisecond, "signal trap timeout")
}

func testOrphanReapingForLanguage(t *testing.T, h integrationHarness, language, payload, orphanName string) {
	code := mustLoadSampleCode(t, payload, nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 2, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "orphan maker request")

	time.Sleep(400 * time.Millisecond)
	if cnt := countProcessesByComm(t, orphanName); cnt > 0 {
		t.Fatalf("orphan grandchild leaked to host after request completion: count=%d stderr=%q", cnt, resp.Error)
	}
}

func testInodeExhaustionForLanguage(t *testing.T, h integrationHarness, language, payload string) {
	beforeFree := mustFreeBytes(t, os.TempDir())
	beforeCount := countSandboxTempDirs(t)

	code := mustLoadSampleCode(t, payload, nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 4, 32768))

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if !containsAny(combinedLower, []string{"inode bomb completed", "no space left", "disk quota exceeded"}) {
		t.Fatalf("inode exhaustion test produced unexpected result; stdout=%q stderr=%q", resp.Output, resp.Error)
	}

	time.Sleep(400 * time.Millisecond)
	assertDiskReclaimed(t, beforeFree, mustFreeBytes(t, os.TempDir()))
	assertNoSandboxLeak(t, beforeCount, countSandboxTempDirs(t))
	mustCreateAndDeleteTempFile(t)
}

func testPrivilegedSyscallDeniedForLanguage(t *testing.T, h integrationHarness, language, payload string) {
	code := mustLoadSampleCode(t, payload, nil)
	resp := callSimpleExecute(t, h.baseURL, buildLanguageRequest(language, code, 3, 32768))
	assertDurationNotExcessive(t, resp.CPUTime, 3*time.Second, "privileged reboot syscall")

	combinedLower := strings.ToLower(resp.Output + "\n" + resp.Error)
	if strings.Contains(combinedLower, "reboot succeeded unexpectedly") {
		t.Fatalf("privileged reboot syscall unexpectedly succeeded; stdout=%q stderr=%q", resp.Output, resp.Error)
	}
	if !containsAny(combinedLower, []string{"operation not permitted", "permission denied", "bad system call", "killed", "hangup"}) {
		t.Fatalf("expected privileged syscall denial signal was not observed; stdout=%q stderr=%q", resp.Output, resp.Error)
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
	return buildLanguageRequest("c", code, timeoutSec, maxMemoryKB)
}

func buildPython3Request(code string, timeoutSec, maxMemoryKB uint) simpleExecuteRequest {
	return buildLanguageRequest("python3", code, timeoutSec, maxMemoryKB)
}

func buildCppRequest(code string, timeoutSec, maxMemoryKB uint) simpleExecuteRequest {
	return buildLanguageRequest("cpp", code, timeoutSec, maxMemoryKB)
}

func buildJavaRequest(code string, timeoutSec, maxMemoryKB uint) simpleExecuteRequest {
	return buildLanguageRequest("java", code, timeoutSec, maxMemoryKB)
}

func buildLanguageRequest(language, code string, timeoutSec, maxMemoryKB uint) simpleExecuteRequest {
	return simpleExecuteRequest{Language: language, Code: code, Timeout: timeoutSec, MaxMemory: maxMemoryKB}
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

func countProcessesByComm(t *testing.T, expected string) int {
	t.Helper()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Fatalf("failed to read /proc: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() || !isNumeric(entry.Name()) {
			continue
		}
		if hasProcessComm(entry, expected) {
			count++
		}
	}
	return count
}

func hasProcessComm(entry fs.DirEntry, expected string) bool {
	commPath := filepath.Join("/proc", entry.Name(), "comm")
	content, err := os.ReadFile(commPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(content)) == expected
}

func isNumeric(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func assertDurationWindow(t *testing.T, durationRaw string, min, max time.Duration, label string) {
	t.Helper()
	d, err := time.ParseDuration(durationRaw)
	if err != nil {
		t.Fatalf("failed to parse cpu_time for %s (%q): %v", label, durationRaw, err)
	}
	if d < min || d > max {
		t.Fatalf("%s duration out of range: got=%s min=%s max=%s", label, d, min, max)
	}
}

func assertDurationNotExcessive(t *testing.T, durationRaw string, max time.Duration, label string) {
	t.Helper()
	d, err := time.ParseDuration(durationRaw)
	if err != nil {
		t.Fatalf("failed to parse cpu_time for %s (%q): %v", label, durationRaw, err)
	}
	if d > max {
		t.Fatalf("%s took too long: got=%s max=%s", label, d, max)
	}
}

func mustCreateAndDeleteTempFile(t *testing.T) {
	t.Helper()
	f, err := os.CreateTemp("", "inode-health-*")
	if err != nil {
		t.Fatalf("host temp filesystem unhealthy after inode test: %v", err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp file %s: %v", name, err)
	}
	if err := os.Remove(name); err != nil {
		t.Fatalf("failed to remove temp file %s: %v", name, err)
	}
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
