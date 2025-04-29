-- Remove deleted_at column from users table
BEGIN;

-- Remove the index first if it exists
DROP INDEX IF EXISTS idx_users_deleted_at_null;

ALTER TABLE users
DROP COLUMN IF EXISTS deleted_at;

COMMIT; 