# MMA Backend (Money Management App)

This is the backend service for the Money Management Application (MMA). It provides a RESTful API for managing financial data.

## Key Technologies

*   **Language:** Go
*   **Web Framework:** Gin (`github.com/gin-gonic/gin`)
*   **CORS Handling:** `github.com/gin-contrib/cors`
*   **Database:** PostgreSQL (using `github.com/jackc/pgx/v5`)
*   **Migrations:** `github.com/golang-migrate/migrate`
*   **Configuration:** Custom package (`internal/platform/config`)
*   **Logging:** Standard Go `log/slog`
*   **Authentication:** JWT (`github.com/golang-jwt/jwt/v5`)
*   **API Documentation:** Swagger (`github.com/swaggo/swag`, `github.com/swaggo/gin-swagger`)

## Project Structure

*   `cmd/mma_backend`: Main application entry point (`main.go`).
*   `internal/`: Contains core application logic, separated by concern:
    *   `apperrors/`: Custom application errors.
    *   `core/`: Core business logic:
        *   `domain/`: Core domain models.
        *   `ports/`: Interfaces (contracts) for services and repositories.
        *   `services/`: Business logic implementation.
    *   `dto/`: Data Transfer Objects for API request/response.
    *   `handlers/`: Gin HTTP request handlers.
    *   `middleware/`: Gin middleware (e.g., auth, logging, CORS).
    *   `models/`: Data structures mirroring database schema (used by repositories).
    *   `platform/`: Platform-specific concerns (e.g., `config`, `database` connection).
    *   `repositories/`: Implementations of repository interfaces (e.g., `database/pgsql/`).
*   `migrations/`: SQL database migration files.
*   `docs/`: Generated Swagger/OpenAPI documentation files.
*   `makefile`: Contains common development tasks (build, run, test, etc.).
*   `go.mod`, `go.sum`: Go module dependency files.

## Database Schema

The database schema is managed using SQL migrations located in the `migrations/` directory.
Key identifiers (like `user_id`, `workplace_id`, `account_id`, etc.) primarily use `VARCHAR(255)` as their data type.

The `users` table implements soft deletion using a nullable `deleted_at` timestamp column.

Refer to the migration files (`*.up.sql`) for the most up-to-date and detailed schema definition.

## Getting Started

### Prerequisites

*   Go (version specified in `go.mod`)
*   PostgreSQL database
*   Make (optional, for using the `makefile`)
*   `golang-migrate` CLI (optional, for manual migration management - install instructions: [https://github.com/golang-migrate/migrate/tree/master/cmd/migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate))
*   `swag` CLI (optional, for manual swagger generation - `go install github.com/swaggo/swag/cmd/swag@latest`)

### Configuration

Configuration is loaded via environment variables or a `.env` file (check `internal/platform/config/config.go` for details). A sample `.env.sample` file is provided - copy it to `.env` and fill in your details:

```bash
cp .env.sample .env
# Edit .env with your PGSQL_URL, JWT_SECRET etc.
```

### Running the Application

1.  **Set up Database:** Ensure your PostgreSQL server is running and the database specified in `PGSQL_URL` exists.
2.  **Run Migrations:** The application attempts to run migrations automatically *before* the server starts. Alternatively, you can run them manually using the `golang-migrate` CLI against your database and the `migrations/` directory.
3.  **Build & Run:**
    *   Using Make: `make run` (builds and runs)
    *   Manually: `go run cmd/mma_backend/main.go` (ensure required environment variables are set).

## API Documentation

API documentation is generated using Swagger/OpenAPI specifications from GoDoc comments.

*   **Regeneration:** Use `swag init` or `make swag` to update the files in the `docs/` directory after changing handler comments or DTOs.
*   **Access:** When running the server locally in a non-production environment (`IS_PRODUCTION=false`), documentation is available at [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) (adjust port if changed).

### Key Endpoints (Overview)

*(Note: This is not exhaustive, refer to Swagger UI for full details)*

*   `/api/v1/auth/register` [POST]
*   `/api/v1/auth/login` [POST]
*   `/api/v1/users` [GET, POST]
*   `/api/v1/users/{id}` [GET, PUT, DELETE]
*   `/api/v1/currencies` [GET, POST]
*   `/api/v1/currencies/{code}` [GET]
*   `/api/v1/exchange-rates` [POST]
*   `/api/v1/exchange-rates/{from}/{to}` [GET]
*   `/api/v1/workplaces` [GET, POST]
*   `/api/v1/workplaces/{workplace_id}/users` [POST]
*   `/api/v1/workplaces/{workplace_id}/accounts` [GET, POST] (Account CRUD is relative to workplace)
*   `/api/v1/workplaces/{workplace_id}/accounts/{id}` [GET, PUT, DELETE]
*   `/api/v1/workplaces/{workplace_id}/journals` [GET, POST] (Journal CRUD is relative to workplace)
*   `/api/v1/workplaces/{workplace_id}/journals/{id}` [GET, PUT, DELETE] (GET now includes transaction details)

## Running Tests

*   Using Make: `make test`
*   Manually: `go test ./...`
