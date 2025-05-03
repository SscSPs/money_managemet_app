-- Add default_currency_code column to workplaces table
ALTER TABLE workplaces ADD COLUMN default_currency_code VARCHAR(3);

-- Add a comment to explain the column
COMMENT ON COLUMN workplaces.default_currency_code IS 'The default currency code for this workplace. Used for reports and as default for new accounts.';

-- Add foreign key constraint to ensure the currency exists
ALTER TABLE workplaces 
ADD CONSTRAINT fk_workplace_default_currency 
FOREIGN KEY (default_currency_code) 
REFERENCES currencies(currency_code);

-- Update existing workplaces to use the first available currency
-- This is safe even if no currencies exist (column will remain NULL)
DO $$
DECLARE
    first_currency VARCHAR(3);
BEGIN
    -- Find the first currency in the system
    SELECT currency_code INTO first_currency FROM currencies ORDER BY created_at LIMIT 1;
    
    -- If we found a currency, update all workplaces that don't have a default currency
    IF first_currency IS NOT NULL THEN
        UPDATE workplaces SET default_currency_code = first_currency WHERE default_currency_code IS NULL;
    ELSE
        RAISE WARNING 'No currencies found in the system. Workplaces will have NULL default_currency_code.';
    END IF;
END $$; 