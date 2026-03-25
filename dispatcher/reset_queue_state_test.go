package dispatcher

import "testing"

func TestResetQueueStateForTestsResetsWorkerFlag(t *testing.T) {
	queueLock.Lock()
	workerActive = true
	requestQueue = []queuedRequest{{}}
	queueLock.Unlock()

	resetQueueStateForTests()

	queueLock.Lock()
	defer queueLock.Unlock()
	if workerActive {
		t.Fatal("workerActive should be reset to false")
	}
	if len(requestQueue) != 0 {
		t.Fatalf("requestQueue should be empty, got len=%d", len(requestQueue))
	}
}
