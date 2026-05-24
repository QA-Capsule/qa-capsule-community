package integrations

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunRemediationAsync_doesNotDropJobs(t *testing.T) {
	remediationOnce = sync.Once{}
	remediationJobs = nil
	initRemediationWorkers()

	const n = 80
	var done int32
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			RunRemediationAsync(func() {
				time.Sleep(5 * time.Millisecond)
				atomic.AddInt32(&done, 1)
			})
		}()
	}

	wg.Wait()
	deadline := time.Now().Add(30 * time.Second)
	for atomic.LoadInt32(&done) < n && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&done); got != n {
		t.Fatalf("expected %d jobs completed, got %d (jobs must not be dropped)", n, got)
	}
}
