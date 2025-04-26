# Testing Plan

This document outlines the strategy and plan for implementing automated tests in the MMA Backend codebase.

## I. Testing Philosophy & Levels

We will adopt a multi-layered testing approach:

1.  **Unit Tests:** Test individual functions or components (like service methods, specific utility functions) in isolation. Dependencies will be mocked. These are fast and pinpoint specific logic errors.
2.  **Integration Tests:** Test the interaction between components or layers. Crucially, this will involve testing repositories against a real (but temporary) database and testing services with their actual repository implementations. This verifies contracts between layers and database interactions. Handler integration tests will involve testing API endpoints with real services/repositories connected to a test DB.
3.  **End-to-End (E2E) Tests:** Test the entire application flow by making HTTP requests to a running instance of the server (configured for testing) and verifying the responses and side effects (like database state).

## II. Tools & Setup

1.  **Go Standard Library `testing`:** The foundation for all Go tests (`go test`).
2.  **Assertion Library:** `testify` (`github.com/stretchr/testify`) provides better assertion functions (`assert` and `require`) than the standard library.
3.  **Mocking Library:** `testify/mock` (`github.com/stretchr/testify/mock`) for creating mocks of interfaces (like repository interfaces for service unit tests).
4.  **HTTP Testing:** `net/http/httptest` for testing Gin handlers without needing a live server.
5.  **Database Integration Testing:** `testcontainers-go` (`github.com/testcontainers/testcontainers-go`) to programmatically spin up ephemeral PostgreSQL Docker containers for integration tests. This ensures tests run against a real PostgreSQL instance, matching production.
6.  **Docker:** Required for `testcontainers-go`.

## III. Test Structure & Location

1.  **Unit Tests:** Place test files (`*_test.go`) alongside the code they are testing (e.g., `internal/core/services/account_service_test.go`).
2.  **Integration Tests:**
    *   **Repository Tests:** Place `*_test.go` files within the repository implementation package (e.g., `internal/repositories/database/pgsql/account_repository_test.go`).
    *   **Service/Handler Tests:** Use Go build tags (e.g., `//go:build integration`) in `*_test.go` files alongside the code to separate them from unit tests during execution (`go test -tags=integration`).
3.  **E2E Tests:** Place these in a separate top-level directory, e.g., `tests/e2e/`.

## IV. Detailed Plan & Prioritization

We'll implement tests layer by layer, likely starting from the core outwards or focusing on critical paths first.

**Phase 1: Unit Tests (Core Logic)** - [x] **Completed**

*   **Target:** `internal/core/services/*`
*   **Strategy:**
    *   For each service (e.g., `AccountService`), create `*_test.go` (e.g., `account_service_test.go`).
    *   Use `testify/mock` to create mocks for repository interfaces (`portsrepo.AccountRepository`, etc.).
    *   Write test cases for each public method in the service.
    *   Focus on testing business logic, validation rules, error handling, and correct interaction with repository mocks (e.g., asserting expected method calls on the mock).
    *   Use `testify/assert` or `testify/require` for assertions.
*   **Example:** Test `AccountService.CreateAccount` logic, mocking `AccountRepository.SaveAccount`.
*   **Status:** Completed for `AccountService`, `CurrencyService`, `ExchangeRateService`, `UserService`, `JournalService`. Tests are passing. (Note: `JournalService.CalculateAccountBalance` tests are placeholders pending full implementation visibility).

**Phase 2: Repository Integration Tests (Database Interaction)**

*   **Target:** `internal/repositories/database/pgsql/*`
*   **Strategy:**
    *   For each repository (e.g., `PgxAccountRepository`), create `*_test.go`. Mark with `//go:build integration`.
    *   Use `testcontainers-go` to set up a PostgreSQL container before tests run (e.g., in `TestMain` or using `testify/suite`).
    *   Run database migrations against the test container.
    *   Write test cases for each repository method (Save, FindByID, List, Deactivate, etc.).
    *   Interact with the *real* test database connection.
    *   Assert that data is correctly inserted, retrieved, updated, or marked inactive.
    *   Ensure proper error handling (e.g., returning `apperrors.ErrNotFound`).
    *   Clean up database state between tests (e.g., TRUNCATE tables).
*   **Example:** Test `PgxAccountRepository.SaveAccount` and `PgxAccountRepository.FindAccountByID` by inserting data and then fetching it.

**Phase 3: Handler Integration Tests (API Layer)**

*   **Target:** `internal/handlers/*` (specifically the handler methods and route registration).
*   **Strategy:**
    *   Create `*_test.go` files (e.g., `handler_account_test.go`). Mark with `//go:build integration`.
    *   Set up a test instance of the Gin engine.
    *   Set up *real* dependencies: PostgreSQL test container, real repository implementations, real service implementations.
    *   Use `net/http/httptest` to create `Request` objects simulating API calls and `ResponseRecorder` to capture responses.
    *   Call the Gin engine's `ServeHTTP` method or directly call handler functions after setting up the Gin context.
    *   Assert HTTP status codes, response bodies (parsing JSON), and potentially check the state of the test database after the request.
    *   Test authentication middleware by adding/omitting JWT tokens in requests.
*   **Example:** Test `POST /accounts` by creating a request with valid JSON, serving it to the test engine, asserting a `201 Created` status, checking the response body, and verifying the account exists in the test DB.

**Phase 4: E2E Tests (Full System)**

*   **Target:** Overall application behavior via API endpoints.
*   **Strategy:**
    *   Create tests in `tests/e2e/`. Mark with `//go:build e2e`.
    *   These tests will likely start the *entire* application binary (configured with a test database connection string pointing to a test container).
    *   Use Go's `net/http` client to make requests to the running test server's address.
    *   Test sequences of operations (e.g., create user, login, create account, get balance).
    *   Assert responses and potentially query the test database directly to verify end state.
*   **Example:** Test creating an account (`POST /accounts`), then fetching its balance (`GET /accounts/{id}/balance`), asserting the balance is zero initially.

## V. Implementation Steps

1.  **Add Dependencies:** Run `go get github.com/stretchr/testify github.com/testcontainers/testcontainers-go`. Ensure Docker is installed.
2.  **Start with Unit Tests:** Begin writing unit tests for services, mocking repositories.
3.  **Implement Repository Integration Tests:** Set up test containers and write tests for repository methods.
4.  **Implement Handler Integration Tests:** Build upon the previous phase.
5.  **(Optional but Recommended) Implement E2E Tests:** Add broader scenario tests.
6.  **Integrate into CI:** Add `go test ./...` (for unit tests) and `go test -tags=integration ./...` and `go test -tags=e2e ./...` (or similar) to your CI pipeline. 