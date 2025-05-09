-- Filename: migrations/000011_add_refresh_token_fields_to_users.up.sql

ALTER TABLE users
ADD COLUMN refresh_token_hash TEXT NULL,
ADD COLUMN refresh_token_expiry_time TIMESTAMPTZ NULL;

COMMENT ON COLUMN users.refresh_token_hash IS 'Stores the hashed version of the refresh token.';
COMMENT ON COLUMN users.refresh_token_expiry_time IS 'Stores the expiry timestamp for the refresh token.';
