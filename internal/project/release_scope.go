package project

import (
	"context"
	"strings"
	"time"
)

func normalizeReleaseScopeFields(generatedAt time.Time, projectKey string) (time.Time, string) {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return generatedAt, strings.TrimSpace(projectKey)
}

type resolvedReleaseScope struct {
	ProjectID  int64
	ProjectKey string
	Scope      string
}

func (s Store) resolveReleaseProjectScope(ctx context.Context, projectID int64, projectKey string) (resolvedReleaseScope, error) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey != "" {
		record, err := s.GetByKey(ctx, projectKey)
		if err != nil {
			return resolvedReleaseScope{}, err
		}
		return resolvedReleaseScope{ProjectID: record.ID, ProjectKey: record.Key, Scope: "project"}, nil
	}
	if projectID > 0 {
		return resolvedReleaseScope{ProjectID: projectID, Scope: "project"}, nil
	}
	return resolvedReleaseScope{Scope: "platform"}, nil
}

func releaseScopeAndProjectKey(projectID int64, projectKey string, fallbackKeys ...string) (string, string) {
	projectKey = strings.TrimSpace(projectKey)
	for _, fallback := range fallbackKeys {
		if projectKey != "" {
			break
		}
		projectKey = strings.TrimSpace(fallback)
	}
	if projectID > 0 || projectKey != "" {
		return "project", projectKey
	}
	return "platform", ""
}
