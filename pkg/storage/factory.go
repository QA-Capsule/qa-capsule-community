package storage

import (
	"fmt"
	"os"
	"strings"
)

// NewProviderFromEnv selects local or S3 (stub) from STORAGE_PROVIDER.
func NewProviderFromEnv(localBase string) (Provider, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_PROVIDER")))
	if provider == "" {
		provider = "local"
	}
	switch provider {
	case "local":
		if localBase == "" {
			localBase = "./data/artifacts"
		}
		return NewLocalProvider(localBase)
	case "s3":
		return NewS3Provider(S3Config{
			Bucket: os.Getenv("STORAGE_S3_BUCKET"),
			Region: os.Getenv("STORAGE_S3_REGION"),
			Prefix: os.Getenv("STORAGE_S3_PREFIX"),
		}), nil
	default:
		return nil, fmt.Errorf("unknown STORAGE_PROVIDER %q", provider)
	}
}
