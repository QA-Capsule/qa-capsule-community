package core

import "time"

// IncidentArtifact is a file linked to an incident (trace zip, screenshot, video).
type IncidentArtifact struct {
	ID              int64     `json:"id"`
	IncidentID      int64     `json:"incident_id"`
	FileName        string    `json:"file_name"`
	ContentType     string    `json:"content_type"`
	SizeBytes       int64     `json:"size_bytes"`
	StorageProvider string    `json:"storage_provider"`
	StoragePath     string    `json:"storage_path"`
	CreatedAt       time.Time `json:"created_at"`
}

// TestExecutionMetric stores timing history for performance regression detection.
type TestExecutionMetric struct {
	ProjectName     string
	TestName        string
	Fingerprint     string
	ExecutionTimeMs int64
	Status          string
}
