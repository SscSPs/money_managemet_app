-- migrations/000002_add_workplaces.down.sql

BEGIN;

-- Remove constraints and columns from dependent tables first
ALTER TABLE journals DROP CONSTRAINT fk_journals_workplace;
ALTER TABLE journals DROP COLUMN workplace_id;

ALTER TABLE accounts DROP CONSTRAINT fk_accounts_workplace;
ALTER TABLE accounts DROP COLUMN workplace_id;

-- Drop the junction table
DROP TABLE user_workplaces;

-- Drop the workplaces table
DROP TABLE workplaces;

-- Drop the custom enum type
DROP TYPE user_workplace_role;

COMMIT; 