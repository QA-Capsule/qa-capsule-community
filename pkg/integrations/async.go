package integrations

import (
	"log/slog"
	"sync"
)

// Bounded async remediation: fixed worker pool + blocking job queue (no silent drops).
const (
	defaultRemediationSlots = 32
	remediationQueueSize    = 512
)

var (
	remediationOnce sync.Once
	remediationJobs chan func()
)

func initRemediationWorkers() {
	remediationJobs = make(chan func(), remediationQueueSize)
	for i := 0; i < defaultRemediationSlots; i++ {
		go remediationWorker()
	}
	slog.Info("remediation worker pool started", "workers", defaultRemediationSlots, "queue", remediationQueueSize)
}

func remediationWorker() {
	for fn := range remediationJobs {
		if fn == nil {
			continue
		}
		runRemediationJob(fn)
	}
}

func runRemediationJob(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("remediation job panic", "recover", r)
		}
	}()
	fn()
}

// RunRemediationAsync enqueues fn for a worker. Blocks when the queue is full so jobs are never dropped.
func RunRemediationAsync(fn func()) {
	if fn == nil {
		return
	}
	remediationOnce.Do(initRemediationWorkers)
	remediationJobs <- fn
}
