# Multi-Currency Journal Entries - Product Requirements Document (PRD)

## 1. Overview
Enable support for multi-currency transactions within a single journal entry while maintaining the existing accounting integrity and reporting capabilities.

## 2. Goals
- Allow transactions in different currencies within the same journal
- Maintain backward compatibility with existing single-currency journals
- Ensure accurate financial reporting in the journal's base currency
- Provide clear audit trail of currency conversions

## 3. User Stories

### 3.1 Multi-Currency Transaction Entry
**As a** user  
**I want to** record transactions in different currencies  
**So that** I can accurately reflect business operations in local currencies  

**Acceptance Criteria:**
- User can specify an amount and currency different from journal's currency
- System automatically converts to journal's currency using the latest exchange rate
- Both original and converted amounts are stored
- Transaction displays both amounts in the UI

### 3.2 Exchange Rate Management
**As a** user  
**I want to** view and manage exchange rates  
**So that** I can ensure accurate currency conversions  

**Acceptance Criteria:**
- System provides default exchange rates
- Users can override rates for specific transactions
- Historical rate changes are audited

## 4. Technical Specifications

### 4.1 Database Changes
```sql
ALTER TABLE transactions
ADD COLUMN original_amount DECIMAL(19, 6) NULL,
ADD COLUMN original_currency_code VARCHAR(3) NULL,
ADD COLUMN exchange_rate_id UUID NULL REFERENCES exchange_rates(exchange_rate_id),
ADD CONSTRAINT chk_foreign_currency 
    CHECK (
        (original_amount IS NULL AND original_currency_code IS NULL) OR
        (original_amount IS NOT NULL AND original_currency_code IS NOT NULL)
    );
```

### 4.2 API Changes
#### Create Transaction (New Fields)
```json
{
  "account_id": "acc_123",
  "amount": 100.00,
  "original_amount": 90.00,
  "original_currency": "EUR",
  "exchange_rate_id": "rate_123",
  "transaction_type": "DEBIT"
}
```

#### Get Transaction (Response)
```json
{
  "transaction_id": "txn_123",
  "account_id": "acc_123",
  "amount": 110.00,
  "currency": "USD",
  "original_amount": 100.00,
  "original_currency": "EUR",
  "exchange_rate": 1.1,
  "exchange_rate_id": "rate_123",
  "transaction_type": "DEBIT"
}
```

## 5. Business Rules
1. **Currency Conversion**:
   - If `original_currency` differs from journal's currency, `exchange_rate_id` is required
   - Converted amount = `original_amount * exchange_rate`

2. **Validation**:
   - Journal must balance in its base currency
   - All required currency rates must exist
   - Original amount and currency must be provided together

3. **Reporting**:
   - All reports show amounts in journal's base currency
   - Option to view original amounts in transaction currency

## 6. Implementation Phases

### Phase 1: Core Functionality
- Database schema updates
- Domain model changes
- Basic transaction creation with currency conversion

### Phase 2: Enhanced Features
- Exchange rate management UI
- Bulk transaction import/export with multi-currency support
- Reporting enhancements

### Phase 3: Advanced Features
- Automated rate updates
- Gain/loss calculations
- Multi-currency reconciliation

## 7. Success Metrics
- 100% of existing single-currency journals function unchanged
- Support for all major world currencies
- Sub-second response time for currency conversions
- Zero data loss during migration

## 8. Open Questions
1. Should we allow users to edit exchange rates after posting?
2. How should we handle historical rate changes?
3. What's the rounding policy for currency conversions?
