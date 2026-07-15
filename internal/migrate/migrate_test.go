package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	migrations, err := List()
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(migrations) != 14 {
		t.Fatalf("migration count = %d, want 14", len(migrations))
	}
	if migrations[0].Name != "000001_v0_1_core.sql" {
		t.Fatalf("migration name = %q", migrations[0].Name)
	}
	if migrations[1].Name != "000002_v0_3_command_requests.sql" {
		t.Fatalf("migration name = %q", migrations[1].Name)
	}
	if migrations[2].Name != "000003_v0_3_gate_results.sql" {
		t.Fatalf("migration name = %q", migrations[2].Name)
	}
	if migrations[3].Name != "000004_v0_3_approval_transition.sql" {
		t.Fatalf("migration name = %q", migrations[3].Name)
	}
	if migrations[4].Name != "000005_v0_5_runner_preview.sql" {
		t.Fatalf("migration name = %q", migrations[4].Name)
	}
	if migrations[5].Name != "000006_v0_6_worker_registry.sql" {
		t.Fatalf("migration name = %q", migrations[5].Name)
	}
	if migrations[6].Name != "000007_v0_8_scheduling_policy.sql" {
		t.Fatalf("migration name = %q", migrations[6].Name)
	}
	if migrations[7].Name != "000008_v0_3_workflow_item_links.sql" {
		t.Fatalf("migration name = %q", migrations[7].Name)
	}
	if migrations[8].Name != "000009_v1_boundary_foundation.sql" {
		t.Fatalf("migration name = %q", migrations[8].Name)
	}
	if migrations[9].Name != "000010_v1_status_projections.sql" {
		t.Fatalf("migration name = %q", migrations[9].Name)
	}
	if migrations[10].Name != "000011_v1_migration_ledger.sql" {
		t.Fatalf("migration name = %q", migrations[10].Name)
	}
	if migrations[11].Name != ReleaseExceptionMigrationName {
		t.Fatalf("migration name = %q", migrations[11].Name)
	}
	if migrations[12].Name != ChecksumMigrationName {
		t.Fatalf("migration name = %q", migrations[12].Name)
	}
	if migrations[13].Name != "000014_v1_project_history_attribution.sql" {
		t.Fatalf("migration name = %q", migrations[13].Name)
	}
	for _, migration := range migrations {
		if migration.SQL == "" {
			t.Fatalf("migration %s SQL is empty", migration.Name)
		}
	}
}

func TestMigrationChecksumSchema(t *testing.T) {
	migration, err := migrationByName(ChecksumMigrationName)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"ADD COLUMN IF NOT EXISTS sha256", "hash_algorithm", "hash_recorded_at", "^[0-9a-f]{64}$"} {
		if !strings.Contains(migration.SQL, fragment) {
			t.Fatalf("migration checksum schema missing %q", fragment)
		}
	}
	digest, err := MigrationSetDigest()
	if err != nil {
		t.Fatal(err)
	}
	if len(digest) != 64 {
		t.Fatalf("migration set digest length = %d", len(digest))
	}
}

func TestReleaseExceptionMigrationContainsLifecycleSchema(t *testing.T) {
	migration, err := migrationByName(ReleaseExceptionMigrationName)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS release_exceptions",
		"UNIQUE (project_id, exception_key)",
		"status IN ('requested', 'approved', 'rejected', 'revoked', 'expired')",
		"release_exceptions_project_status_idx",
	} {
		if !strings.Contains(migration.SQL, fragment) {
			t.Fatalf("release exception migration missing %q", fragment)
		}
	}
}

func TestApprovalAllowsMigrationRequiresApprovedMatchingHash(t *testing.T) {
	if !approvalAllowsMigration(ApprovalState{Status: "approved", MigrationHash: "hash"}, "hash") {
		t.Fatal("matching approved migration should be allowed")
	}
	if approvalAllowsMigration(ApprovalState{Status: "revoked", MigrationHash: "hash"}, "hash") {
		t.Fatal("revoked migration approval must be denied")
	}
	if approvalAllowsMigration(ApprovalState{Status: "approved", MigrationHash: "stale"}, "hash") {
		t.Fatal("stale migration hash must be denied")
	}
}

func TestMigrationLedgerMigrationReservesLedgerTable(t *testing.T) {
	migrations, err := List()
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}

	var sql string
	for _, migration := range migrations {
		if migration.Name == "000011_v1_migration_ledger.sql" {
			sql = migration.SQL
			break
		}
	}
	if sql == "" {
		t.Fatal("migration ledger migration missing")
	}
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS migration_ledger",
		"migration_name TEXT NOT NULL",
		"phase TEXT NOT NULL CHECK",
		"status TEXT NOT NULL CHECK",
		"evidence_json JSONB NOT NULL DEFAULT '{}'::jsonb",
		"remediation TEXT",
		"UNIQUE (migration_name, phase)",
		"'preflight'",
		"'apply'",
		"'verify'",
		"'remediation'",
		"schema_migrations_recorded",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration ledger migration missing %q", fragment)
		}
	}
}

func TestStatusProjectionsMigrationReservesProjectionTable(t *testing.T) {
	migrations, err := List()
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}

	var sql string
	for _, migration := range migrations {
		if migration.Name == "000010_v1_status_projections.sql" {
			sql = migration.SQL
			break
		}
	}
	if sql == "" {
		t.Fatal("status projections migration missing")
	}
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS status_projections",
		"target_kind TEXT NOT NULL",
		"target_uri TEXT NOT NULL",
		"summary_state TEXT NOT NULL",
		"payload_json JSONB NOT NULL",
		"source_event_id BIGINT REFERENCES events(id)",
		"source_hash TEXT",
		"write_state TEXT NOT NULL",
		"written_at TIMESTAMPTZ",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("status projections migration missing %q", fragment)
		}
	}
}

func TestV1BoundaryFoundationMigrationReservesPlatformTables(t *testing.T) {
	migrations, err := List()
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}

	var sql string
	for _, migration := range migrations {
		if migration.Name == "000009_v1_boundary_foundation.sql" {
			sql = migration.SQL
			break
		}
	}
	if sql == "" {
		t.Fatal("v1 boundary foundation migration missing")
	}

	for _, table := range []string{
		"users",
		"teams",
		"memberships",
		"adapters",
		"workflow_profiles",
		"project_configs",
		"artifact_locations",
		"artifact_snapshots",
		"secret_refs",
		"engine_profiles",
		"api_tokens",
		"webhooks",
	} {
		if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("v1 boundary foundation missing table %s", table)
		}
	}
}

func TestEmbeddedMigrationMatchesRootMigration(t *testing.T) {
	migrations, err := List()
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	for _, migration := range migrations {
		rootPath := filepath.Join("..", "..", "migrations", migration.Name)
		rootSQL, err := os.ReadFile(rootPath)
		if err != nil {
			t.Fatalf("read root migration %s: %v", rootPath, err)
		}
		if string(rootSQL) != migration.SQL {
			t.Fatalf("embedded migration %s differs from root migration", migration.Name)
		}
	}
}
