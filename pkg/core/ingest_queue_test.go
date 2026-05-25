package core

import "testing"

func TestPipelineOutcomeFromReport(t *testing.T) {
	report := UnifiedExecutionReport{
		Tests: []UnifiedTestResult{{Status: "fail"}},
	}
	if got := PipelineOutcomeFromReport(0, report); got != "failure" {
		t.Fatalf("got %q want failure", got)
	}
	if got := PipelineOutcomeFromReport(0, UnifiedExecutionReport{}); got != "success" {
		t.Fatalf("got %q want success", got)
	}
}

func TestEnqueueIngest_queueFull(t *testing.T) {
	ingestWorkersOnce = true
	ingestQueue = make(chan IngestBatchJob, 1)
	ingestQueue <- IngestBatchJob{ProjectName: "blocked"}
	if err := EnqueueIngest(IngestBatchJob{ProjectName: "overflow"}); err != ErrIngestQueueFull {
		t.Fatalf("got %v want ErrIngestQueueFull", err)
	}
}
