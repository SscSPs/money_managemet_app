package apperrors

import (
	"fmt"
	"net/http"
)

// Original sentinel errors for direct comparison (e.g., errors.Is())
var (
	ErrNotFound            = NewNotFoundError("resource not found")
	ErrDuplicate           = NewAppError(http.StatusConflict, "resource already exists", nil) // Or unique constraint violation
	ErrForbidden           = NewForbiddenError("forbidden")                                   // User does not have permission
	ErrUnauthorized        = NewUnauthorizedError("unauthorized")                             // User authentication failed or missing
	ErrValidation          = NewValidationFailedError("validation failed")                    // Input data validation failed
	ErrInternal            = NewInternalServerError("internal server error")                  // Generic unexpected error
	ErrConflict            = NewConflictError("operation conflict")                           // e.g., trying to modify a resource in an invalid state
	ErrBadRequest          = NewBadRequestError("bad request")                                // Malformed request or invalid parameters
	ErrRefreshTokenExpired = NewUnauthorizedError("refresh token has expired")                // Specific for expired refresh tokens
)

// AppError is a custom error type that includes an HTTP status code and a user-friendly message.
// It allows for structured error handling and consistent JSON error responses.
type AppError struct {
	Code    int    `json:"code"`    // HTTP status code (e.g., 400, 401, 500)
	Message string `json:"message"` // User-facing error message
	Err     error  `json:"-"`       // Internal underlying error, not exposed in JSON responses by default
}

// Error makes AppError satisfy the standard Go error interface.
// It returns a detailed string representation of the error, primarily for logging.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("AppError: code=%d, message='%s', underlying_error='%v'", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("AppError: code=%d, message='%s'", e.Code, e.Message)
}

// Unwrap provides compatibility with errors.Is and errors.As by returning the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// --- Constructor Functions for Common HTTP Errors ---

// NewAppError is a generic constructor for AppError.
// It's useful for creating AppErrors that don't fit predefined types or for wrapping existing errors.
func NewAppError(code int, message string, originalError error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     originalError,
	}
}

// NewBadRequestError creates an AppError representing a 400 Bad Request.
func NewBadRequestError(message string) *AppError {
	return NewAppError(http.StatusBadRequest, message, nil)
}

// NewUnauthorizedError creates an AppError representing a 401 Unauthorized.
func NewUnauthorizedError(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, message, nil)
}

// NewForbiddenError creates an AppError representing a 403 Forbidden.
func NewForbiddenError(message string) *AppError {
	return NewAppError(http.StatusForbidden, message, nil)
}

// NewNotFoundError creates an AppError representing a 404 Not Found.
func NewNotFoundError(message string) *AppError {
	return NewAppError(http.StatusNotFound, message, nil)
}

// NewConflictError creates an AppError representing a 409 Conflict.
func NewConflictError(message string) *AppError {
	return NewAppError(http.StatusConflict, message, nil)
}

// NewInternalServerError creates an AppError representing a 500 Internal Server Error.
// It's advisable to keep the user-facing message generic for internal errors.
func NewInternalServerError(message string) *AppError {
	// For internal errors, you might not want to expose the originalError's message directly to the client.
	// So, 'originalError' might be logged but not necessarily part of the 'message' arg here.
	return NewAppError(http.StatusInternalServerError, message, nil)
}

// NewGatewayTimeoutError creates an AppError representing a 504 Gateway Timeout.
func NewGatewayTimeoutError(message string) *AppError {
	return NewAppError(http.StatusGatewayTimeout, message, nil)
}

// NewValidationError creates an AppError for validation issues with a custom message.
// It's an alias for NewValidationFailedError for backward compatibility.
func NewValidationError(message string) *AppError {
	return NewValidationFailedError(message)
}

// NewValidationFailedError creates an AppError for validation issues, typically a 422 Unprocessable Entity or 400 Bad Request.
func NewValidationFailedError(message string) *AppError {
	// HTTP 422 is often used for validation errors, but 400 is also common.
	return NewAppError(http.StatusUnprocessableEntity, message, nil)
}

// NewNotImplementedError creates an AppError representing a 501 Not Implemented.
func NewNotImplementedError(message string) *AppError {
	return NewAppError(http.StatusNotImplemented, message, nil)
}
