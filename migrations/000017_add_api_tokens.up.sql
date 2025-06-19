-- Add api_tokens table
CREATE TABLE IF NOT EXISTS api_tokens (
    api_token_id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    token_hash TEXT NOT NULL,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_tokens_token_hash ON api_tokens(token_hash);

-- Add comment for documentation
COMMENT ON TABLE api_tokens IS 'Stores API tokens for user authentication';
COMMENT ON COLUMN api_tokens.token_hash IS 'Bcrypt hash of the API token (only stored once during creation)';
COMMENT ON COLUMN api_tokens.expires_at IS 'Optional expiration timestamp for the token';

-- Enable row-level security (if using RLS)
ALTER TABLE api_tokens ENABLE ROW LEVEL SECURITY;
