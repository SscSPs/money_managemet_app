-- Drop the unique index first
DROP INDEX IF EXISTS idx_accounts_workplace_id_cfid;

-- Remove the cfid column
ALTER TABLE accounts 
    DROP COLUMN IF EXISTS cfid;
