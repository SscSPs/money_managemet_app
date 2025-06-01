-- Drop the index first
DROP INDEX IF EXISTS idx_transactions_transaction_date;

-- Remove the transaction_date column
ALTER TABLE transactions 
DROP COLUMN IF EXISTS transaction_date;
