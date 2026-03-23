package executor

import (
	"CodeSandboxAPI/resourcemanager"
	"context"
	"os/exec"
)

func waitForSandbox(ctx context.Context, cmd *exec.Cmd, cg *resourcemanager.CgroupHandle) (error, bool) {
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return err, false
	case <-ctx.Done():
		_ = hardKillProcessGroup(cmd.Process.Pid)
		_ = cg.KillAll()
		return <-done, true
	}
}
