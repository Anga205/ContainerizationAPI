package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	sandboxInitOnce   sync.Once
	sandboxInitBinary []byte
	sandboxInitErr    error
)

func ensureSandboxInitBinary(ws *sandboxWorkspace) error {
	if ws.sandboxInitHostPath == "" || ws.sandboxInitExecPath == "" {
		return nil
	}

	binary, err := loadSandboxInitBinary()
	if err != nil {
		return err
	}
	if err := os.WriteFile(ws.sandboxInitHostPath, binary, 0o755); err != nil {
		return fmt.Errorf("failed to install sandbox init binary: %w", err)
	}
	return nil
}

func loadSandboxInitBinary() ([]byte, error) {
	sandboxInitOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "codesandbox-init-build-*")
		if err != nil {
			sandboxInitErr = fmt.Errorf("failed to create sandbox init build dir: %w", err)
			return
		}
		defer os.RemoveAll(tempDir)

		sourcePath := filepath.Join(tempDir, "sandbox_init.c")
		binaryPath := filepath.Join(tempDir, "sandbox-init")

		if err := os.WriteFile(sourcePath, []byte(sandboxInitSource), 0o600); err != nil {
			sandboxInitErr = fmt.Errorf("failed to write sandbox init source: %w", err)
			return
		}

		compileCmd := exec.Command("gcc", "-O2", "-pipe", "-static", "-s", "-o", binaryPath, sourcePath)
		output, err := compileCmd.CombinedOutput()
		if err != nil {
			sandboxInitErr = fmt.Errorf("failed to compile sandbox init binary: %w: %s", err, string(output))
			return
		}

		sandboxInitBinary, sandboxInitErr = os.ReadFile(binaryPath)
		if sandboxInitErr != nil {
			sandboxInitErr = fmt.Errorf("failed to read compiled sandbox init binary: %w", sandboxInitErr)
		}
	})

	if sandboxInitErr != nil {
		return nil, sandboxInitErr
	}
	return sandboxInitBinary, nil
}

const sandboxInitSource = `#define _GNU_SOURCE
#include <errno.h>
#include <linux/audit.h>
#include <linux/filter.h>
#include <linux/seccomp.h>
#include <linux/unistd.h>
#include <stddef.h>
#include <stdio.h>
#include <sys/prctl.h>
#include <unistd.h>

#ifndef SECCOMP_RET_KILL_PROCESS
#define SECCOMP_RET_KILL_PROCESS SECCOMP_RET_KILL
#endif

#ifndef AUDIT_ARCH_X86_64
#define AUDIT_ARCH_X86_64 0xC000003E
#endif

#ifndef AUDIT_ARCH_AARCH64
#define AUDIT_ARCH_AARCH64 0xC00000B7
#endif

#ifndef AUDIT_ARCH_ARM
#define AUDIT_ARCH_ARM 0x40000028
#endif

#if defined(__x86_64__)
#define SANDBOX_AUDIT_ARCH AUDIT_ARCH_X86_64
#elif defined(__aarch64__)
#define SANDBOX_AUDIT_ARCH AUDIT_ARCH_AARCH64
#elif defined(__arm__)
#define SANDBOX_AUDIT_ARCH AUDIT_ARCH_ARM
#else
#error "unsupported architecture for seccomp filter"
#endif

#define DENY_SYSCALL(syscall_nr) \
    BPF_JUMP(BPF_JMP | BPF_JEQ | BPF_K, syscall_nr, 0, 1), \
    BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_ERRNO | (EPERM & SECCOMP_RET_DATA))

static int install_seccomp_filter(void) {
    struct sock_filter filter[] = {
        BPF_STMT(BPF_LD | BPF_W | BPF_ABS, (unsigned int)offsetof(struct seccomp_data, arch)),
                BPF_JUMP(BPF_JMP | BPF_JEQ | BPF_K, SANDBOX_AUDIT_ARCH, 1, 0),
        BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_KILL_PROCESS),
        BPF_STMT(BPF_LD | BPF_W | BPF_ABS, (unsigned int)offsetof(struct seccomp_data, nr)),
#ifdef __NR_mount
        DENY_SYSCALL(__NR_mount),
#endif
#ifdef __NR_umount
        DENY_SYSCALL(__NR_umount),
#endif
#ifdef __NR_umount2
        DENY_SYSCALL(__NR_umount2),
#endif
#ifdef __NR_unshare
        DENY_SYSCALL(__NR_unshare),
#endif
#ifdef __NR_setns
        DENY_SYSCALL(__NR_setns),
#endif
#ifdef __NR_pivot_root
        DENY_SYSCALL(__NR_pivot_root),
#endif
#ifdef __NR_chroot
        DENY_SYSCALL(__NR_chroot),
#endif
#ifdef __NR_init_module
        DENY_SYSCALL(__NR_init_module),
#endif
#ifdef __NR_finit_module
        DENY_SYSCALL(__NR_finit_module),
#endif
#ifdef __NR_delete_module
        DENY_SYSCALL(__NR_delete_module),
#endif
#ifdef __NR_kexec_load
        DENY_SYSCALL(__NR_kexec_load),
#endif
#ifdef __NR_kexec_file_load
        DENY_SYSCALL(__NR_kexec_file_load),
#endif
#ifdef __NR_reboot
        DENY_SYSCALL(__NR_reboot),
#endif
#ifdef __NR_syslog
        DENY_SYSCALL(__NR_syslog),
#endif
#ifdef __NR_nfsservctl
        DENY_SYSCALL(__NR_nfsservctl),
#endif
#ifdef __NR_quotactl
        DENY_SYSCALL(__NR_quotactl),
#endif
#ifdef __NR_swapon
        DENY_SYSCALL(__NR_swapon),
#endif
#ifdef __NR_swapoff
        DENY_SYSCALL(__NR_swapoff),
#endif
#ifdef __NR_setdomainname
        DENY_SYSCALL(__NR_setdomainname),
#endif
#ifdef __NR_sethostname
        DENY_SYSCALL(__NR_sethostname),
#endif
#ifdef __NR_iopl
        DENY_SYSCALL(__NR_iopl),
#endif
#ifdef __NR_iopl2
        DENY_SYSCALL(__NR_iopl2),
#endif
#ifdef __NR_ioperm
        DENY_SYSCALL(__NR_ioperm),
#endif
#ifdef __NR_perf_event_open
        DENY_SYSCALL(__NR_perf_event_open),
#endif
#ifdef __NR_ptrace
        DENY_SYSCALL(__NR_ptrace),
#endif
#ifdef __NR_process_vm_readv
        DENY_SYSCALL(__NR_process_vm_readv),
#endif
#ifdef __NR_process_vm_writev
        DENY_SYSCALL(__NR_process_vm_writev),
#endif
        BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_ALLOW),
    };

    struct sock_fprog program;
    program.len = (unsigned short)(sizeof(filter) / sizeof(filter[0]));
    program.filter = filter;

    if (prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) != 0) {
        return -1;
    }
    if (prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, &program) != 0) {
        return -1;
    }
    return 0;
}

int main(int argc, char **argv) {
    if (argc < 2) {
        fprintf(stderr, "sandbox-init: missing target command\n");
        return 126;
    }

    if (install_seccomp_filter() != 0) {
        perror("sandbox-init seccomp");
        return 126;
    }

    execvp(argv[1], &argv[1]);
    perror("sandbox-init execvp");
    if (errno == ENOENT) {
        return 127;
    }
    return 126;
}
`
