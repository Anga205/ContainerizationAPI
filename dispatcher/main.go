package dispatcher

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/executor"
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
)

func dispatch(req models.Request) (models.Response, error) {
	if req.MemoryLimit > config.Config.Globals.RAM_LIMIT {
		return models.Response{
			Stdout:        "",
			Stderr:        "Requested memory limit exceeds global RAM limit",
			ExecutionTime: 0,
		}, nil
	}
	if !resourcemanager.ReserveRAM(req.MemoryLimit) { // TODO: handle condition for queueing requests when RAM is not available
		return models.Response{
			Stdout:        "",
			Stderr:        "Failed to reserve RAM",
			ExecutionTime: 0,
		}, nil
	}
	defer resourcemanager.ReleaseRAM(req.MemoryLimit)
	return executor.Execute(req)
}
