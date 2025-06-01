-- Revert journal_date back to DATE type
ALTER TABLE journals 
ALTER COLUMN journal_date TYPE DATE 
USING DATE(journal_date);
