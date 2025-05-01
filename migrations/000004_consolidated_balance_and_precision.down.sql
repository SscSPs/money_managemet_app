-- Revert Consolidated Migration: Remove balance columns and revert precision

-- Revert precision for rate column in exchange_rates
ALTER TABLE exchange_rates
    ALTER COLUMN rate TYPE NUMERIC(19, 8); -- Revert to original precision

-- Revert precision for amount column in transactions
ALTER TABLE transactions
    ALTER COLUMN amount TYPE NUMERIC(19, 4);

-- Remove running_balance from transactions
ALTER TABLE transactions
DROP COLUMN IF EXISTS running_balance;

-- Remove balance from accounts
ALTER TABLE accounts
DROP COLUMN IF EXISTS balance;

-- Revert comments (optional)
-- COMMENT ON COLUMN transactions.amount IS 'Positive value; Precision and scale match common currency formats';
-- COMMENT ON COLUMN exchange_rates.rate IS 'Exchange rate value (e.g., 1 USD = X target)'; 