package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/areasong/areaflow/internal/artifact"
	"github.com/areasong/areaflow/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestArtifactMigrationRelativePathIsDeterministic(t *testing.T) {
	path, err := artifactMigrationRelativePath(Record{Key: "areamatrix"}, ArtifactRecord{ID: 42, SourcePath: "v1/reports/result.json"})
	if err != nil {
		t.Fatalf("build migration path: %v", err)
	}
	if path != "areamatrix/migration/artifact-42/v1/reports/result.json" {
		t.Fatalf("path = %q", path)
	}
}

func TestArtifactMigrationRelativePathRejectsEscape(t *testing.T) {
	if _, err := artifactMigrationRelativePath(Record{Key: "areamatrix"}, ArtifactRecord{ID: 42, SourcePath: "../secret"}); err == nil {
		t.Fatal("expected escaping source path to fail")
	}
}

func TestVerifyMigratedArtifact(t *testing.T) {
	source := ArtifactRecord{SHA256: "abc", SizeBytes: 12}
	if err := verifyMigratedArtifact(source, artifact.Stored{SHA256: "abc", SizeBytes: 12}); err != nil {
		t.Fatalf("matching migration should verify: %v", err)
	}
	if err := verifyMigratedArtifact(source, artifact.Stored{SHA256: "def", SizeBytes: 12}); err == nil {
		t.Fatal("expected hash mismatch")
	}
	if err := verifyMigratedArtifact(source, artifact.Stored{SHA256: "abc", SizeBytes: 11}); err == nil {
		t.Fatal("expected size mismatch")
	}
	content := []byte("verified-content")
	sha, size := hashBytes(content)
	if err := verifyMigratedArtifactContent(ArtifactRecord{SHA256: sha, SizeBytes: size}, content); err != nil {
		t.Fatalf("readback content should verify: %v", err)
	}
	if err := verifyMigratedArtifactContent(ArtifactRecord{SHA256: "wrong", SizeBytes: size}, content); err == nil {
		t.Fatal("expected readback hash mismatch")
	}
}

func TestArtifactMigrationStatus(t *testing.T) {
	verified := time.Now().UTC()
	item := ArtifactMigrationItem{Artifact: ArtifactRecord{SHA256: "abc", SizeBytes: 12}}
	if status := artifactMigrationStatus(item); status != "pending" {
		t.Fatalf("status = %q", status)
	}
	item.Target = ArtifactLocation{ID: 1, Role: "migration_candidate", SHA256: "abc", SizeBytes: 12, VerifiedAt: &verified}
	if status := artifactMigrationStatus(item); status != "verified" {
		t.Fatalf("status = %q", status)
	}
	item.Target.Role = "primary"
	item.Target.Metadata = map[string]any{"status": "observing"}
	if status := artifactMigrationStatus(item); status != "observing" {
		t.Fatalf("status = %q", status)
	}
	item.Target.Metadata["status"] = "stable"
	if status := artifactMigrationStatus(item); status != "stable" {
		t.Fatalf("status = %q", status)
	}
}

func TestArtifactMigrationPostgresS3Smoke(t *testing.T) {
	if os.Getenv("AREAFLOW_ARTIFACT_MIGRATION_SMOKE") != "1" {
		t.Skip("set AREAFLOW_ARTIFACT_MIGRATION_SMOKE=1 to run PostgreSQL and S3 migration smoke")
	}
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("AREAFLOW_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}

	fixtureKey := fmt.Sprintf("artifact-migration-%d", time.Now().UnixNano())
	var projectID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO projects (project_key, name, kind, adapter, workflow_profile, default_branch)
VALUES ($1, $1, 'fixture', 'fixture', 'fixture', 'main') RETURNING id`, fixtureKey).Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM projects WHERE id = $1`, projectID)

	content := []byte(`{"migration":"verified"}`)
	localRoot := t.TempDir()
	stored, err := artifact.WriteConfigured(ctx, "local", localRoot, filepath.Join(fixtureKey, "report.json"), content, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var artifactID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO artifacts (project_id, artifact_type, storage_backend, uri, source_path, sha256, size_bytes, content_type)
VALUES ($1, 'migration_smoke', 'local', $2, 'report.json', $3, $4, 'application/json') RETURNING id`,
		projectID, stored.URI, stored.SHA256, stored.SizeBytes).Scan(&artifactID); err != nil {
		t.Fatal(err)
	}

	record := Record{ID: projectID, Key: fixtureKey, ArtifactBackend: "local", ArtifactRoot: localRoot}
	store := NewStore(pool)
	inventory, err := store.ArtifactMigrationInventory(ctx, record, "local", "s3")
	if err != nil || inventory.Pending != 1 {
		t.Fatalf("inventory = %+v err=%v", inventory, err)
	}
	target, err := store.CopyArtifactToBackend(ctx, record, artifactID, CopyArtifactOptions{
		TargetBackend: "s3", TargetRoot: "migration-smoke", Actor: "integration-test", Reason: "verify local to S3 migration",
	})
	if err != nil {
		t.Fatal(err)
	}
	observationUntil := time.Now().UTC().Add(250 * time.Millisecond)
	activated, err := store.ActivateArtifactLocation(ctx, record, artifactID, ActivateArtifactOptions{
		TargetLocationID: target.ID, ObservationUntil: observationUntil, Actor: "integration-test", Reason: "activate verified S3 location",
	})
	if err != nil || activated.StorageBackend != "s3" {
		t.Fatalf("activated = %+v err=%v", activated, err)
	}
	inventory, err = store.ArtifactMigrationInventory(ctx, record, "local", "s3")
	if err != nil || inventory.Activated != 1 || inventory.Observing != 1 {
		t.Fatalf("activated inventory = %+v err=%v", inventory, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE artifacts SET uri = 's3://unavailable/missing' WHERE id = $1`, artifactID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE artifact_locations SET uri = 's3://unavailable/missing' WHERE id = $1`, target.ID); err != nil {
		t.Fatal(err)
	}
	read, err := store.GetArtifactContent(ctx, artifactID)
	if err != nil || string(read.Content) != string(content) {
		t.Fatalf("dual-read fallback content=%q err=%v", read.Content, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE artifacts SET uri = $2 WHERE id = $1`, artifactID, target.URI); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE artifact_locations SET uri = $2 WHERE id = $1`, target.ID, target.URI); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(observationUntil) + 25*time.Millisecond)
	stable, err := store.CompleteArtifactObservation(ctx, record, artifactID, CompleteArtifactObservationOptions{Actor: "integration-test", Reason: "observation completed"})
	if err != nil || artifactMetadataString(stable.Metadata, "artifact_migration_status") != "stable" {
		t.Fatalf("stable artifact = %+v err=%v", stable, err)
	}
	inventory, err = store.ArtifactMigrationInventory(ctx, record, "local", "s3")
	if err != nil || inventory.Stable != 1 {
		t.Fatalf("stable inventory = %+v err=%v", inventory, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE artifacts SET uri = 's3://unavailable/missing' WHERE id = $1`, artifactID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE artifact_locations SET uri = 's3://unavailable/missing' WHERE id = $1`, target.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetArtifactContent(ctx, artifactID); err == nil {
		t.Fatal("stable artifact must not fall back to retained local source")
	}
}
