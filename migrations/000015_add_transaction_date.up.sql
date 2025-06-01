-- Add transaction_date column to transactions table
ALTER TABLE transactions 
ADD COLUMN transaction_date DATE NOT NULL DEFAULT CURRENT_DATE;

-- Create index on transaction_date for better query performance
CREATE INDEX idx_transactions_transaction_date ON transactions(transaction_date);

-- Update the default value to use the journal's date for existing records
UPDATE transactions t
SET transaction_date = j.journal_date
FROM journals j
WHERE t.journal_id = j.journal_id;
