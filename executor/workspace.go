package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

var syscallMount = syscall.Mount

type sandboxWorkspace struct {
	dir                 string
	sourcePath          string
	binaryPath          string
	sandboxInitHostPath string
	sandboxInitExecPath string
	compileCommand      []string
	runCommand          []string
	extraEnv            []string
	useChroot           bool
	runtimeMounts       []string
	mountedTargets      []string
}

func prepareWorkspace(language, code string) (sandboxWorkspace, error) {
	dir, err := os.MkdirTemp("", "codesandbox-*")
	if err != nil {
		return sandboxWorkspace{}, fmt.Errorf("failed to create sandbox dir: %w", err)
	}
	if err := os.Chown(dir, 65534, 65534); err != nil {
		_ = os.RemoveAll(dir)
		return sandboxWorkspace{}, fmt.Errorf("failed to set sandbox dir ownership: %w", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		_ = os.RemoveAll(dir)
		return sandboxWorkspace{}, fmt.Errorf("failed to set sandbox dir permissions: %w", err)
	}
	ws, err := buildWorkspacePlan(dir, language)
	if err != nil {
		_ = os.RemoveAll(dir)
		return sandboxWorkspace{}, err
	}
	if err := os.WriteFile(ws.sourcePath, []byte(code), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return sandboxWorkspace{}, fmt.Errorf("failed to write source: %w", err)
	}
	return ws, nil
}

func buildWorkspacePlan(dir, language string) (sandboxWorkspace, error) {
	normalized := normalizeLanguage(language)
	ws := sandboxWorkspace{
		dir:                 dir,
		binaryPath:          filepath.Join(dir, "program"),
		sandboxInitHostPath: filepath.Join(dir, "sandbox-init"),
		sandboxInitExecPath: "/sandbox-init",
	}

	switch normalized {
	case "c":
		ws.sourcePath = filepath.Join(dir, "main.c")
		ws.compileCommand = []string{"gcc", "-O2", "-pipe", "-std=c11", "-static", "-s", "-o", ws.binaryPath, ws.sourcePath}
		ws.runCommand = []string{"/program"}
		ws.useChroot = true
	case "cpp":
		ws.sourcePath = filepath.Join(dir, "main.cpp")
		ws.compileCommand = []string{"g++", "-O2", "-pipe", "-std=c++17", "-static", "-s", "-o", ws.binaryPath, ws.sourcePath}
		ws.runCommand = []string{"/program"}
		ws.useChroot = true
	case "java":
		javaBin, javaLibDir, javaHome, err := resolveJavaRuntime()
		if err != nil {
			return sandboxWorkspace{}, err
		}
		ws.sourcePath = filepath.Join(dir, "Main.java")
		ws.compileCommand = []string{"javac", ws.sourcePath}
		ws.runCommand = []string{javaBin, "-Xms8m", "-Xmx24m", "-cp", "/", "Main"}
		ws.extraEnv = []string{"LD_LIBRARY_PATH=" + javaLibDir, "JAVA_HOME=" + javaHome}
		ws.useChroot = true
		ws.runtimeMounts = defaultRuntimeMounts()
	case "python3":
		ws.sourcePath = filepath.Join(dir, "main.py")
		ws.runCommand = []string{"/usr/bin/python3", "/main.py"}
		ws.useChroot = true
		ws.runtimeMounts = defaultRuntimeMounts()
	default:
		return sandboxWorkspace{}, fmt.Errorf("unsupported language: %s", language)
	}

	return ws, nil
}

func normalizeLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "python", "py", "python3":
		return "python3"
	case "c++", "cpp":
		return "cpp"
	default:
		return strings.ToLower(strings.TrimSpace(language))
	}
}

func resolveJavaRuntime() (javaBin string, javaLibDir string, javaHome string, err error) {
	javaPath, lookErr := exec.LookPath("java")
	if lookErr != nil {
		return "", "", "", fmt.Errorf("java runtime not found: %w", lookErr)
	}

	resolvedJava, evalErr := filepath.EvalSymlinks(javaPath)
	if evalErr != nil {
		resolvedJava = javaPath
	}

	javaBin = resolvedJava
	javaHome = filepath.Clean(filepath.Join(filepath.Dir(resolvedJava), ".."))
	javaLibDir = filepath.Join(javaHome, "lib")
	return javaBin, javaLibDir, javaHome, nil
}

func defaultRuntimeMounts() []string {
	return []string{"/usr", "/lib", "/lib64", "/bin", "/etc"}
}

func prepareRuntimeMounts(ws *sandboxWorkspace) error {
	if !ws.useChroot || len(ws.runtimeMounts) == 0 {
		return nil
	}

	if err := syscallMount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		cleanupRuntimeMounts(ws)
		return fmt.Errorf("failed to mark root mount propagation private: %w", err)
	}

	for _, source := range ws.runtimeMounts {
		sourceInfo, err := os.Lstat(source)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to inspect runtime mount source %s: %w", source, err)
		}

		target := filepath.Join(ws.dir, strings.TrimPrefix(source, "/"))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to prepare mount target %s: %w", target, err)
		}

		if sourceInfo.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(source)
			if err != nil {
				cleanupRuntimeMounts(ws)
				return fmt.Errorf("failed to read runtime mount symlink %s: %w", source, err)
			}
			if err := os.RemoveAll(target); err != nil {
				cleanupRuntimeMounts(ws)
				return fmt.Errorf("failed to reset symlink target %s: %w", target, err)
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				cleanupRuntimeMounts(ws)
				return fmt.Errorf("failed to mirror symlink %s -> %s: %w", target, linkTarget, err)
			}
			continue
		}

		if err := os.MkdirAll(target, 0o755); err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to prepare mount target %s: %w", target, err)
		}
		if err := bindMountReadOnly(source, target); err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to prepare read-only bind mount %s -> %s: %w", source, target, err)
		}
		ws.mountedTargets = append(ws.mountedTargets, target)
	}

	return nil
}

func bindMountReadOnly(source, target string) error {
	if err := syscallMount(source, target, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}
	if err := syscallMount("", target, "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_REC, ""); err != nil {
		return err
	}
	return nil
}

func cleanupRuntimeMounts(ws *sandboxWorkspace) {
	for i := len(ws.mountedTargets) - 1; i >= 0; i-- {
		target := ws.mountedTargets[i]
		_ = syscall.Unmount(target, syscall.MNT_DETACH)
	}
	ws.mountedTargets = nil
}

func compileWorkspace(ws sandboxWorkspace) (string, bool) {
	if len(ws.compileCommand) == 0 {
		return "", true
	}

	cmd := exec.Command(ws.compileCommand[0], ws.compileCommand[1:]...)
	cmd.Dir = ws.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), false
	}
	return "", true
}
