package storage

import (
	"context"
	"fmt"
	"io"
)

// S3Config holds future AWS S3 settings (not implemented in community edition).
type S3Config struct {
	Bucket    string
	Region    string
	Prefix    string
	AccessKey string
	SecretKey string
}

// S3Provider is a placeholder for enterprise object storage.
type S3Provider struct {
	Config S3Config
}

func NewS3Provider(cfg S3Config) *S3Provider {
	return &S3Provider{Config: cfg}
}

func (p *S3Provider) Save(ctx context.Context, incidentID int64, safeName string, _ io.Reader, _ int64, _ string) (StoredObject, error) {
	return StoredObject{}, fmt.Errorf("S3 storage not enabled: configure STORAGE_PROVIDER=local or implement S3 in enterprise build (bucket=%q)", p.Config.Bucket)
}

func (p *S3Provider) Open(ctx context.Context, obj StoredObject) (io.ReadCloser, error) {
	return nil, fmt.Errorf("S3 storage not enabled (path=%q)", obj.Path)
}
