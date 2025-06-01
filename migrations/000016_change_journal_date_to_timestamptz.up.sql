-- Change journal_date from DATE to TIMESTAMPTZ to store both date and time
ALTER TABLE journals 
ALTER COLUMN journal_date TYPE TIMESTAMPTZ 
USING journal_date AT TIME ZONE 'UTC';
