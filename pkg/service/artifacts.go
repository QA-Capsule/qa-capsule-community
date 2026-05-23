package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
	"github.com/QA-Capsule/qa-capsule-community/pkg/storage"
)

const MaxArtifactBytes = 50 << 20 // 50MB

var allowedArtifactExt = map[string]bool{
	".zip": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webm": true, ".mp4": true, ".trace": true,
}

// ArtifactService handles validation and async persistence.
type ArtifactService struct {
	Provider storage.Provider
}

func NewArtifactService(p storage.Provider) *ArtifactService {
	return &ArtifactService{Provider: p}
}

func (s *ArtifactService) ValidateUpload(fileName string, size int64) error {
	if size <= 0 {
		return fmt.Errorf("empty file")
	}
	if size > MaxArtifactBytes {
		return fmt.Errorf("file exceeds maximum size of 50MB")
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if !allowedArtifactExt[ext] {
		return fmt.Errorf("file extension %q is not allowed", ext)
	}
	return nil
}

// SaveBackground validates synchronously then persists in a goroutine (non-blocking).
func (s *ArtifactService) SaveBackground(incidentID int64, fileName, contentType string, data []byte) error {
	safeName := filepath.Base(fileName)
	size := int64(len(data))
	if err := s.ValidateUpload(safeName, size); err != nil {
		return err
	}
	if ok, err := core.IncidentExists(incidentID); err != nil || !ok {
		return fmt.Errorf("incident not found")
	}
	go func() {
		ctx := context.Background()
		stored, err := s.Provider.Save(ctx, incidentID, safeName, bytes.NewReader(data), size, contentType)
		if err != nil {
			slog.Error("artifact storage failed", "incident_id", incidentID, "error", err)
			return
		}
		rec := &core.IncidentArtifact{
			IncidentID:      incidentID,
			FileName:        safeName,
			ContentType:     contentType,
			SizeBytes:       stored.Size,
			StorageProvider: stored.Provider,
			StoragePath:     stored.Path,
		}
		id, err := core.InsertArtifact(rec)
		if err != nil {
			slog.Error("artifact db insert failed", "error", err)
			return
		}
		slog.Info("artifact stored", "artifact_id", id, "incident_id", incidentID, "file", safeName)
	}()
	return nil
}
