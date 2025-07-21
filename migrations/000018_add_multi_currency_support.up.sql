-- Add multi-currency support to transactions
ALTER TABLE transactions
ADD COLUMN original_amount NUMERIC(19, 6) NULL,
ADD COLUMN original_currency_code VARCHAR(3) NULL,
ADD COLUMN exchange_rate_id VARCHAR(255) NULL,
ADD CONSTRAINT fk_exchange_rate FOREIGN KEY (exchange_rate_id) REFERENCES exchange_rates(exchange_rate_id),
ADD CONSTRAINT chk_foreign_currency 
    CHECK (
        -- Either both original fields are NULL (single currency)
        (original_amount IS NULL AND original_currency_code IS NULL) OR
        -- Or both are set (multi-currency)
        (original_amount IS NOT NULL AND original_currency_code IS NOT NULL)
    );

-- Add index for exchange rate lookups
CREATE INDEX idx_transactions_exchange_rate_id ON transactions(exchange_rate_id);

-- Add comment to explain the new columns
COMMENT ON COLUMN transactions.original_amount IS 'The original amount in the transaction''s currency before conversion';
COMMENT ON COLUMN transactions.original_currency_code IS 'The original currency code of the transaction';
COMMENT ON COLUMN transactions.exchange_rate_id IS 'Reference to the exchange rate used for currency conversion';

-- Update the existing currency_code column comment to clarify its purpose
COMMENT ON COLUMN transactions.currency_code IS 'The base currency of the journal entry (matches journal.currency_code)';

-- Add a comment to explain the check constraint
COMMENT ON CONSTRAINT chk_foreign_currency ON transactions IS 'Ensures that both original_amount and original_currency_code are either both NULL or both NOT NULL';
