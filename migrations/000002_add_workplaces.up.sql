-- migrations/000002_add_workplaces.up.sql

BEGIN;

-- Create workplaces table
CREATE TABLE workplaces (
    workplace_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL REFERENCES users(user_id),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_by VARCHAR(255) NOT NULL REFERENCES users(user_id)
);

COMMENT ON TABLE workplaces IS 'Stores workplace definitions, acting as data segregation units.';
COMMENT ON COLUMN workplaces.created_by IS 'User who initially created the workplace.';

-- Create user_workplaces junction table (Many-to-Many)
CREATE TYPE user_workplace_role AS ENUM ('ADMIN', 'MEMBER');

CREATE TABLE user_workplaces (
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    workplace_id VARCHAR(255) NOT NULL REFERENCES workplaces(workplace_id) ON DELETE CASCADE,
    role user_workplace_role NOT NULL DEFAULT 'MEMBER',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, workplace_id) -- Composite primary key
);

COMMENT ON TABLE user_workplaces IS 'Links users to workplaces and defines their role within that workplace.';

-- Add workplace_id to accounts table
-- We add it as NULLABLE first to handle potential existing rows,
-- then populate it (manually or via script outside migration), then set NOT NULL.
-- For simplicity here, we assume we can add it as NOT NULL if the table is new/empty
-- or if a default can be immediately assigned.
-- Adding as NOT NULL directly here, assuming it's acceptable for this stage.
ALTER TABLE accounts
ADD COLUMN workplace_id VARCHAR(255); -- Add column first

-- Update existing accounts - Requires a strategy!
-- Example: Assign existing accounts to the workplace of their creator?
-- This requires joining accounts with users and potentially pre-creating workplaces.
-- Skipping automatic population in this basic migration.
-- UPDATE accounts a SET workplace_id = (SELECT w.workplace_id FROM user_workplaces w WHERE w.user_id = a.created_by LIMIT 1) WHERE a.workplace_id IS NULL;

-- Add NOT NULL constraint (assuming population happened or table was empty)
ALTER TABLE accounts
ALTER COLUMN workplace_id SET NOT NULL;

-- Add foreign key constraint
ALTER TABLE accounts
ADD CONSTRAINT fk_accounts_workplace
FOREIGN KEY (workplace_id) REFERENCES workplaces(workplace_id) ON DELETE RESTRICT; -- Prevent deleting workplace if accounts exist

CREATE INDEX idx_accounts_workplace_id ON accounts(workplace_id);

COMMENT ON COLUMN accounts.workplace_id IS 'The workplace this account belongs to.';

-- Add workplace_id to journals table
ALTER TABLE journals
ADD COLUMN workplace_id VARCHAR(255);

-- Update existing journals - Requires a strategy!
-- Skipping automatic population.
-- UPDATE journals j SET workplace_id = (SELECT w.workplace_id FROM user_workplaces w WHERE w.user_id = j.created_by LIMIT 1) WHERE j.workplace_id IS NULL;

-- Add NOT NULL constraint
ALTER TABLE journals
ALTER COLUMN workplace_id SET NOT NULL;

-- Add foreign key constraint
ALTER TABLE journals
ADD CONSTRAINT fk_journals_workplace
FOREIGN KEY (workplace_id) REFERENCES workplaces(workplace_id) ON DELETE RESTRICT; -- Prevent deleting workplace if journals exist

CREATE INDEX idx_journals_workplace_id ON journals(workplace_id);

COMMENT ON COLUMN journals.workplace_id IS 'The workplace this journal belongs to.';

COMMIT; 