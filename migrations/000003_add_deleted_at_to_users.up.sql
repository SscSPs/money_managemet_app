-- Add deleted_at column for soft deletes to users table
BEGIN;

ALTER TABLE users
ADD COLUMN deleted_at TIMESTAMPTZ NULL; -- Nullable timestamp

COMMENT ON COLUMN users.deleted_at IS 'Timestamp when the user was marked as deleted (soft delete).';

-- Optional: Add an index for potentially querying non-deleted users frequently
-- CREATE INDEX idx_users_deleted_at_null ON users (deleted_at) WHERE deleted_at IS NULL;
-- Note: The repository queries already use "WHERE deleted_at IS NULL",
-- so a partial index like the one above can significantly speed up those lookups.
-- Let's add it.
CREATE INDEX idx_users_deleted_at_null ON users (deleted_at) WHERE deleted_at IS NULL;


COMMIT; 