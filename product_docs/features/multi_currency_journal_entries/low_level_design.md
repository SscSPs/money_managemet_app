# Multi-Currency Journal Entries - Low-Level Design

## 1. Request/Response Structures

### 1.1 Domain Model Updates

#### Transaction Model (Updated)

```go
type Transaction struct {
    // Existing fields
    TransactionID    string          `json:"transactionID"`
    JournalID        string          `json:"journalID"`
    AccountID        string          `json:"accountID"`
    Amount          decimal.Decimal `json:"amount"`
    TransactionType TransactionType `json:"transactionType"`
    CurrencyCode    string          `json:"currencyCode"` // Will match journal's base currency
    
    // New fields for multi-currency support
    OriginalAmount      *decimal.Decimal `json:"originalAmount,omitempty"`
    OriginalCurrency    *string          `json:"originalCurrency,omitempty"`
    ExchangeRateID      *string          `json:"exchangeRateId,omitempty"`
    
    // Rest of the existing fields
    Notes            string    `json:"notes"`
    TransactionDate  time.Time `json:"transactionDate"`
    AuditFields                    // Embed audit fields
    RunningBalance     decimal.Decimal `json:"runningBalance"`
    JournalDate        time.Time       `json:"journalDate"`
    JournalDescription string          `json:"journalDescription"`
}
```

### 1.2 Create Journal Request (Updated)

```go
// CreateJournalRequest defines data for creating a journal entry
type CreateJournalRequest struct {
    Date         time.Time                  `json:"date" binding:"required"`
    Description  string                     `json:"description"`
    CurrencyCode string                     `json:"currencyCode" binding:"required,iso4217"` // Base currency for the journal
    Transactions []CreateTransactionRequest `json:"transactions" binding:"required,min=2,dive"`
}

// CreateTransactionRequest defines data for a single transaction
type CreateTransactionRequest struct {
    AccountID          string                 `json:"accountID" binding:"required,uuid"`
    OriginalAmount     decimal.Decimal        `json:"originalAmount" binding:"required,decimal_gtz"`
    OriginalCurrency   *string                `json:"originalCurrency,omitempty"`
    ExchangeRateID     *string                `json:"exchangeRateId,omitempty"`
    TransactionType    domain.TransactionType `json:"transactionType" binding:"required,oneof=DEBIT CREDIT"`
    TransactionDate    *time.Time             `json:"transactionDate,omitempty"`
    Notes              string                 `json:"notes"`
}
```

### 1.2 Transaction Response (Updated)

```go
type TransactionResponse struct {
    TransactionID       string                 `json:"transactionID"`
    JournalID           string                 `json:"journalID"`
    AccountID           string                 `json:"accountID"`
    Amount              decimal.Decimal        `json:"amount"`              // In journal's base currency
    OriginalAmount      *decimal.Decimal       `json:"originalAmount,omitempty"`
    OriginalCurrency    *string                `json:"originalCurrency,omitempty"`
    ExchangeRate        *decimal.Decimal       `json:"exchangeRate,omitempty"`
    ExchangeRateID      *string                `json:"exchangeRateId,omitempty"`
    TransactionType     domain.TransactionType `json:"transactionType"`
    Notes               string                 `json:"notes"`
    TransactionDate     time.Time              `json:"transactionDate"`
    CreatedAt           time.Time              `json:"createdAt"`
    CreatedBy           string                 `json:"createdBy"`
    RunningBalance      decimal.Decimal        `json:"runningBalance,omitempty"`
}
```

## 2. Service Layer Changes

## 2. Service Layer Changes

### 2.1 Exchange Rate Service Updates

#### 2.1.1 Interface Updates

```go
type ExchangeRateSvcFacade interface {
    // Existing methods...
    
    // GetExchangeRateByID retrieves an exchange rate by its ID
    GetExchangeRateByID(ctx context.Context, id string) (*domain.ExchangeRate, error)
    
    // GetExchangeRate gets the exchange rate between two currencies for a specific date
    GetExchangeRate(ctx context.Context, fromCode, toCode string, date time.Time) (*domain.ExchangeRate, error)
}
```

### 2.2 Journal Service Updates

#### 2.2.1 Create Journal Flow

```go
func (s *journalService) CreateJournal(
    ctx context.Context, 
    workplaceID string, 
    req dto.CreateJournalRequest, 
    creatorUserID string,
) (*domain.Journal, error) {
    // 1. Validate request
    if err := s.validateCreateJournalRequest(ctx, &req); err != nil {
        return nil, err
    }

    // 2. Convert transactions to domain model
    transactions := make([]domain.Transaction, 0, len(req.Transactions))
    for _, txnReq := range req.Transactions {
        txn, err := s.convertToDomainTransaction(
            ctx,
            txnReq,
            req.CurrencyCode,
            creatorUserID,
        )
        if err != nil {
            return nil, err
        }
        transactions = append(transactions, *txn)
    }

    // 3. Validate journal balance in base currency
    if err := s.validateJournalBalance(transactions); err != nil {
        return nil, fmt.Errorf("invalid journal: %w", err)
    }

    // 4. Create and save journal
    journal := &domain.Journal{
        JournalID:    uuid.NewString(),
        WorkplaceID:  workplaceID,
        JournalDate:  req.Date,
        Description:  req.Description,
        CurrencyCode: req.CurrencyCode,
        Status:       domain.Posted,
        Amount:       s.calculateJournalAmount(transactions),
        AuditFields: domain.AuditFields{
            CreatedAt: time.Now(),
            CreatedBy: creatorUserID,
        },
    }

    // 5. Save to database within a transaction
    return s.journalRepo.SaveJournalWithTransactions(ctx, journal, transactions)
}

// convertToDomainTransaction converts a transaction request to domain model
func (s *journalService) convertToDomainTransaction(
    ctx context.Context,
    req dto.CreateTransactionRequest,
    journalCurrency string,
    creatorUserID string,
) (*domain.Transaction, error) {
    txn := &domain.Transaction{
        TransactionID:   uuid.NewString(),
        AccountID:       req.AccountID,
        TransactionType: req.TransactionType,
        Notes:           req.Notes,
        TransactionDate: timeNowPtr(req.TransactionDate),
        AuditFields: domain.AuditFields{
            CreatedAt: time.Now(),
            CreatedBy: creatorUserID,
        },
    }

    // Handle currency conversion if needed
    if req.OriginalCurrency != "" && req.OriginalCurrency != journalCurrency {
        if req.ExchangeRateID == nil {
            return nil, fmt.Errorf("exchangeRateId is required for currency conversion")
        }
        
        // Get exchange rate
        rate, err := s.exchangeRateSvc.GetExchangeRateByID(ctx, *req.ExchangeRateID)
        if err != nil {
            return nil, fmt.Errorf("invalid exchange rate: %w", err)
        }
        
        if rate.FromCurrencyCode != req.OriginalCurrency || 
           rate.ToCurrencyCode != journalCurrency {
            return nil, fmt.Errorf("exchange rate currency mismatch")
        }
        
        txn.OriginalAmount = req.OriginalAmount
        txn.OriginalCurrency = req.OriginalCurrency
        txn.ExchangeRateID = req.ExchangeRateID
        txn.Amount = req.OriginalAmount.Mul(rate.Rate)
    } else {
        // No conversion needed
        txn.Amount = req.OriginalAmount
    }
    
    return txn, nil
}
```

## 3. Database Layer

## 3. Repository Layer

### 3.1 Repository Interface Updates

#### Exchange Rate Repository

```go
type ExchangeRateRepositoryFacade interface {
    // FindExchangeRate retrieves an exchange rate between two currencies
    FindExchangeRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (*domain.ExchangeRate, error)
    
    // SaveExchangeRate persists a new exchange rate
    SaveExchangeRate(ctx context.Context, rate domain.ExchangeRate) error
    
    // New method for getting exchange rate by ID
    GetExchangeRateByID(ctx context.Context, id string) (*domain.ExchangeRate, error)
}
```

#### Journal Repository

```go
type JournalRepositoryFacade interface {
    // Existing journal methods...
    
    // SaveJournal persists a journal and its transactions, updating account balances
    SaveJournal(
        ctx context.Context, 
        journal domain.Journal, 
        transactions []domain.Transaction, 
        balanceChanges map[string]decimal.Decimal,
    ) error
    
    // New method to save transactions with exchange rate information
    SaveTransactions(ctx context.Context, transactions []domain.Transaction) error
}
```

### 3.2 SQL Migrations

```sql
-- Add new columns to transactions table
ALTER TABLE transactions
ADD COLUMN original_amount DECIMAL(19, 6) NULL,
ADD COLUMN original_currency_code CHAR(3) NULL,
ADD COLUMN exchange_rate_id UUID NULL REFERENCES exchange_rates(exchange_rate_id);

-- Add check constraint for currency consistency
ALTER TABLE transactions
ADD CONSTRAINT chk_foreign_currency 
    CHECK (
        -- Either both original fields are NULL (single currency)
        (original_amount IS NULL AND original_currency_code IS NULL) OR
        -- Or both are set (multi-currency)
        (original_amount IS NOT NULL AND original_currency_code IS NOT NULL)
    );

-- Add index for better query performance
CREATE INDEX idx_transactions_currency ON transactions(original_currency_code);
CREATE INDEX idx_transactions_exchange_rate ON transactions(exchange_rate_id);
```

## 4. API Endpoints

### 4.1 Create Journal

```http
POST /api/v1/workplaces/{workplaceId}/journals
Content-Type: application/json
Authorization: Bearer {token}

{
  "date": "2025-07-10T00:00:00Z",
  "description": "Multi-currency expense",
  "currencyCode": "USD",
  "transactions": [
    {
      "accountID": "550e8400-e29b-41d4-a716-446655440001",
      "originalAmount": 100.00,
      "originalCurrency": "EUR",
      "exchangeRateId": "550e8400-e29b-41d4-a716-446655440010",
      "transactionType": "DEBIT",
      "notes": "Office supplies"
    },
    {
      "accountID": "550e8400-e29b-41d4-a716-446655440002",
      "originalAmount": 1200.00,
      "originalCurrency": "GBP",
      "exchangeRateId": "550e8400-e29b-41d4-a716-446655440011",
      "transactionType": "CREDIT",
      "notes": "Vendor payment"
    }
  ]
}

HTTP/1.1 201 Created
{
  "journalID": "660e8400-e29b-41d4-a716-446655440100",
  "workplaceID": "550e8400-e29b-41d4-a716-446655440000",
  "date": "2025-07-10T00:00:00Z",
  "description": "Multi-currency expense",
  "currencyCode": "USD",
  "status": "POSTED",
  "amount": 2500.00,
  "createdAt": "2025-07-09T12:00:00Z",
  "createdBy": "user-123",
  "transactions": [
    {
      "transactionID": "770e8400-e29b-41d4-a716-446655440101",
      "accountID": "550e8400-e29b-41d4-a716-446655440001",
      "amount": 110.00,
      "originalAmount": 100.00,
      "originalCurrency": "EUR",
      "exchangeRate": 1.1,
      "exchangeRateId": "550e8400-e29b-41d4-a716-446655440010",
      "transactionType": "DEBIT",
      "notes": "Office supplies",
      "transactionDate": "2025-07-10T00:00:00Z"
    },
    {
      "transactionID": "770e8400-e29b-41d4-a716-446655440102",
      "accountID": "550e8400-e29b-41d4-a716-446655440002",
      "amount": 2390.00,
      "originalAmount": 1200.00,
      "originalCurrency": "GBP",
      "exchangeRate": 1.991667,
      "exchangeRateId": "550e8400-e29b-41d4-a716-446655440011",
      "transactionType": "CREDIT",
      "notes": "Vendor payment",
      "transactionDate": "2025-07-10T00:00:00Z"
    }
  ]
}
```

## 5. Validation Rules

### 5.1 Journal Validation
1. Must have at least 2 transactions
2. Must balance in base currency (debits = credits)
3. All transactions must be valid:
   - Must have a valid account ID
   - Must have a positive amount
   - Must have a valid transaction type (DEBIT/CREDIT)
4. Currency validation:
   - If `originalCurrency` is provided, it must be different from journal's base currency
   - If `originalCurrency` is provided, `exchangeRateId` must be provided and valid
   - Exchange rate must convert from `originalCurrency` to journal's base currency
5. All referenced accounts must exist and be active
6. All referenced exchange rates must exist and be valid

### 5.2 Transaction Validation
1. Must have a valid account ID
2. Original amount must be positive
3. Currency validation:
   - If `originalCurrency` is nil, the transaction is in the journal's base currency
   - If `originalCurrency` is provided:
     - Must be different from journal's base currency
     - Must include a valid `exchangeRateId`
     - Exchange rate must be from `originalCurrency` to journal's `currencyCode`
4. Transaction type must be DEBIT or CREDIT
5. Transaction date must not be in the future
6. If exchange rate is provided, it must be valid and active for the transaction date

## 6. Error Handling

### 6.1 Error Responses

#### Invalid Currency (400)
```json
{
  "code": "INVALID_CURRENCY",
  "message": "Currency 'XYZ' is not supported"
}
```

#### Exchange Rate Not Found (404)
```json
{
  "code": "EXCHANGE_RATE_NOT_FOUND",
  "message": "No exchange rate found from EUR to USD"
}
```

#### Unbalanced Journal (400)
```json
{
  "code": "UNBALANCED_JOURNAL",
  "message": "Journal debits (1000.00) do not equal credits (950.00)"
}
```

## 7. Testing Strategy

### 7.1 Unit Tests
- Test currency conversion calculations
- Test validation rules
- Test transaction creation with/without currency conversion

### 7.2 Integration Tests
- Test journal creation with multi-currency transactions
- Test exchange rate validation
- Test balance validation in base currency

### 7.3 Performance Tests
- Test journal creation with large numbers of transactions
- Test concurrent journal creation
- Test with high volume of currency conversions

## 8. Rollout Plan

### 8.1 Phase 1: Foundation
1. **Database Migrations**
   - Add new columns to transactions table
   - Add necessary indexes for performance
   - Update constraints

2. **Domain Model Updates**
   - Add new fields to Transaction model
   - Update validation logic
   - Add helper methods for currency conversion

3. **Repository Layer**
   - Implement new repository methods
   - Update transaction handling
   - Add exchange rate lookup functionality

4. **Service Layer**
   - Update journal service for multi-currency support
   - Implement exchange rate service updates
   - Add validation logic

### 8.2 Phase 2: Testing
- Unit and integration testing
- Performance testing
- User acceptance testing

### 8.3 Phase 3: Deployment
- Deploy to staging for final validation
- Deploy to production with feature flag
- Monitor for issues
- Enable for all users once stable
