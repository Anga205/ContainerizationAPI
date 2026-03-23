package executor

import (
	"CodeSandboxAPI/resourcemanager"
	"sync"
	"sync/atomic"
	"time"
)

type memorySampler struct {
	peak   uint64
	cancel chan struct{}
	wg     sync.WaitGroup
}

func startMemorySampler(cg *resourcemanager.CgroupHandle) *memorySampler {
	s := &memorySampler{cancel: make(chan struct{})}
	s.wg.Add(1)
	go s.run(cg)
	return s
}

func (s *memorySampler) run(cg *resourcemanager.CgroupHandle) {
	defer s.wg.Done()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.cancel:
			return
		case <-ticker.C:
			current, err := cg.ReadMemoryCurrentBytes()
			if err == nil {
				updatePeak(&s.peak, current)
			}
		}
	}
}

func (s *memorySampler) stop() uint64 {
	close(s.cancel)
	s.wg.Wait()
	return atomic.LoadUint64(&s.peak)
}

func updatePeak(target *uint64, value uint64) {
	for {
		prev := atomic.LoadUint64(target)
		if value <= prev {
			return
		}
		if atomic.CompareAndSwapUint64(target, prev, value) {
			return
		}
	}
}
