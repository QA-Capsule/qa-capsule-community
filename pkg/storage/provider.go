package storage

import (
	"context"
	"io"
)

// StoredObject metadata returned after a successful save.
type StoredObject struct {
	Provider string
	Path     string // provider-specific key or relative path
	Size     int64
}

// Provider persists binary artifacts (local disk, S3, etc.).
type Provider interface {
	Save(ctx context.Context, incidentID int64, safeName string, r io.Reader, size int64, contentType string) (StoredObject, error)
	Open(ctx context.Context, obj StoredObject) (io.ReadCloser, error)
}
