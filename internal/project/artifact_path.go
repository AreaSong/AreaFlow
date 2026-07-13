package project

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/areasong/areaflow/internal/artifact"
)

func writeLocalProjectArtifact(record Record, relativePath string, content []byte, contentType string) (artifact.Stored, error) {
	cleanRelative := filepath.Clean(relativePath)
	if cleanRelative == "." || filepath.IsAbs(cleanRelative) || strings.HasPrefix(cleanRelative, ".."+string(filepath.Separator)) {
		return artifact.Stored{}, fmt.Errorf("artifact path must stay under project artifact namespace")
	}
	return artifact.WriteLocal(record.ArtifactRoot, filepath.Join(record.Key, cleanRelative), content, contentType)
}
