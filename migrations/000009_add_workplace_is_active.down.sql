BEGIN;

-- Drop the index first
DROP INDEX IF EXISTS idx_workplaces_is_active;

-- Then drop the column
ALTER TABLE workplaces
DROP COLUMN IF EXISTS is_active;

-- next convert any REMOVED users to MEMBER to avoid losing data
UPDATE user_workplaces 
SET role = 'MEMBER'::user_workplace_role 
WHERE role = 'REMOVED'::user_workplace_role;

-- Create a new enum without the REMOVED value
CREATE TYPE user_workplace_role_old AS ENUM ('ADMIN', 'MEMBER');

-- Drop the default constraint
ALTER TABLE user_workplaces
  ALTER COLUMN role DROP DEFAULT;

-- Update the table to use the new type
ALTER TABLE user_workplaces
  ALTER COLUMN role TYPE user_workplace_role_old
  USING (role::text::user_workplace_role_old);

-- Add back the default with the new type
ALTER TABLE user_workplaces
  ALTER COLUMN role SET DEFAULT 'MEMBER'::user_workplace_role_old;

-- Drop the old type
DROP TYPE user_workplace_role;

-- Rename the new type back to the original name
ALTER TYPE user_workplace_role_old RENAME TO user_workplace_role;

-- Update the default to use the renamed type
ALTER TABLE user_workplaces
  ALTER COLUMN role SET DEFAULT 'MEMBER'::user_workplace_role;

COMMIT; 