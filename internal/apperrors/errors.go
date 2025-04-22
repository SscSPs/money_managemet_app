package apperrors

import "errors"

// ErrNotFound indicates that a requested resource could not be found.
var ErrNotFound = errors.New("resource not found")

// ErrValidation indicates that input data failed validation checks.
var ErrValidation = errors.New("validation error")
