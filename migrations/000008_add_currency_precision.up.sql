-- Add precision column to currencies table
ALTER TABLE currencies ADD COLUMN precision INT NOT NULL DEFAULT 2;

-- Add a comment to explain the column
COMMENT ON COLUMN currencies.precision IS 'Number of decimal places for the currency (e.g., 2 for USD, 0 for JPY, 8+ for cryptocurrencies)';

-- Update known currencies with correct precision values
UPDATE currencies SET precision = 0 WHERE currency_code IN ('JPY', 'KRW', 'VND', 'HUF');
UPDATE currencies SET precision = 3 WHERE currency_code IN ('KWD', 'BHD', 'OMR');
UPDATE currencies SET precision = 8 WHERE currency_code IN ('BTC', 'XBT'); -- Bitcoin
UPDATE currencies SET precision = 18 WHERE currency_code IN ('ETH'); -- Ethereum
-- All other currencies will keep the default precision of 2 