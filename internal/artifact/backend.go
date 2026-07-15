package artifact

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type Backend interface {
	Put(context.Context, string, []byte, string) (Stored, error)
	Get(context.Context, string) ([]byte, error)
}

func WriteConfigured(ctx context.Context, backend, root, relativePath string, content []byte, contentType string) (Stored, error) {
	cleanRelative, err := safeRelativePath(relativePath)
	if err != nil {
		return Stored{}, err
	}
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "local":
		return NewLocalBackend(root).Put(ctx, cleanRelative, content, contentType)
	case "s3", "object":
		s3Backend, err := DefaultS3Backend(ctx)
		if err != nil {
			return Stored{}, err
		}
		key := filepath.ToSlash(filepath.Join(strings.Trim(root, "/"), cleanRelative))
		return s3Backend.Put(ctx, key, content, contentType)
	default:
		return Stored{}, fmt.Errorf("unsupported artifact backend %q", backend)
	}
}

func ReadConfigured(ctx context.Context, backend, uri string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "local":
		return NewLocalBackend("").Get(ctx, uri)
	case "s3", "object":
		s3Backend, err := DefaultS3Backend(ctx)
		if err != nil {
			return nil, err
		}
		return s3Backend.Get(ctx, uri)
	default:
		return nil, fmt.Errorf("artifact backend %q does not expose content", backend)
	}
}

func safeRelativePath(path string) (string, error) {
	clean := filepath.Clean(path)
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact path must stay under artifact store root")
	}
	return clean, nil
}
