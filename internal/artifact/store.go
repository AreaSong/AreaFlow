package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	return NewLocalBackend(root).Put(context.Background(), relativePath, content, contentType)
}

func stored(backend, uri string, content []byte, contentType string) Stored {
	sum := sha256.Sum256(content)
	return Stored{Backend: backend, URI: uri, SHA256: hex.EncodeToString(sum[:]), SizeBytes: int64(len(content)), ContentType: contentType}
}

func sha256Bytes(content []byte) []byte {
	sum := sha256.Sum256(content)
	return sum[:]
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
