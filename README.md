# MMA Backend (Money Management App)

This is the backend service for the Money Management Application (MMA). It provides a RESTful API for managing financial data.

## Key Technologies

*   **Language:** Go
*   **Web Framework:** Gin (`github.com/gin-gonic/gin`)
*   **Database:** PostgreSQL (using `github.com/jackc/pgx/v5`)
*   **Migrations:** `github.com/golang-migrate/migrate`
*   **Configuration:** Custom package (`pkg/config`)
*   **Logging:** Standard Go `log/slog`
*   **Authentication:** JWT (currently dummy impl)

## Project Structure

*   `cmd/mma_backend`: Main application entry point (`main.go`).
*   `internal/`: Contains core application logic, separated by concern:
    *   `adapters/`: Implementations of ports (e.g., database repositories).
    *   `apperrors/`: Custom application errors.
    *   `core/`: Core business logic:
        *   `ports/`: Interfaces (contracts) for services and repositories.
        *   `services/`: Business logic implementation.
    *   `dto/`: Data Transfer Objects for API communication.
    *   `handlers/`: Gin HTTP request handlers.
    *   `middleware/`: Gin middleware (e.g., auth, logging).
    *   `models/`: Data structures (likely database models).
*   `pkg/`: Shared libraries/utilities (e.g., `config`, `database`).
*   `migrations/`: SQL database migration files.
*   `docs/`: Potentially API documentation source files (e.g., for Swagger).
*   `makefile`: Contains common development tasks (build, run, test, etc.).
*   `go.mod`, `go.sum`: Go module dependency files.

## Getting Started

### Prerequisites

*   Go (version specified in `go.mod`)
*   PostgreSQL database
*   Make (optional, for using the `makefile`)

### Configuration

Configuration is loaded via environment variables or a `.env` file (check `pkg/config/config.go` for details). Sample `.env.sample` file is provided:

### Running the Application

1.  **Set up Database:** Ensure your PostgreSQL server is running and the database specified in `DATABASE_URL` exists.
2.  **Run Migrations:** The application attempts to run migrations automatically on startup. You might need to create the initial database if it doesn't exist.
3.  **Build & Run:**
    *   Using Make (if available): Check the `makefile` for targets like `make run` or `make build`.
    *   Manually: `go run cmd/mma_backend/main.go` (ensure required environment variables are set).

## API Documentation

API documentation is generated using Swagger. When running in a non-production environment, it's typically available at `/swagger/index.html`. 
