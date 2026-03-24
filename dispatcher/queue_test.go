package dispatcher

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/models"
	"sync"
	"testing"
	"time"
)

func TestDispatchQueueingFIFO(t *testing.T) {
	originalQueue := config.Config.Globals.ENABLE_QUEUE
	originalGlobalRAM := config.Config.Globals.RAM_LIMIT
	originalDefaultMem := config.Config.Limits.DefaultMemoryLimit
	originalMaxMem := config.Config.Limits.MaxMemoryLimit
	originalReserve := reserveRAM
	originalRelease := releaseRAM
	originalExecute := execute
	t.Cleanup(func() {
		config.Config.Globals.ENABLE_QUEUE = originalQueue
		config.Config.Globals.RAM_LIMIT = originalGlobalRAM
		config.Config.Limits.DefaultMemoryLimit = originalDefaultMem
		config.Config.Limits.MaxMemoryLimit = originalMaxMem
		reserveRAM = originalReserve
		releaseRAM = originalRelease
		execute = originalExecute
	})

	config.Config.Globals.ENABLE_QUEUE = true
	config.Config.Globals.RAM_LIMIT = 1
	config.Config.Limits.DefaultMemoryLimit = 1
	config.Config.Limits.MaxMemoryLimit = 1
	resetQueueStateForTests()

	var memLock sync.Mutex
	available := uint(1)
	reserveRAM = func(amount uint) bool {
		memLock.Lock()
		defer memLock.Unlock()
		if amount > available {
			return false
		}
		available -= amount
		return true
	}
	releaseRAM = func(amount uint) {
		memLock.Lock()
		available += amount
		memLock.Unlock()
	}

	var orderLock sync.Mutex
	startOrder := make([]string, 0, 3)
	allowFirstToFinish := make(chan struct{})
	execute = func(req models.Request) (models.Response, error) {
		orderLock.Lock()
		startOrder = append(startOrder, req.Code)
		orderLock.Unlock()

		if req.Code == "A" {
			<-allowFirstToFinish
		}

		return models.Response{Stdout: req.Code}, nil
	}

	resultA := make(chan models.Response, 1)
	errA := make(chan error, 1)
	go func() {
		resp, err := Dispatch(models.Request{Code: "A", MemoryLimit: 1})
		resultA <- resp
		errA <- err
	}()

	// Wait until the first request starts and holds RAM.
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		orderLock.Lock()
		started := len(startOrder) == 1 && startOrder[0] == "A"
		orderLock.Unlock()
		if started {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first request did not start in time")
		}
		time.Sleep(5 * time.Millisecond)
	}

	resultB := make(chan models.Response, 1)
	errB := make(chan error, 1)
	go func() {
		resp, err := Dispatch(models.Request{Code: "B", MemoryLimit: 1})
		resultB <- resp
		errB <- err
	}()

	// Ensure B is at the front of the queue before C arrives.
	queueDeadline := time.Now().Add(500 * time.Millisecond)
	for {
		queueLock.Lock()
		ready := len(requestQueue) == 1 && requestQueue[0].req.Code == "B"
		queueLock.Unlock()
		if ready {
			break
		}
		if time.Now().After(queueDeadline) {
			t.Fatal("B was not queued in time")
		}
		time.Sleep(5 * time.Millisecond)
	}

	resultC := make(chan models.Response, 1)
	errC := make(chan error, 1)
	go func() {
		resp, err := Dispatch(models.Request{Code: "C", MemoryLimit: 1})
		resultC <- resp
		errC <- err
	}()

	// B and C should be queued while A is still running.
	select {
	case <-resultB:
		t.Fatal("B should have stayed queued while A holds RAM")
	case <-time.After(40 * time.Millisecond):
	}

	select {
	case <-resultC:
		t.Fatal("C should have stayed queued while A holds RAM")
	case <-time.After(40 * time.Millisecond):
	}

	close(allowFirstToFinish)

	if err := <-errA; err != nil {
		t.Fatalf("A returned error: %v", err)
	}
	if resp := <-resultA; resp.Stdout != "A" {
		t.Fatalf("unexpected A stdout: %q", resp.Stdout)
	}

	if err := <-errB; err != nil {
		t.Fatalf("B returned error: %v", err)
	}
	if resp := <-resultB; resp.Stdout != "B" {
		t.Fatalf("unexpected B stdout: %q", resp.Stdout)
	}

	if err := <-errC; err != nil {
		t.Fatalf("C returned error: %v", err)
	}
	if resp := <-resultC; resp.Stdout != "C" {
		t.Fatalf("unexpected C stdout: %q", resp.Stdout)
	}

	orderLock.Lock()
	defer orderLock.Unlock()
	if len(startOrder) != 3 {
		t.Fatalf("unexpected execution count: got=%d order=%v", len(startOrder), startOrder)
	}
	if startOrder[0] != "A" || startOrder[1] != "B" || startOrder[2] != "C" {
		t.Fatalf("queue did not execute in FIFO order: %v", startOrder)
	}
}
