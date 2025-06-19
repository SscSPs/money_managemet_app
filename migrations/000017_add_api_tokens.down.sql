-- Drop indexes
DROP INDEX IF EXISTS idx_api_tokens_user_id;
DROP INDEX IF EXISTS idx_api_tokens_token_hash;

-- Drop the api_tokens table
DROP TABLE IF EXISTS api_tokens;
