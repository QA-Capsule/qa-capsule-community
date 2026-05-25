package core

import (
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/QA-Capsule/qa-capsule-community/pkg/quarantine"
)

// ErrIngestQueueFull is returned when the ingest buffer is saturated.
var ErrIngestQueueFull = errors.New("ingest queue full")

// IngestBatchJob is a unit of webhook work processed asynchronously.
type IngestBatchJob struct {
	Config         Config
	ProjectName    string
	PipelineRunID  string
	CommitSHA      string
	Branch         string
	Flags          ExecutionFlags
	Alerts         []UnifiedAlert
	Report         UnifiedExecutionReport
	AlertContext   map[string]string
	AllowedPlugins map[string]bool
}

var (
	ingestQueue       chan IngestBatchJob
	ingestWorkersOnce bool
	ingestQueueDepth  atomic.Int64
	ingestDropped     atomic.Int64
	ingestProcessed   atomic.Int64
)

func ingestQueueSize() int {
	if v := os.Getenv("QACAPSULE_INGEST_QUEUE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 256
}

func ingestWorkerCount() int {
	if v := os.Getenv("QACAPSULE_INGEST_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 4
}

// StartIngestWorkers starts the buffered ingest pipeline (idempotent).
func StartIngestWorkers() {
	if ingestWorkersOnce {
		return
	}
	ingestWorkersOnce = true
	size := ingestQueueSize()
	ingestQueue = make(chan IngestBatchJob, size)
	workers := ingestWorkerCount()
	for i := 0; i < workers; i++ {
		go ingestWorker(i)
	}
	slog.Info("ingest queue started", "buffer", size, "workers", workers)
}

// EnqueueIngest submits a webhook batch for asynchronous persistence.
func EnqueueIngest(job IngestBatchJob) error {
	if ingestQueue == nil {
		StartIngestWorkers()
	}
	select {
	case ingestQueue <- job:
		ingestQueueDepth.Add(1)
		return nil
	default:
		ingestDropped.Add(1)
		return ErrIngestQueueFull
	}
}

// IngestQueueDepth returns jobs waiting in the buffer (best-effort gauge).
func IngestQueueDepth() int64 {
	return ingestQueueDepth.Load()
}

// IngestDroppedTotal returns batches rejected because the queue was full.
func IngestDroppedTotal() int64 {
	return ingestDropped.Load()
}

// IngestProcessedTotal returns batches completed by workers.
func IngestProcessedTotal() int64 {
	return ingestProcessed.Load()
}

// PipelineOutcomeFromReport marks the run failed when failures exist even if incidents were deduplicated.
func PipelineOutcomeFromReport(processed int, report UnifiedExecutionReport) string {
	if processed > 0 {
		return "failure"
	}
	if report.Summary.Failed > 0 || report.Summary.Flaky > 0 {
		return "failure"
	}
	for _, tc := range report.Tests {
		st := strings.ToLower(strings.TrimSpace(tc.Status))
		if st == "fail" || st == "flaky" {
			return "failure"
		}
	}
	return "success"
}

func ingestWorker(workerID int) {
	for job := range ingestQueue {
		processIngestBatch(job, workerID)
		ingestQueueDepth.Add(-1)
		ingestProcessed.Add(1)
	}
}

func processIngestBatch(job IngestBatchJob, workerID int) {
	alerts := job.Alerts
	if len(alerts) == 0 {
		alerts = AlertsFromReport(job.Report, nil)
	}
	processed := 0
	created := 0
	skipped := 0
	var ingested []IngestedCase
	seenTests := make(map[string]struct{})
	for _, alert := range alerts {
		res := ProcessAlert(job.Config, job.ProjectName, job.PipelineRunID, alert, job.AlertContext, job.AllowedPlugins)
		if res.IncidentID > 0 {
			created++
			ingested = append(ingested, IngestedCase{
				Fingerprint: res.Fingerprint,
				FinalName:   res.FinalName,
				IncidentID:  res.IncidentID,
				Flaky:       res.Flaky,
			})
		}
		if res.Skipped {
			skipped++
		}
		if !res.Quarantined && !res.Skipped {
			processed++
		}
		if !isPassStatus(alert.Status) {
			key := quarantine.NormalizeTestName(alert.Name)
			if key != "" {
				seenTests[key] = struct{}{}
			}
		}
	}
	for testKey := range seenTests {
		reconcileFlakyTagsForTest(job.ProjectName, testKey)
	}
	outcome := PipelineOutcomeFromReport(processed, job.Report)
	ctx := IngestExecutionContext{
		ProjectName:   job.ProjectName,
		PipelineRunID: job.PipelineRunID,
		CommitSHA:     job.CommitSHA,
		Branch:        job.Branch,
		Flags:         job.Flags,
	}
	recordReportTestMetrics(job.ProjectName, job.Report)
	if err := FinalizePipelineExecution(ctx, outcome, job.Report, ingested); err != nil {
		slog.Error("finalize pipeline failed", "worker", workerID, "project", job.ProjectName, "run", job.PipelineRunID, "error", err)
	}
	slog.Info("ingest batch complete",
		"worker", workerID,
		"project", job.ProjectName,
		"run", job.PipelineRunID,
		"alerts_in", len(alerts),
		"incidents_created", created,
		"skipped_duplicate", skipped,
		"matrix_failures", job.Report.Summary.Failed,
	)
}

// recordReportTestMetrics stores pass/fail samples from the full matrix for flaky oscillation detection.
func recordReportTestMetrics(projectName string, report UnifiedExecutionReport) {
	if DB == nil || projectName == "" {
		return
	}
	for _, tc := range report.Tests {
		st := strings.ToLower(strings.TrimSpace(tc.Status))
		if st == "skip" || st == "skipped" {
			continue
		}
		metricStatus := "CRITICAL"
		if st == "pass" || st == "passed" {
			metricStatus = "PASSED"
		}
		identityFP := quarantine.TestIdentityFingerprint(projectName, tc.Name)
		RecordExecutionMetric(projectName, quarantine.NormalizeTestName(tc.Name), identityFP, tc.DurationMs, metricStatus)
	}
}
