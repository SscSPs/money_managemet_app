-- Remove foreign key constraints first (if they exist)
ALTER TABLE journals
DROP CONSTRAINT IF EXISTS fk_original_journal,
DROP CONSTRAINT IF EXISTS fk_reversing_journal;

-- Remove indexes (if they exist)
DROP INDEX IF EXISTS idx_journals_original_journal_id;
DROP INDEX IF EXISTS idx_journals_reversing_journal_id;

-- Remove columns for journal reversal links
ALTER TABLE journals
DROP COLUMN IF EXISTS original_journal_id,
DROP COLUMN IF EXISTS reversing_journal_id; 