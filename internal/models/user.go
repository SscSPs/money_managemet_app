package models

import "time"

// User represents a user of the application.
// Now includes username and password hash for authentication.
// Note: UserID type might be string (UUID) or int depending on final design.
type User struct {
	UserID       string `json:"userID"`
	Username     string `json:"username" db:"username"`
	PasswordHash string `json:"-" db:"password_hash"`
	Name         string `json:"name"`
	AuditFields
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}
