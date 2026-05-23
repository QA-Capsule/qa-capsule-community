package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalProvider stores files under baseDir/incident_{id}/.
type LocalProvider struct {
	BaseDir string
}

func NewLocalProvider(baseDir string) (*LocalProvider, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, err
	}
	return &LocalProvider{BaseDir: baseDir}, nil
}

func (p *LocalProvider) Save(ctx context.Context, incidentID int64, safeName string, r io.Reader, size int64, _ string) (StoredObject, error) {
	dir := filepath.Join(p.BaseDir, fmt.Sprintf("incident_%d", incidentID))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return StoredObject{}, err
	}
	dest := filepath.Join(dir, safeName)
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return StoredObject{}, err
	}
	defer f.Close()

	written, err := copyWithContext(ctx, f, r)
	if err != nil {
		os.Remove(dest)
		return StoredObject{}, err
	}
	rel := filepath.ToSlash(filepath.Join(fmt.Sprintf("incident_%d", incidentID), safeName))
	return StoredObject{Provider: "local", Path: rel, Size: written}, nil
}

func (p *LocalProvider) Open(_ context.Context, obj StoredObject) (io.ReadCloser, error) {
	return os.Open(filepath.Join(p.BaseDir, filepath.FromSlash(obj.Path)))
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	type result struct {
		n   int64
		err error
	}
	ch := make(chan result, 1)
	go func() {
		n, err := io.Copy(dst, src)
		ch <- result{n, err}
	}()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-ch:
		return res.n, res.err
	}
}
