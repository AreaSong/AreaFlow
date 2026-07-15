package artifact

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LocalBackend struct {
	root string
}

func NewLocalBackend(root string) LocalBackend {
	return LocalBackend{root: expandHome(strings.TrimSpace(root))}
}

func (b LocalBackend) Put(_ context.Context, relativePath string, content []byte, contentType string) (Stored, error) {
	if b.root == "" {
		return Stored{}, fmt.Errorf("artifact store root is required")
	}
	cleanRelative, err := safeRelativePath(relativePath)
	if err != nil {
		return Stored{}, err
	}
	target := filepath.Join(b.root, cleanRelative)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Stored{}, fmt.Errorf("create artifact directory: %w", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return Stored{}, fmt.Errorf("write artifact: %w", err)
	}
	return stored("local", target, content, contentType), nil
}

func (b LocalBackend) Get(_ context.Context, uri string) ([]byte, error) {
	path := strings.TrimSpace(uri)
	if path == "" {
		return nil, fmt.Errorf("local artifact URI is required")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read local artifact: %w", err)
	}
	return content, nil
}

func (b LocalBackend) Ping(_ context.Context) error {
	if b.root == "" {
		return fmt.Errorf("artifact store root is required")
	}
	if err := os.MkdirAll(b.root, 0o755); err != nil {
		return fmt.Errorf("prepare local artifact store: %w", err)
	}
	info, err := os.Stat(b.root)
	if err != nil {
		return fmt.Errorf("stat local artifact store: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("local artifact store root is not a directory")
	}
	return nil
}
