-- Add columns for journal reversal links
ALTER TABLE journals
ADD COLUMN original_journal_id TEXT NULL,
ADD COLUMN reversing_journal_id TEXT NULL;

-- Add foreign key constraints (optional but recommended)
-- Using DEFERRABLE INITIALLY DEFERRED allows updates to both related rows before constraint check
ALTER TABLE journals
ADD CONSTRAINT fk_original_journal
FOREIGN KEY (original_journal_id)
REFERENCES journals(journal_id)
ON DELETE SET NULL
DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE journals
ADD CONSTRAINT fk_reversing_journal
FOREIGN KEY (reversing_journal_id)
REFERENCES journals(journal_id)
ON DELETE SET NULL
DEFERRABLE INITIALLY DEFERRED;

-- Optional: Add indexes for faster lookup
CREATE INDEX IF NOT EXISTS idx_journals_original_journal_id ON journals (original_journal_id);
CREATE INDEX IF NOT EXISTS idx_journals_reversing_journal_id ON journals (reversing_journal_id); 