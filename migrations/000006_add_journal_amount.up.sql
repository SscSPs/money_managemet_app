-- Add amount column to journals table
ALTER TABLE journals ADD COLUMN amount DECIMAL(19,4) NOT NULL DEFAULT 0;

-- Add a comment to explain the column
COMMENT ON COLUMN journals.amount IS 'Total amount of the journal (sum of all debit transactions)';

-- Update existing journal amounts using a calculation from transactions
-- Calculate amount as sum of debit transactions for each journal
UPDATE journals j
SET amount = (
    SELECT COALESCE(SUM(t.amount), 0)
    FROM transactions t
    WHERE t.journal_id = j.journal_id AND t.transaction_type = 'DEBIT'
);

-- Validate that all journals have an amount
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM journals WHERE amount = 0 AND status = 'POSTED') THEN
        RAISE WARNING 'Some posted journals have zero amount. Check for data consistency.';
    END IF;
END $$; 