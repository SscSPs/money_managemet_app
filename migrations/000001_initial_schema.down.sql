-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_transactions_update_last_updated_at ON transactions;
DROP TRIGGER IF EXISTS trigger_journals_update_last_updated_at ON journals;
DROP TRIGGER IF EXISTS trigger_exchange_rates_update_last_updated_at ON exchange_rates;
DROP TRIGGER IF EXISTS trigger_accounts_update_last_updated_at ON accounts;
DROP TRIGGER IF EXISTS trigger_currencies_update_last_updated_at ON currencies;
DROP TRIGGER IF EXISTS trigger_users_update_last_updated_at ON users;

-- Drop tables in reverse order of creation (considering dependencies)
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS journals;
DROP TABLE IF EXISTS exchange_rates;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS currencies;
DROP TABLE IF EXISTS users;

-- Drop custom types if they were created
-- DROP TYPE IF EXISTS transaction_type_enum;
-- DROP TYPE IF EXISTS journal_status_enum;
-- DROP TYPE IF EXISTS account_type_enum;

-- Drop the function
DROP FUNCTION IF EXISTS update_last_updated_at_column();

-- Disable UUID generation if it was enabled here
-- DROP EXTENSION IF EXISTS "uuid-ossp"; 