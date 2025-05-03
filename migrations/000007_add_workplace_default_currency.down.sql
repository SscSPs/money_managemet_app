-- Remove the foreign key constraint
ALTER TABLE workplaces DROP CONSTRAINT IF EXISTS fk_workplace_default_currency;

-- Remove the default_currency_code column
ALTER TABLE workplaces DROP COLUMN IF EXISTS default_currency_code; 