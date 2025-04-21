-- Enable UUID generation if not already enabled
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp"; -- Use this if you prefer native UUID type

-- Function to update last_updated_at column
CREATE OR REPLACE FUNCTION update_last_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.last_updated_at = timezone('utc', now());
   RETURN NEW;
END;
$$ language 'plpgsql';

-- Users table
CREATE TABLE users (
    user_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    created_by VARCHAR(255), -- Assuming FK to users table, but nullable for initial/system actions
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    last_updated_by VARCHAR(255) -- FK to users table
    -- Add FK constraints if needed:
    -- CONSTRAINT fk_created_by FOREIGN KEY (created_by) REFERENCES users(user_id),
    -- CONSTRAINT fk_last_updated_by FOREIGN KEY (last_updated_by) REFERENCES users(user_id)
);

CREATE TRIGGER trigger_users_update_last_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_last_updated_at_column();

-- Currencies table
CREATE TABLE currencies (
    currency_code VARCHAR(10) PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    created_by VARCHAR(255) REFERENCES users(user_id),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    last_updated_by VARCHAR(255) REFERENCES users(user_id)
);

CREATE TRIGGER trigger_currencies_update_last_updated_at
BEFORE UPDATE ON currencies
FOR EACH ROW EXECUTE FUNCTION update_last_updated_at_column();

-- Account Types Enum (using CHECK constraint)
-- Note: PostgreSQL also supports CREATE TYPE ... AS ENUM, which might be preferred.
-- CREATE TYPE account_type_enum AS ENUM ('ASSET', 'LIABILITY', 'EQUITY', 'INCOME', 'EXPENSE');

-- Accounts table
CREATE TABLE accounts (
    account_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    account_type VARCHAR(50) NOT NULL CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'INCOME', 'EXPENSE')),
    currency_code VARCHAR(10) NOT NULL,
    parent_account_id VARCHAR(255),
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    created_by VARCHAR(255) REFERENCES users(user_id),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    last_updated_by VARCHAR(255) REFERENCES users(user_id),
    CONSTRAINT fk_currency FOREIGN KEY (currency_code) REFERENCES currencies(currency_code),
    CONSTRAINT fk_parent_account FOREIGN KEY (parent_account_id) REFERENCES accounts(account_id) ON DELETE SET NULL -- Or RESTRICT depending on desired behavior
);

CREATE INDEX idx_accounts_currency_code ON accounts(currency_code);
CREATE INDEX idx_accounts_parent_account_id ON accounts(parent_account_id);
CREATE INDEX idx_accounts_account_type ON accounts(account_type);

CREATE TRIGGER trigger_accounts_update_last_updated_at
BEFORE UPDATE ON accounts
FOR EACH ROW EXECUTE FUNCTION update_last_updated_at_column();

-- Exchange Rates table
CREATE TABLE exchange_rates (
    exchange_rate_id VARCHAR(255) PRIMARY KEY,
    from_currency_code VARCHAR(10) NOT NULL,
    to_currency_code VARCHAR(10) NOT NULL,
    rate NUMERIC(19, 8) NOT NULL CHECK (rate > 0), -- Precision and scale can be adjusted
    date_effective DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    created_by VARCHAR(255) REFERENCES users(user_id),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    last_updated_by VARCHAR(255) REFERENCES users(user_id),
    CONSTRAINT fk_from_currency FOREIGN KEY (from_currency_code) REFERENCES currencies(currency_code),
    CONSTRAINT fk_to_currency FOREIGN KEY (to_currency_code) REFERENCES currencies(currency_code),
    CONSTRAINT uq_exchange_rate_date UNIQUE (from_currency_code, to_currency_code, date_effective)
);

CREATE INDEX idx_exchange_rates_from_currency ON exchange_rates(from_currency_code);
CREATE INDEX idx_exchange_rates_to_currency ON exchange_rates(to_currency_code);
CREATE INDEX idx_exchange_rates_date_effective ON exchange_rates(date_effective);

CREATE TRIGGER trigger_exchange_rates_update_last_updated_at
BEFORE UPDATE ON exchange_rates
FOR EACH ROW EXECUTE FUNCTION update_last_updated_at_column();

-- Journal Status Enum (using CHECK constraint)
-- CREATE TYPE journal_status_enum AS ENUM ('POSTED', 'REVERSED');

-- Journals table
CREATE TABLE journals (
    journal_id VARCHAR(255) PRIMARY KEY,
    journal_date DATE NOT NULL,
    description TEXT,
    currency_code VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'POSTED' CHECK (status IN ('POSTED', 'REVERSED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    created_by VARCHAR(255) REFERENCES users(user_id),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    last_updated_by VARCHAR(255) REFERENCES users(user_id),
    CONSTRAINT fk_journal_currency FOREIGN KEY (currency_code) REFERENCES currencies(currency_code)
);

CREATE INDEX idx_journals_currency_code ON journals(currency_code);
CREATE INDEX idx_journals_journal_date ON journals(journal_date);
CREATE INDEX idx_journals_status ON journals(status);

CREATE TRIGGER trigger_journals_update_last_updated_at
BEFORE UPDATE ON journals
FOR EACH ROW EXECUTE FUNCTION update_last_updated_at_column();

-- Transaction Type Enum (using CHECK constraint)
-- CREATE TYPE transaction_type_enum AS ENUM ('DEBIT', 'CREDIT');

-- Transactions table
CREATE TABLE transactions (
    transaction_id VARCHAR(255) PRIMARY KEY,
    journal_id VARCHAR(255) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    amount NUMERIC(19, 4) NOT NULL CHECK (amount >= 0), -- Precision and scale match common currency formats
    transaction_type VARCHAR(50) NOT NULL CHECK (transaction_type IN ('DEBIT', 'CREDIT')),
    currency_code VARCHAR(10) NOT NULL, -- Should logically match journal currency
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    created_by VARCHAR(255) REFERENCES users(user_id),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc', now()),
    last_updated_by VARCHAR(255) REFERENCES users(user_id),
    CONSTRAINT fk_journal FOREIGN KEY (journal_id) REFERENCES journals(journal_id) ON DELETE CASCADE,
    CONSTRAINT fk_account FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE RESTRICT, -- Prevent deleting accounts with transactions
    CONSTRAINT fk_transaction_currency FOREIGN KEY (currency_code) REFERENCES currencies(currency_code) -- Enforce consistency, though journal FK implies it
);

CREATE INDEX idx_transactions_journal_id ON transactions(journal_id);
CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_currency_code ON transactions(currency_code);
CREATE INDEX idx_transactions_transaction_type ON transactions(transaction_type);

CREATE TRIGGER trigger_transactions_update_last_updated_at
BEFORE UPDATE ON transactions
FOR EACH ROW EXECUTE FUNCTION update_last_updated_at_column(); 