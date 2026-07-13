package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Stored struct {
	Backend     string
	URI         string
	SHA256      string
	SizeBytes   int64
	ContentType string
}

func WriteLocal(root string, relativePath string, content []byte, contentType string) (Stored, error) {
	root = expandHome(strings.TrimSpace(root))
	if root == "" {
		return Stored{}, fmt.Errorf("artifact store root is required")
	}
	cleanRelative := filepath.Clean(relativePath)
	if cleanRelative == "." || strings.HasPrefix(cleanRelative, ".."+string(filepath.Separator)) || filepath.IsAbs(cleanRelative) {
		return Stored{}, fmt.Errorf("artifact path must stay under artifact store root")
	}
	target := filepath.Join(root, cleanRelative)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Stored{}, fmt.Errorf("create artifact directory: %w", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return Stored{}, fmt.Errorf("write artifact: %w", err)
	}
	sum := sha256.Sum256(content)
	return Stored{
		Backend:     "local",
		URI:         target,
		SHA256:      hex.EncodeToString(sum[:]),
		SizeBytes:   int64(len(content)),
		ContentType: contentType,
	}, nil
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
