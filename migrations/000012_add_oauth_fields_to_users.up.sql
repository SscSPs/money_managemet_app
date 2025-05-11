ALTER TABLE users
ADD COLUMN auth_provider TEXT NULL,
ADD COLUMN provider_user_id TEXT NULL,
ADD COLUMN email VARCHAR(255) UNIQUE;

ALTER TABLE users
ALTER COLUMN password_hash DROP NOT NULL;

-- Optional: Add an index for faster lookups by provider and provider_user_id
CREATE INDEX IF NOT EXISTS idx_users_auth_provider_provider_user_id ON users (auth_provider, provider_user_id);

COMMENT ON COLUMN users.email IS 'User''s email address, unique across all users.';
