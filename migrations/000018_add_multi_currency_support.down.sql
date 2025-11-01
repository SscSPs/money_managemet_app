-- Drop the index first
DROP INDEX IF EXISTS idx_transactions_exchange_rate_id;

-- Drop the foreign key constraint
ALTER TABLE transactions 
DROP CONSTRAINT IF EXISTS fk_exchange_rate;

-- Drop the check constraint
ALTER TABLE transactions 
DROP CONSTRAINT IF EXISTS chk_foreign_currency;

-- Remove the columns
ALTER TABLE transactions
DROP COLUMN IF EXISTS original_amount,
DROP COLUMN IF EXISTS original_currency_code,
DROP COLUMN IF EXISTS exchange_rate_id;
