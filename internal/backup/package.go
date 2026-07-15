package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileRecord struct {
	ID             int64  `json:"id"`
	ProjectKey     string `json:"project_key"`
	StorageBackend string `json:"storage_backend"`
	OriginalURI    string `json:"original_uri"`
	PackagePath    string `json:"package_path,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	SizeBytes      int64  `json:"size_bytes,omitempty"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
}

type MigrationRecord struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type TableCount struct {
	Table string `json:"table"`
	Rows  int64  `json:"rows"`
}

type DatabaseRecord struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
	Format    string `json:"format"`
}

type Manifest struct {
	SchemaVersion       int               `json:"schema_version"`
	BackupID            string            `json:"backup_id"`
	CreatedAt           time.Time         `json:"created_at"`
	Status              string            `json:"status"`
	Consistency         string            `json:"consistency"`
	Database            DatabaseRecord    `json:"database"`
	Migrations          []MigrationRecord `json:"migrations"`
	TableCounts         []TableCount      `json:"table_counts"`
	Files               []FileRecord      `json:"files"`
	IncludedLocalFiles  int               `json:"included_local_files"`
	MissingLocalFiles   int               `json:"missing_local_files"`
	ReferencedArtifacts int               `json:"referenced_artifacts"`
	ManifestSHA256      string            `json:"manifest_sha256"`
}

type CreateOptions struct {
	Destination string
	DatabaseURL string
	Quiesced    bool
}

type DrillOptions struct {
	PackagePath string
	DatabaseURL string
	DrillRoot   string
	Actor       string
	Reason      string
}

type DrillResult struct {
	Status        string `json:"status"`
	BackupID      string `json:"backup_id"`
	DatabaseName  string `json:"database_name"`
	ArtifactRoot  string `json:"artifact_root"`
	VerifiedFiles int    `json:"verified_files"`
}

func Create(ctx context.Context, pool *pgxpool.Pool, options CreateOptions) (Manifest, error) {
	if strings.TrimSpace(options.Destination) == "" || strings.TrimSpace(options.DatabaseURL) == "" {
		return Manifest{}, fmt.Errorf("backup destination and database URL are required")
	}
	if !options.Quiesced {
		return Manifest{}, fmt.Errorf("backup create requires --quiesced after stopping AreaFlow writers")
	}
	destination, err := filepath.Abs(options.Destination)
	if err != nil {
		return Manifest{}, fmt.Errorf("resolve backup destination: %w", err)
	}
	if entries, err := os.ReadDir(destination); err == nil && len(entries) > 0 {
		return Manifest{}, fmt.Errorf("backup destination must be empty: %s", destination)
	} else if err != nil && !os.IsNotExist(err) {
		return Manifest{}, fmt.Errorf("inspect backup destination: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(destination, "artifacts"), 0o700); err != nil {
		return Manifest{}, fmt.Errorf("create backup destination: %w", err)
	}
	manifest := Manifest{
		SchemaVersion: 1,
		BackupID:      filepath.Base(destination),
		CreatedAt:     time.Now().UTC(),
		Status:        "ready",
		Consistency:   "quiesced",
		Database:      DatabaseRecord{Path: "database.dump", Format: "postgres_custom"},
	}
	dumpPath := filepath.Join(destination, manifest.Database.Path)
	if err := runPGDump(ctx, options.DatabaseURL, dumpPath); err != nil {
		return Manifest{}, err
	}
	if err := validatePGDump(ctx, options.DatabaseURL, dumpPath); err != nil {
		return Manifest{}, err
	}
	manifest.Database.SHA256, manifest.Database.SizeBytes, err = hashFile(dumpPath)
	if err != nil {
		return Manifest{}, err
	}
	manifest.Migrations, err = migrationRecords(ctx, pool)
	if err != nil {
		return Manifest{}, err
	}
	manifest.TableCounts, err = tableCounts(ctx, pool)
	if err != nil {
		return Manifest{}, err
	}
	manifest.Files, err = copyArtifactFiles(ctx, pool, destination)
	if err != nil {
		return Manifest{}, err
	}
	for _, file := range manifest.Files {
		switch file.Status {
		case "included":
			manifest.IncludedLocalFiles++
		case "missing":
			manifest.MissingLocalFiles++
			manifest.Status = "needs_attention"
		case "referenced":
			manifest.ReferencedArtifacts++
			manifest.Status = "needs_attention"
		}
	}
	if err := writeManifest(destination, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Drill(ctx context.Context, sourcePool *pgxpool.Pool, options DrillOptions) (result DrillResult, err error) {
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" || options.Reason == "" {
		return DrillResult{}, fmt.Errorf("drill actor and reason are required")
	}
	packagePath, err := filepath.Abs(options.PackagePath)
	if err != nil {
		return DrillResult{}, fmt.Errorf("resolve backup package: %w", err)
	}
	manifest, err := readAndVerifyManifest(packagePath)
	if err != nil {
		return DrillResult{}, err
	}
	result = DrillResult{Status: "blocked", BackupID: manifest.BackupID}
	defer func() {
		auditErr := recordDrillAudit(ctx, sourcePool, manifest, options, result, err)
		if err == nil && auditErr != nil {
			err = auditErr
		}
	}()
	if manifest.MissingLocalFiles > 0 {
		return result, fmt.Errorf("backup has %d missing local artifact files", manifest.MissingLocalFiles)
	}
	dumpHash, dumpSize, err := hashFile(filepath.Join(packagePath, manifest.Database.Path))
	if err != nil || dumpHash != manifest.Database.SHA256 || dumpSize != manifest.Database.SizeBytes {
		return result, fmt.Errorf("database dump hash or size mismatch")
	}
	for _, file := range manifest.Files {
		if file.Status != "included" {
			continue
		}
		hash, size, hashErr := hashFile(filepath.Join(packagePath, filepath.FromSlash(file.PackagePath)))
		if hashErr != nil || hash != file.SHA256 || size != file.SizeBytes {
			return result, fmt.Errorf("artifact package verification failed for artifact %d", file.ID)
		}
		result.VerifiedFiles++
	}
	result.DatabaseName = "areaflow_drill_" + sanitizeID(manifest.BackupID)
	if len(result.DatabaseName) > 60 {
		result.DatabaseName = result.DatabaseName[:60]
	}
	maintenanceURL, err := databaseURLWithName(options.DatabaseURL, "postgres")
	if err != nil {
		return result, err
	}
	maintenancePool, err := pgxpool.New(ctx, maintenanceURL)
	if err != nil {
		return result, fmt.Errorf("connect maintenance database: %w", err)
	}
	defer maintenancePool.Close()
	var exists bool
	if err := maintenancePool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, result.DatabaseName).Scan(&exists); err != nil {
		return result, fmt.Errorf("check drill database: %w", err)
	}
	if exists {
		return result, fmt.Errorf("drill database already exists: %s", result.DatabaseName)
	}
	if _, err := maintenancePool.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{result.DatabaseName}.Sanitize()+" TEMPLATE template0"); err != nil {
		return result, fmt.Errorf("create drill database: %w", err)
	}
	restoreURL, err := databaseURLWithName(options.DatabaseURL, result.DatabaseName)
	if err != nil {
		return result, err
	}
	if restoreErr := runPGRestore(ctx, restoreURL, filepath.Join(packagePath, manifest.Database.Path)); restoreErr != nil {
		return result, restoreErr
	}
	drillPool, err := pgxpool.New(ctx, restoreURL)
	if err != nil {
		return result, fmt.Errorf("connect restored drill database: %w", err)
	}
	defer drillPool.Close()
	if err := verifyRestoredDatabase(ctx, drillPool, manifest); err != nil {
		return result, err
	}
	if strings.TrimSpace(options.DrillRoot) == "" {
		options.DrillRoot = filepath.Join(filepath.Dir(packagePath), "restore-drills")
	}
	result.ArtifactRoot = filepath.Join(options.DrillRoot, manifest.BackupID)
	if err := os.MkdirAll(result.ArtifactRoot, 0o700); err != nil {
		return result, fmt.Errorf("create drill artifact root: %w", err)
	}
	for _, file := range manifest.Files {
		if file.Status != "included" {
			continue
		}
		target := filepath.Join(result.ArtifactRoot, filepath.FromSlash(file.PackagePath))
		if err := copyFile(filepath.Join(packagePath, filepath.FromSlash(file.PackagePath)), target); err != nil {
			return result, fmt.Errorf("restore drill artifact %d: %w", file.ID, err)
		}
	}
	result.Status = "pass"
	return result, nil
}

func migrationRecords(ctx context.Context, pool *pgxpool.Pool) ([]MigrationRecord, error) {
	rows, err := pool.Query(ctx, `SELECT name, COALESCE(sha256, '') FROM schema_migrations ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list migration checksums: %w", err)
	}
	defer rows.Close()
	result := []MigrationRecord{}
	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.Name, &record.SHA256); err != nil {
			return nil, err
		}
		if record.SHA256 == "" {
			return nil, fmt.Errorf("migration is not checksum verified: %s", record.Name)
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func tableCounts(ctx context.Context, pool *pgxpool.Pool) ([]TableCount, error) {
	rows, err := pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename`)
	if err != nil {
		return nil, fmt.Errorf("list public tables: %w", err)
	}
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	rows.Close()
	result := make([]TableCount, 0, len(names))
	for _, name := range names {
		var count int64
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+pgx.Identifier{name}.Sanitize()).Scan(&count); err != nil {
			return nil, fmt.Errorf("count table %s: %w", name, err)
		}
		result = append(result, TableCount{Table: name, Rows: count})
	}
	return result, nil
}

func copyArtifactFiles(ctx context.Context, pool *pgxpool.Pool, destination string) ([]FileRecord, error) {
	rows, err := pool.Query(ctx, `SELECT a.id, p.project_key, a.storage_backend, a.uri, COALESCE(a.sha256, ''), COALESCE(a.size_bytes, 0), COALESCE(c.root_path, '') FROM artifacts a JOIN projects p ON p.id = a.project_id LEFT JOIN LATERAL (SELECT root_path FROM project_connections WHERE project_id = p.id AND connection_type = 'artifact_store' ORDER BY updated_at DESC, id DESC LIMIT 1) c ON true ORDER BY a.id`)
	if err != nil {
		return nil, fmt.Errorf("list backup artifacts: %w", err)
	}
	defer rows.Close()
	result := []FileRecord{}
	for rows.Next() {
		var record FileRecord
		var expectedHash, artifactRoot string
		var expectedSize int64
		if err := rows.Scan(&record.ID, &record.ProjectKey, &record.StorageBackend, &record.OriginalURI, &expectedHash, &expectedSize, &artifactRoot); err != nil {
			return nil, err
		}
		if record.StorageBackend != "local" {
			record.Status = "referenced"
			record.Reason = "artifact bytes are owned by an external project or backend"
			result = append(result, record)
			continue
		}
		if _, err := os.Stat(record.OriginalURI); err != nil {
			record.Status = "missing"
			record.Reason = "local artifact file is unavailable"
			result = append(result, record)
			continue
		}
		if err := requireWithinRoot(record.OriginalURI, artifactRoot); err != nil {
			return nil, fmt.Errorf("artifact %d path boundary: %w", record.ID, err)
		}
		hash, size, err := hashFile(record.OriginalURI)
		if err != nil {
			return nil, err
		}
		if expectedHash != "" && hash != expectedHash {
			return nil, fmt.Errorf("artifact %d SHA-256 does not match database metadata", record.ID)
		}
		if expectedSize > 0 && size != expectedSize {
			return nil, fmt.Errorf("artifact %d size does not match database metadata", record.ID)
		}
		record.PackagePath = filepath.ToSlash(filepath.Join("artifacts", record.ProjectKey, fmt.Sprintf("%d", record.ID), filepath.Base(record.OriginalURI)))
		if err := copyFile(record.OriginalURI, filepath.Join(destination, filepath.FromSlash(record.PackagePath))); err != nil {
			return nil, err
		}
		record.SHA256 = hash
		record.SizeBytes = size
		record.Status = "included"
		result = append(result, record)
	}
	return result, rows.Err()
}

func requireWithinRoot(path string, root string) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("artifact root is missing")
	}
	root, err := expandHome(root)
	if err != nil {
		return err
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(realRoot, realPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path is outside configured artifact root")
	}
	return nil
}

func expandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home for artifact root: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func writeManifest(destination string, manifest *Manifest) error {
	manifest.ManifestSHA256 = ""
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup manifest: %w", err)
	}
	sum := sha256.Sum256(raw)
	manifest.ManifestSHA256 = hex.EncodeToString(sum[:])
	raw, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(destination, "manifest.json"), append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write backup manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(destination, "manifest.sha256"), []byte(manifest.ManifestSHA256+"  manifest.json\n"), 0o600)
}

func readAndVerifyManifest(packagePath string) (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(packagePath, "manifest.json"))
	if err != nil {
		return Manifest{}, fmt.Errorf("read backup manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode backup manifest: %w", err)
	}
	recorded := manifest.ManifestSHA256
	manifest.ManifestSHA256 = ""
	canonical, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	sum := sha256.Sum256(canonical)
	if recorded != hex.EncodeToString(sum[:]) {
		return Manifest{}, fmt.Errorf("backup manifest SHA-256 mismatch")
	}
	manifest.ManifestSHA256 = recorded
	return manifest, nil
}

func verifyRestoredDatabase(ctx context.Context, pool *pgxpool.Pool, manifest Manifest) error {
	migrations, err := migrationRecords(ctx, pool)
	if err != nil {
		return err
	}
	if !equalJSON(migrations, manifest.Migrations) {
		return fmt.Errorf("restored migration checksum set differs from manifest")
	}
	counts, err := tableCounts(ctx, pool)
	if err != nil {
		return err
	}
	if !equalJSON(counts, manifest.TableCounts) {
		return fmt.Errorf("restored table counts differ from manifest")
	}
	return nil
}

func recordDrillAudit(ctx context.Context, pool *pgxpool.Pool, manifest Manifest, options DrillOptions, result DrillResult, drillErr error) error {
	decision := "allowed"
	status := result.Status
	if drillErr != nil {
		decision = "blocked"
		status = "blocked"
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `INSERT INTO actors (kind, display_name, external_key) VALUES ('user', $1, $2) ON CONFLICT (external_key) WHERE external_key IS NOT NULL DO UPDATE SET display_name = EXCLUDED.display_name RETURNING id`, options.Actor, "backup-drill:"+options.Actor).Scan(&actorID); err != nil {
		return fmt.Errorf("ensure backup drill actor: %w", err)
	}
	metadata, _ := json.Marshal(map[string]any{"backup_id": manifest.BackupID, "status": status, "database_name": result.DatabaseName, "artifact_root": result.ArtifactRoot, "verified_files": result.VerifiedFiles})
	_, err := pool.Exec(ctx, `INSERT INTO audit_events (actor_id, action, capability, resource_type, resource, decision, reason, metadata) VALUES ($1, 'backup.restore.drill', 'backup_restore', 'backup_package', $2, $3, $4, $5::jsonb)`, actorID, manifest.BackupID, decision, options.Reason, string(metadata))
	return err
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open file for hashing %s: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("hash file %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func copyFile(source string, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func databaseURLWithName(raw string, name string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}
	parsed.Path = "/" + name
	return parsed.String(), nil
}

func runPGDump(ctx context.Context, databaseURL string, target string) error {
	if container, user, database, ok := dockerPostgresTarget(databaseURL); ok {
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("create database dump: %w", err)
		}
		command := exec.CommandContext(ctx, "docker", "exec", container, "pg_dump", "--format=custom", "--no-owner", "--no-privileges", "--username", user, "--dbname", database)
		command.Stdout = output
		var stderr bytes.Buffer
		command.Stderr = &stderr
		runErr := command.Run()
		closeErr := output.Close()
		if runErr != nil {
			return fmt.Errorf("container pg_dump failed: %w: %s", runErr, strings.TrimSpace(stderr.String()))
		}
		return closeErr
	}
	if output, err := exec.CommandContext(ctx, "pg_dump", "--format=custom", "--no-owner", "--no-privileges", "--file", target, "--dbname", databaseURL).CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func validatePGDump(ctx context.Context, databaseURL string, source string) error {
	if container, _, _, ok := dockerPostgresTarget(databaseURL); ok {
		input, err := os.Open(source)
		if err != nil {
			return err
		}
		defer input.Close()
		command := exec.CommandContext(ctx, "docker", "exec", "-i", container, "pg_restore", "--list")
		command.Stdin = input
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("validate container PostgreSQL dump: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	}
	if output, err := exec.CommandContext(ctx, "pg_restore", "--list", source).CombinedOutput(); err != nil {
		return fmt.Errorf("validate PostgreSQL dump: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runPGRestore(ctx context.Context, databaseURL string, source string) error {
	if container, user, database, ok := dockerPostgresTarget(databaseURL); ok {
		input, err := os.Open(source)
		if err != nil {
			return err
		}
		defer input.Close()
		command := exec.CommandContext(ctx, "docker", "exec", "-i", container, "pg_restore", "--exit-on-error", "--no-owner", "--no-privileges", "--username", user, "--dbname", database)
		command.Stdin = input
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("restore isolated database with container pg_restore: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	}
	if output, err := exec.CommandContext(ctx, "pg_restore", "--exit-on-error", "--no-owner", "--no-privileges", "--dbname", databaseURL, source).CombinedOutput(); err != nil {
		return fmt.Errorf("restore isolated database: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func dockerPostgresTarget(databaseURL string) (container string, user string, database string, ok bool) {
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.User == nil {
		return "", "", "", false
	}
	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return "", "", "", false
	}
	container = strings.TrimSpace(os.Getenv("AREAFLOW_POSTGRES_CONTAINER"))
	if container == "" {
		container = "areaflow-postgres"
	}
	if exec.Command("docker", "inspect", container).Run() != nil {
		return "", "", "", false
	}
	user = parsed.User.Username()
	database = strings.TrimPrefix(parsed.Path, "/")
	if user == "" || database == "" {
		return "", "", "", false
	}
	return container, user, database, true
}

func sanitizeID(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(value) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func equalJSON(left any, right any) bool {
	leftRaw, _ := json.Marshal(left)
	rightRaw, _ := json.Marshal(right)
	return string(leftRaw) == string(rightRaw)
}

func SortedMissingProjects(manifest Manifest) []string {
	seen := map[string]bool{}
	for _, file := range manifest.Files {
		if file.Status == "missing" {
			seen[file.ProjectKey] = true
		}
	}
	result := make([]string, 0, len(seen))
	for key := range seen {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
