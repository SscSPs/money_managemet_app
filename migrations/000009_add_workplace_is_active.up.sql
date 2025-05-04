BEGIN;

-- Add is_active column to the workplaces table
ALTER TABLE workplaces
ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN workplaces.is_active IS 'Indicates whether the workplace is active (true) or disabled (false)';

-- Create an index on the is_active column for efficient filtering
CREATE INDEX idx_workplaces_is_active ON workplaces(is_active);

-- Add the REMOVED value to the user_workplace_role enum
-- Postgres doesn't allow direct ALTER TYPE ADD VALUE in transactions
-- so we use a workaround with a temporary type
CREATE TYPE user_workplace_role_new AS ENUM ('ADMIN', 'MEMBER', 'REMOVED');

-- First, drop the default constraint on the role column
ALTER TABLE user_workplaces 
  ALTER COLUMN role DROP DEFAULT;

-- Update user_workplaces table to use the new enum type
ALTER TABLE user_workplaces 
  ALTER COLUMN role TYPE user_workplace_role_new 
  USING (role::text::user_workplace_role_new);

-- Add back the default constraint with the new type
ALTER TABLE user_workplaces
  ALTER COLUMN role SET DEFAULT 'MEMBER'::user_workplace_role_new;

-- Drop the old enum type
DROP TYPE user_workplace_role;

-- Rename the new enum type to the original name
ALTER TYPE user_workplace_role_new RENAME TO user_workplace_role;

-- Update the default to use the renamed type
ALTER TABLE user_workplaces
  ALTER COLUMN role SET DEFAULT 'MEMBER'::user_workplace_role;

COMMENT ON TYPE user_workplace_role IS 'Role of a user within a workspace: ADMIN, MEMBER, or REMOVED (for users who have been removed)';

COMMIT; 