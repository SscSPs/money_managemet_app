-- Filename: migrations/000011_add_refresh_token_fields_to_users.down.sql

ALTER TABLE users
DROP COLUMN IF EXISTS refresh_token_expiry_time,
DROP COLUMN IF EXISTS refresh_token_hash;
