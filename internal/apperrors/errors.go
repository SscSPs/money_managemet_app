package apperrors

import "errors"

// ErrNotFound indicates that a requested resource could not be found.
var ErrNotFound = errors.New("resource not found")

// ErrValidation indicates that input data failed validation checks.
var ErrValidation = errors.New("validation error")

// ErrDuplicate indicates that an attempt was made to create a resource that already exists.
var ErrDuplicate = errors.New("resource already exists")

// ErrForbidden indicates that the user is authenticated but does not have permission to perform the action.
var ErrForbidden = errors.New("forbidden")

// ErrInternal indicates an unexpected server error.
var ErrInternal = errors.New("internal server error")

// TODO: Add other specific error types as needed (e.g., ErrUnauthorized)
