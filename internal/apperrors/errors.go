package apperrors

import "errors"

var (
	ErrNotFound       = errors.New("resource not found")
	ErrDuplicate      = errors.New("resource already exists") // Or unique constraint violation
	ErrForbidden      = errors.New("forbidden")               // User does not have permission
	ErrUnauthorized   = errors.New("unauthorized")            // User authentication failed or missing
	ErrValidation     = errors.New("validation failed")       // Input data validation failed
	ErrInternal       = errors.New("internal server error")   // Generic unexpected error
	ErrConflict       = errors.New("operation conflict")      // e.g., trying to modify a resource in an invalid state
	ErrBadRequest     = errors.New("bad request")             // Malformed request or invalid parameters
	ErrNotImplemented = errors.New("not implemented")         // Feature not yet implemented
)
