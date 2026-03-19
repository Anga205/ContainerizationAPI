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
	resourcemanager.ReserveRAM(req.MemoryLimit) // TODO: handle error and implement timeout for reservation
	defer resourcemanager.ReleaseRAM(req.MemoryLimit)
	return executor.Execute(req)
}
