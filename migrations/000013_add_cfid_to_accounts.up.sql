-- Add cfid column to accounts table
ALTER TABLE accounts 
    ADD COLUMN cfid VARCHAR(255);

-- Create a unique index on cfid and workplace_id to ensure uniqueness within a workplace
CREATE UNIQUE INDEX idx_accounts_workplace_id_cfid ON accounts(workplace_id, cfid) 
    WHERE cfid IS NOT NULL;
