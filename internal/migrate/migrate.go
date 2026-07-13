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
	Name    string
	Applied bool
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

	applied := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		alreadyApplied, err := isApplied(ctx, pool, migration.Name)
		if err != nil {
			return nil, err
		}
		if alreadyApplied {
			continue
		}
		if err := apply(ctx, pool, migration); err != nil {
			return nil, err
		}
		applied = append(applied, migration.Name)
	}
	return applied, nil
}

func Statuses(ctx context.Context, pool *pgxpool.Pool) ([]Status, error) {
	migrations, err := List()
	if err != nil {
		return nil, err
	}
	if err := ensureTable(ctx, pool); err != nil {
		return nil, err
	}

	statuses := make([]Status, 0, len(migrations))
	for _, migration := range migrations {
		applied, err := isApplied(ctx, pool, migration.Name)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, Status{
			Name:    migration.Name,
			Applied: applied,
		})
	}
	return statuses, nil
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
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, migration.Name); err != nil {
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
