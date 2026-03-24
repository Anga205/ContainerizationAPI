package dispatcher

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/executor"
	"CodeSandboxAPI/models"
	"CodeSandboxAPI/resourcemanager"
	"fmt"
	"sync"
)

type dispatchResult struct {
	response models.Response
	err      error
}

type queuedRequest struct {
	req    models.Request
	result chan dispatchResult
}

var (
	reserveRAM = resourcemanager.ReserveRAM
	releaseRAM = resourcemanager.ReleaseRAM
	execute    = executor.Execute

	queueLock    sync.Mutex
	queueCond    = sync.NewCond(&queueLock)
	requestQueue []queuedRequest
	workerActive bool
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
		notifyQueueWaiters()
	}()
	return execute(req)
}

func notifyQueueWaiters() {
	queueLock.Lock()
	queueCond.Broadcast()
	queueLock.Unlock()
}

func startQueueWorkerLocked() {
	if workerActive {
		return
	}
	workerActive = true
	go processQueue()
}

func processQueue() {
	for {
		queueLock.Lock()
		for len(requestQueue) == 0 {
			queueCond.Wait()
		}

		head := requestQueue[0]
		for !reserveRAM(head.req.MemoryLimit) {
			queueCond.Wait()
			head = requestQueue[0]
		}

		requestQueue = requestQueue[1:]
		queueLock.Unlock()

		resp, err := executeWithReservedRAM(head.req)
		head.result <- dispatchResult{response: resp, err: err}
	}
}

func queueOrExecute(req models.Request) (models.Response, error) {
	queueLock.Lock()
	if len(requestQueue) == 0 && reserveRAM(req.MemoryLimit) {
		queueLock.Unlock()
		return executeWithReservedRAM(req)
	}

	entry := queuedRequest{
		req:    req,
		result: make(chan dispatchResult, 1),
	}
	requestQueue = append(requestQueue, entry)
	startQueueWorkerLocked()
	queueCond.Broadcast()
	queueLock.Unlock()

	result := <-entry.result
	return result.response, result.err
}

func cannotReserveResponse() (models.Response, error) {
	return models.Response{
		Stdout:        "",
		Stderr:        "Resource limit reached, please try again later",
		ExecutionTime: 0,
	}, fmt.Errorf("failed to reserve RAM")
}

func Dispatch(req models.Request) (models.Response, error) {
	req = normalizeRequest(req)

	if req.MemoryLimit > config.Config.Globals.RAM_LIMIT {
		return cannotReserveResponse()
	}

	if !config.Config.Globals.ENABLE_QUEUE {
		if !reserveRAM(req.MemoryLimit) {
			return cannotReserveResponse()
		}
		return executeWithReservedRAM(req)
	}

	return queueOrExecute(req)
}

func resetQueueStateForTests() {
	queueLock.Lock()
	requestQueue = nil
	queueCond.Broadcast()
	queueLock.Unlock()
}
