-- Extend high-entropy API tokens with explicit production lifecycle metadata.

ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS token_type TEXT NOT NULL DEFAULT 'service';
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS created_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS rotated_from_token_id BIGINT REFERENCES api_tokens(id) ON DELETE SET NULL;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS reason TEXT;
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE api_tokens DROP CONSTRAINT IF EXISTS api_tokens_token_type_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_token_type_check
    CHECK (token_type IN ('service', 'legacy'));

CREATE INDEX IF NOT EXISTS api_tokens_expiry_idx
    ON api_tokens (status, expires_at);

CREATE INDEX IF NOT EXISTS api_tokens_rotated_from_idx
    ON api_tokens (rotated_from_token_id)
    WHERE rotated_from_token_id IS NOT NULL;
