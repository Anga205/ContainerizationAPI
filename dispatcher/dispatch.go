package dispatcher

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/executor"
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
	"fmt"
)

func dispatch(req models.Request) (models.Response, error) {
	if req.MemoryLimit > config.Config.Globals.RAM_LIMIT {
		return models.Response{
			Stdout:        "",
			Stderr:        "Requested memory limit exceeds global RAM limit",
			ExecutionTime: 0,
		}, nil
	}
	if !config.Config.Globals.ENABLE_QUEUE {
		if !resourcemanager.ReserveRAM(req.MemoryLimit) { // TODO: handle condition for queueing requests when RAM is not available
			return models.Response{
				Stdout:        "",
				Stderr:        "Resource limit reached, please try again later",
				ExecutionTime: 0,
			}, fmt.Errorf("Failed to reserve RAM")
		}
		defer resourcemanager.ReleaseRAM(req.MemoryLimit)
		return executor.Execute(req)
	} else {
		// TODO: Implement queueing logic
		return models.Response{
			Stdout:        "",
			Stderr:        "Queueing is not implemented yet",
			ExecutionTime: 0,
		}, nil
	}
}
