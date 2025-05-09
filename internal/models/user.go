package models

import (
	"database/sql"
	"time"
)

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

	// Refresh Token Fields
	RefreshTokenHash       sql.NullString `db:"refresh_token_hash"`        // Store hash of the refresh token
	RefreshTokenExpiryTime sql.NullTime   `db:"refresh_token_expiry_time"` // Expiry of the stored refresh token
}
