ALTER TABLE users
DROP COLUMN auth_provider,
DROP COLUMN provider_user_id,
DROP COLUMN email;

-- Revert password_hash to NOT NULL if it was previously. 
-- This might fail if there are actual NULL values. Handle with care or make it conditional.
-- For simplicity, assuming it was NOT NULL and can be reverted if no NULLs were introduced.
ALTER TABLE users
ALTER COLUMN password_hash SET NOT NULL;

DROP INDEX IF EXISTS idx_users_auth_provider_provider_user_id;
