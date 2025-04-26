package apperrors

import "errors"

// ErrNotFound indicates that a requested resource could not be found.
var ErrNotFound = errors.New("resource not found")

// ErrValidation indicates that input data failed validation checks.
var ErrValidation = errors.New("validation error")

// ErrDuplicate indicates that an attempt was made to create a resource that already exists.
var ErrDuplicate = errors.New("resource already exists")

// TODO: Add other specific error types as needed (e.g., ErrUnauthorized, ErrForbidden)
