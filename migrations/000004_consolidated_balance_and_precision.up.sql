-- Consolidated Migration: Add balance columns and increase numeric precision

-- Add running_balance to transactions with increased precision
ALTER TABLE transactions
ADD COLUMN running_balance NUMERIC(57, 18) NULL;
COMMENT ON COLUMN transactions.running_balance IS 'Running balance with increased precision/scale (57, 18).';

-- Add balance to accounts with increased precision
ALTER TABLE accounts
ADD COLUMN balance NUMERIC(57, 18) NOT NULL DEFAULT 0;
COMMENT ON COLUMN accounts.balance IS 'Account balance with increased precision/scale (57, 18).';

-- Update precision for existing amount column in transactions
ALTER TABLE transactions
    ALTER COLUMN amount TYPE NUMERIC(57, 18);
COMMENT ON COLUMN transactions.amount IS 'Monetary amount with increased precision/scale (57, 18).';

-- Update precision for existing rate column in exchange_rates
ALTER TABLE exchange_rates
    ALTER COLUMN rate TYPE NUMERIC(57, 18);
COMMENT ON COLUMN exchange_rates.rate IS 'Exchange rate with increased precision/scale (57, 18).'; 