ALTER TABLE users
ADD COLUMN username VARCHAR(255) UNIQUE NOT NULL,
ADD COLUMN password_hash VARCHAR(255) NOT NULL;

-- Optionally, migrate existing users to have default username/password if needed.
