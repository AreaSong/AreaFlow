package migrate

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Migration struct {
	Name string
	SQL  string
}

type Status struct {
	Name           string
	Applied        bool
	ChecksumStatus string
	ExpectedSHA256 string
	RecordedSHA256 string
}

const ReleaseExceptionMigrationName = "000012_v1_release_exceptions.sql"
const ChecksumMigrationName = "000013_v1_migration_checksums.sql"
const releaseExceptionApprovalLedgerPhase = "remediation"

type ApprovalState struct {
	Status        string
	MigrationHash string
	Actor         string
	Reason        string
	Applied       bool
}

func List() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		migrations = append(migrations, Migration{
			Name: entry.Name(),
			SQL:  string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})
	return migrations, nil
}

func Up(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	migrations, err := List()
	if err != nil {
		return nil, err
	}
	if err := ensureTable(ctx, pool); err != nil {
		return nil, err
	}
	if err := rejectChecksumMismatch(ctx, pool, migrations); err != nil {
		return nil, err
	}

	applied := make([]string, 0, len(migrations))
	appliedThisRun := make(map[string]string)
	for _, migration := range migrations {
		alreadyApplied, err := isApplied(ctx, pool, migration.Name)
		if err != nil {
			return nil, err
		}
		if alreadyApplied {
			continue
		}
		if migration.Name > ChecksumMigrationName {
			if err := requireVerifiedChecksums(ctx, pool, migrations); err != nil {
				return nil, err
			}
		}
		if migration.Name == ReleaseExceptionMigrationName {
			approval, err := Approval(ctx, pool, migration.Name)
			if err != nil {
				return nil, err
			}
			if !approvalAllowsMigration(approval, migrationHash(migration.SQL)) {
				continue
			}
		}
		if err := apply(ctx, pool, migration); err != nil {
			return nil, err
		}
		applied = append(applied, migration.Name)
		appliedThisRun[migration.Name] = migrationHash(migration.SQL)
		if migration.Name == ChecksumMigrationName {
			if err := recordAppliedRunHashes(ctx, pool, appliedThisRun); err != nil {
				return nil, err
			}
		}
	}
	return applied, nil
}

func Approve(ctx context.Context, pool *pgxpool.Pool, name string, actor string, reason string) (ApprovalState, error) {
	migration, err := migrationByName(name)
	if err != nil {
		return ApprovalState{}, err
	}
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return ApprovalState{}, fmt.Errorf("migration approval actor and reason are required")
	}
	if err := ensureTable(ctx, pool); err != nil {
		return ApprovalState{}, err
	}
	var ledgerExists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.migration_ledger') IS NOT NULL`).Scan(&ledgerExists); err != nil {
		return ApprovalState{}, fmt.Errorf("check migration ledger: %w", err)
	}
	if !ledgerExists {
		return ApprovalState{}, fmt.Errorf("migration ledger is required before approving %s", name)
	}
	evidence, err := json.Marshal(map[string]any{
		"actor": actor, "reason": reason, "risk_level": "R4 migration_security", "approval_status": "approved",
	})
	if err != nil {
		return ApprovalState{}, fmt.Errorf("marshal migration approval evidence: %w", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO migration_ledger (migration_name, phase, status, message, migration_hash, evidence_json, remediation)
VALUES ($1, $4, 'pass', 'explicit R4 migration approval recorded', $2, $3::jsonb, 'revoke approval before apply or disable release exception writes after apply')
ON CONFLICT (migration_name, phase) DO UPDATE
SET status = EXCLUDED.status,
    message = EXCLUDED.message,
    migration_hash = EXCLUDED.migration_hash,
    evidence_json = EXCLUDED.evidence_json,
    remediation = EXCLUDED.remediation,
    updated_at = now()`, name, migrationHash(migration.SQL), string(evidence), releaseExceptionApprovalLedgerPhase); err != nil {
		return ApprovalState{}, fmt.Errorf("record migration approval: %w", err)
	}
	return Approval(ctx, pool, name)
}

func Revoke(ctx context.Context, pool *pgxpool.Pool, name string, actor string, reason string) (ApprovalState, error) {
	migration, err := migrationByName(name)
	if err != nil {
		return ApprovalState{}, err
	}
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return ApprovalState{}, fmt.Errorf("migration revocation actor and reason are required")
	}
	evidence, err := json.Marshal(map[string]any{
		"actor": actor, "reason": reason, "risk_level": "R4 migration_security", "approval_status": "revoked",
	})
	if err != nil {
		return ApprovalState{}, fmt.Errorf("marshal migration revocation evidence: %w", err)
	}
	result, err := pool.Exec(ctx, `
UPDATE migration_ledger
SET status = 'blocked',
    message = 'R4 migration approval revoked; release exception writes are disabled',
    migration_hash = $2,
    evidence_json = $3::jsonb,
    remediation = 'record a new explicit approval before any further release exception write',
    updated_at = now()
WHERE migration_name = $1 AND phase = $4`, name, migrationHash(migration.SQL), string(evidence), releaseExceptionApprovalLedgerPhase)
	if err != nil {
		return ApprovalState{}, fmt.Errorf("revoke migration approval: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ApprovalState{}, fmt.Errorf("migration approval not found for %s", name)
	}
	return Approval(ctx, pool, name)
}

func ApplyApproved(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	migration, err := migrationByName(name)
	if err != nil {
		return false, err
	}
	if err := ensureTable(ctx, pool); err != nil {
		return false, err
	}
	applied, err := isApplied(ctx, pool, name)
	if err != nil || applied {
		return false, err
	}
	approval, err := Approval(ctx, pool, name)
	if err != nil {
		return false, err
	}
	if !approvalAllowsMigration(approval, migrationHash(migration.SQL)) {
		return false, fmt.Errorf("effective approved migration approval is required for %s", name)
	}
	if err := apply(ctx, pool, migration); err != nil {
		return false, err
	}
	return true, nil
}

func Approval(ctx context.Context, pool *pgxpool.Pool, name string) (ApprovalState, error) {
	state := ApprovalState{}
	var tablesReady bool
	if err := pool.QueryRow(ctx, `
SELECT to_regclass('public.schema_migrations') IS NOT NULL
   AND to_regclass('public.migration_ledger') IS NOT NULL`).Scan(&tablesReady); err != nil {
		return ApprovalState{}, fmt.Errorf("check migration approval tables: %w", err)
	}
	if !tablesReady {
		return state, nil
	}
	var evidenceRaw []byte
	var ledgerStatus string
	err := pool.QueryRow(ctx, `
SELECT status, migration_hash, evidence_json,
       EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)
FROM migration_ledger
WHERE migration_name = $1 AND phase = $2`, name, releaseExceptionApprovalLedgerPhase).Scan(&ledgerStatus, &state.MigrationHash, &evidenceRaw, &state.Applied)
	if err == pgx.ErrNoRows {
		applied, applyErr := isApplied(ctx, pool, name)
		state.Applied = applied
		return state, applyErr
	}
	if err != nil {
		return ApprovalState{}, fmt.Errorf("load migration approval: %w", err)
	}
	var evidence map[string]any
	if err := json.Unmarshal(evidenceRaw, &evidence); err != nil {
		return ApprovalState{}, fmt.Errorf("decode migration approval evidence: %w", err)
	}
	state.Actor, _ = evidence["actor"].(string)
	state.Reason, _ = evidence["reason"].(string)
	state.Status, _ = evidence["approval_status"].(string)
	if state.Status == "" {
		state.Status = ledgerStatus
	}
	return state, nil
}

func migrationByName(name string) (Migration, error) {
	migrations, err := List()
	if err != nil {
		return Migration{}, err
	}
	for _, migration := range migrations {
		if migration.Name == name {
			return migration, nil
		}
	}
	return Migration{}, fmt.Errorf("migration not found: %s", name)
}

func ApprovalEffective(name string, state ApprovalState) (bool, error) {
	migration, err := migrationByName(name)
	if err != nil {
		return false, err
	}
	return approvalAllowsMigration(state, migrationHash(migration.SQL)), nil
}

func ExpectedHash(name string) (string, error) {
	migration, err := migrationByName(name)
	if err != nil {
		return "", err
	}
	return migrationHash(migration.SQL), nil
}

func approvalAllowsMigration(state ApprovalState, expectedHash string) bool {
	return state.Status == "approved" && state.MigrationHash == expectedHash
}

func Statuses(ctx context.Context, pool *pgxpool.Pool) ([]Status, error) {
	migrations, err := List()
	if err != nil {
		return nil, err
	}
	if err := ensureTable(ctx, pool); err != nil {
		return nil, err
	}

	checksumColumns, err := checksumColumnsExist(ctx, pool)
	if err != nil {
		return nil, err
	}
	statuses := make([]Status, 0, len(migrations))
	for _, migration := range migrations {
		applied, err := isApplied(ctx, pool, migration.Name)
		if err != nil {
			return nil, err
		}
		status := Status{Name: migration.Name, Applied: applied, ExpectedSHA256: migrationHash(migration.SQL)}
		switch {
		case !applied:
			status.ChecksumStatus = "pending"
		case !checksumColumns:
			status.ChecksumStatus = "legacy_unverified"
		default:
			if err := pool.QueryRow(ctx, `SELECT COALESCE(sha256, '') FROM schema_migrations WHERE name = $1`, migration.Name).Scan(&status.RecordedSHA256); err != nil {
				return nil, fmt.Errorf("load migration checksum %s: %w", migration.Name, err)
			}
			if status.RecordedSHA256 == "" {
				status.ChecksumStatus = "legacy_unverified"
			} else if status.RecordedSHA256 == status.ExpectedSHA256 {
				status.ChecksumStatus = "verified"
			} else {
				status.ChecksumStatus = "mismatch"
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func MigrationSetDigest() (string, error) {
	migrations, err := List()
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		parts = append(parts, migration.Name+":"+migrationHash(migration.SQL))
	}
	return migrationHash(strings.Join(parts, "\n")), nil
}

func AttestLegacyHashes(ctx context.Context, pool *pgxpool.Pool, actor string, reason string, expectedSetDigest string) (int64, error) {
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	expectedSetDigest = strings.TrimSpace(expectedSetDigest)
	if actor == "" || reason == "" || expectedSetDigest == "" {
		return 0, fmt.Errorf("actor, reason and expected migration set digest are required")
	}
	currentSetDigest, err := MigrationSetDigest()
	if err != nil {
		return 0, err
	}
	if expectedSetDigest != currentSetDigest {
		return 0, fmt.Errorf("migration set digest mismatch: expected %s, current %s", expectedSetDigest, currentSetDigest)
	}
	migrations, err := List()
	if err != nil {
		return 0, err
	}
	if err := requireChecksumColumns(ctx, pool); err != nil {
		return 0, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin migration checksum attestation: %w", err)
	}
	defer tx.Rollback(ctx)

	embedded := make(map[string]string, len(migrations))
	for _, migration := range migrations {
		embedded[migration.Name] = migrationHash(migration.SQL)
	}
	rows, err := tx.Query(ctx, `SELECT name, COALESCE(sha256, '') FROM schema_migrations ORDER BY name FOR UPDATE`)
	if err != nil {
		return 0, fmt.Errorf("lock schema migrations for attestation: %w", err)
	}
	type legacyHash struct{ name, hash string }
	legacy := []legacyHash{}
	for rows.Next() {
		var name, recorded string
		if err := rows.Scan(&name, &recorded); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan schema migration for attestation: %w", err)
		}
		expected, ok := embedded[name]
		if !ok {
			rows.Close()
			return 0, fmt.Errorf("applied migration is missing from embedded set: %s", name)
		}
		if recorded != "" && recorded != expected {
			rows.Close()
			return 0, fmt.Errorf("migration checksum mismatch for %s", name)
		}
		if recorded == "" {
			legacy = append(legacy, legacyHash{name: name, hash: expected})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("list schema migrations for attestation: %w", err)
	}
	rows.Close()
	for _, item := range legacy {
		if _, err := tx.Exec(ctx, `UPDATE schema_migrations SET sha256 = $2, hash_algorithm = 'sha256', hash_recorded_at = now() WHERE name = $1 AND sha256 IS NULL`, item.name, item.hash); err != nil {
			return 0, fmt.Errorf("attest migration checksum %s: %w", item.name, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE migration_ledger SET migration_hash = $2, evidence_json = evidence_json || jsonb_build_object('checksum_attested', true, 'migration_set_digest', $3::text), updated_at = now() WHERE migration_name = $1 AND phase = 'verify'`, item.name, item.hash, currentSetDigest); err != nil {
			return 0, fmt.Errorf("record migration checksum ledger attestation %s: %w", item.name, err)
		}
	}
	var actorID int64
	if err := tx.QueryRow(ctx, `INSERT INTO actors (kind, display_name, external_key) VALUES ('user', $1, $2) ON CONFLICT (external_key) WHERE external_key IS NOT NULL DO UPDATE SET display_name = EXCLUDED.display_name RETURNING id`, actor, "migration-attestation:"+actor).Scan(&actorID); err != nil {
		return 0, fmt.Errorf("ensure migration attestation actor: %w", err)
	}
	metadata, _ := json.Marshal(map[string]any{"migration_set_digest": currentSetDigest, "attested_count": len(legacy), "hash_algorithm": "sha256"})
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (actor_id, action, capability, resource_type, resource, decision, reason, metadata) VALUES ($1, 'migration.checksum.attest', 'migration_security', 'schema_migrations', $2, 'allowed', $3, $4::jsonb)`, actorID, currentSetDigest, reason, string(metadata)); err != nil {
		return 0, fmt.Errorf("audit migration checksum attestation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit migration checksum attestation: %w", err)
	}
	return int64(len(legacy)), nil
}

func ensureTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return exists, nil
}

func apply(ctx context.Context, pool *pgxpool.Pool, migration Migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Name, err)
	}
	defer tx.Rollback(ctx)

	ledgerAvailable, err := migrationLedgerTableExists(ctx, tx)
	if err != nil {
		return fmt.Errorf("check migration ledger before %s: %w", migration.Name, err)
	}
	if ledgerAvailable {
		if err := recordMigrationLedgerPhase(ctx, tx, migration, "preflight", "ready", "migration preflight checks passed", "rerun areaflow migrate up after checking schema_migrations and embedded migration list"); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("apply migration %s: %w", migration.Name, err)
	}
	checksumColumns, err := checksumColumnsExistTx(ctx, tx)
	if err != nil {
		return fmt.Errorf("check checksum columns after %s: %w", migration.Name, err)
	}
	insertSQL := `INSERT INTO schema_migrations (name) VALUES ($1)`
	args := []any{migration.Name}
	if checksumColumns {
		insertSQL = `INSERT INTO schema_migrations (name, sha256, hash_algorithm, hash_recorded_at) VALUES ($1, $2, 'sha256', now())`
		args = append(args, migrationHash(migration.SQL))
	}
	if _, err := tx.Exec(ctx, insertSQL, args...); err != nil {
		return fmt.Errorf("record migration %s: %w", migration.Name, err)
	}
	ledgerAvailable, err = migrationLedgerTableExists(ctx, tx)
	if err != nil {
		return fmt.Errorf("check migration ledger after %s: %w", migration.Name, err)
	}
	if ledgerAvailable {
		if err := recordMigrationLedgerPhase(ctx, tx, migration, "preflight", "ready", "migration preflight checks passed", "rerun areaflow migrate up after checking schema_migrations and embedded migration list"); err != nil {
			return err
		}
		if err := recordMigrationLedgerPhase(ctx, tx, migration, "apply", "ready", "migration SQL applied and schema_migrations row recorded", "restore from backup or apply a targeted remediation migration after approval"); err != nil {
			return err
		}
		if err := recordMigrationLedgerPhase(ctx, tx, migration, "verify", "ready", "migration is visible in schema_migrations before commit", "rerun migration ledger readiness after repairing schema state"); err != nil {
			return err
		}
		if err := recordMigrationLedgerPhase(ctx, tx, migration, "remediation", "ready", "rollback/remediation wording is recorded for migration audit readiness", "prepare approved forward-only remediation migration; destructive rollback is not opened"); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.Name, err)
	}
	return nil
}

func checksumColumnsExist(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'schema_migrations' AND column_name = 'sha256')`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check schema migration checksum columns: %w", err)
	}
	return exists, nil
}

func checksumColumnsExistTx(ctx context.Context, tx pgx.Tx) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'schema_migrations' AND column_name = 'sha256')`).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func requireChecksumColumns(ctx context.Context, pool *pgxpool.Pool) error {
	exists, err := checksumColumnsExist(ctx, pool)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("migration checksum schema is not applied; run areaflow migrate up first")
	}
	return nil
}

func rejectChecksumMismatch(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) error {
	exists, err := checksumColumnsExist(ctx, pool)
	if err != nil || !exists {
		return err
	}
	for _, migration := range migrations {
		var recorded string
		err := pool.QueryRow(ctx, `SELECT COALESCE(sha256, '') FROM schema_migrations WHERE name = $1`, migration.Name).Scan(&recorded)
		if err == pgx.ErrNoRows || recorded == "" {
			continue
		}
		if err != nil {
			return fmt.Errorf("load migration checksum %s: %w", migration.Name, err)
		}
		if recorded != migrationHash(migration.SQL) {
			return fmt.Errorf("migration checksum mismatch for %s", migration.Name)
		}
	}
	return nil
}

func requireVerifiedChecksums(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) error {
	if err := requireChecksumColumns(ctx, pool); err != nil {
		return err
	}
	if err := rejectChecksumMismatch(ctx, pool, migrations); err != nil {
		return err
	}
	var legacyCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE sha256 IS NULL`).Scan(&legacyCount); err != nil {
		return fmt.Errorf("count legacy migration checksums: %w", err)
	}
	if legacyCount > 0 {
		return fmt.Errorf("%d applied migrations are legacy_unverified; run explicit checksum attestation before applying new migrations", legacyCount)
	}
	return nil
}

func recordAppliedRunHashes(ctx context.Context, pool *pgxpool.Pool, applied map[string]string) error {
	for name, hash := range applied {
		if _, err := pool.Exec(ctx, `UPDATE schema_migrations SET sha256 = $2, hash_algorithm = 'sha256', hash_recorded_at = now() WHERE name = $1 AND sha256 IS NULL`, name, hash); err != nil {
			return fmt.Errorf("record same-run migration checksum %s: %w", name, err)
		}
	}
	return nil
}

func migrationLedgerTableExists(ctx context.Context, tx pgx.Tx) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass('public.migration_ledger') IS NOT NULL`).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func recordMigrationLedgerPhase(ctx context.Context, tx pgx.Tx, migration Migration, phase string, status string, message string, remediation string) error {
	evidence, err := json.Marshal(map[string]any{
		"embedded":                    true,
		"migration_hash":              migrationHash(migration.SQL),
		"schema_migrations_recorded":  phase != "preflight",
		"migration_runner_recorded":   true,
		"destructive_rollback_opened": false,
	})
	if err != nil {
		return fmt.Errorf("marshal migration ledger evidence for %s/%s: %w", migration.Name, phase, err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO migration_ledger (migration_name, phase, status, message, migration_hash, evidence_json, remediation)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
ON CONFLICT (migration_name, phase) DO UPDATE
SET status = EXCLUDED.status,
    message = EXCLUDED.message,
    migration_hash = EXCLUDED.migration_hash,
    evidence_json = migration_ledger.evidence_json || EXCLUDED.evidence_json,
    remediation = EXCLUDED.remediation,
    updated_at = now()`,
		migration.Name,
		phase,
		status,
		message,
		migrationHash(migration.SQL),
		string(evidence),
		remediation,
	); err != nil {
		return fmt.Errorf("record migration ledger phase %s/%s: %w", migration.Name, phase, err)
	}
	return nil
}

func migrationHash(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}
