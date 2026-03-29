package dispatcher

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/executor"
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
	"fmt"
)

var (
	reserveRAM = resourcemanager.ReserveRAM
	releaseRAM = resourcemanager.ReleaseRAM
	execute    = executor.Execute
)

func normalizeRequest(req models.Request) models.Request {
	if req.Timeout == 0 {
		req.Timeout = config.Config.Limits.DefaultTimeout
	} else if req.Timeout > config.Config.Limits.MaxTimeout {
		req.Timeout = config.Config.Limits.MaxTimeout
	}

	if req.MemoryLimit == 0 {
		req.MemoryLimit = config.Config.Limits.DefaultMemoryLimit
	} else if req.MemoryLimit > config.Config.Limits.MaxMemoryLimit {
		req.MemoryLimit = config.Config.Limits.MaxMemoryLimit
	}

	return req
}

func executeWithReservedRAM(req models.Request) (models.Response, error) {
	defer func() {
		releaseRAM(req.MemoryLimit)
	}()
	return execute(req)
}

func cannotReserveResponse() (models.Response, error) {
	return models.Response{
		Stdout:        "",
		Stderr:        "Server is overloaded, please try again later",
		ExecutionTime: 0,
	}, fmt.Errorf("failed to reserve RAM")
}

func Dispatch(req models.Request) (models.Response, error) {
	req = normalizeRequest(req)

	if req.MemoryLimit > config.Config.Globals.RAM_LIMIT {
		return cannotReserveResponse()
	}

	if !reserveRAM(req.MemoryLimit) {
		return cannotReserveResponse()
	}

	return executeWithReservedRAM(req)
}
