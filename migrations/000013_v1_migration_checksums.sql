-- AreaFlow v1 migration checksum source of truth.
-- NULL hashes identify legacy rows that require explicit attestation.

ALTER TABLE schema_migrations
    ADD COLUMN IF NOT EXISTS sha256 TEXT,
    ADD COLUMN IF NOT EXISTS hash_algorithm TEXT,
    ADD COLUMN IF NOT EXISTS hash_recorded_at TIMESTAMPTZ;

ALTER TABLE schema_migrations
    DROP CONSTRAINT IF EXISTS schema_migrations_sha256_check;

ALTER TABLE schema_migrations
    ADD CONSTRAINT schema_migrations_sha256_check
    CHECK (sha256 IS NULL OR sha256 ~ '^[0-9a-f]{64}$');

ALTER TABLE schema_migrations
    DROP CONSTRAINT IF EXISTS schema_migrations_hash_algorithm_check;

ALTER TABLE schema_migrations
    ADD CONSTRAINT schema_migrations_hash_algorithm_check
    CHECK (
        (sha256 IS NULL AND hash_algorithm IS NULL AND hash_recorded_at IS NULL)
        OR
        (sha256 IS NOT NULL AND hash_algorithm = 'sha256' AND hash_recorded_at IS NOT NULL)
    );
