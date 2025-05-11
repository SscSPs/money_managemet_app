package models

import (
	"database/sql"
	"time"
)

// User represents a user of the application.
// Now includes username and password hash for authentication.
// Note: UserID type might be string (UUID) or int depending on final design.
type User struct {
	UserID         string         `json:"userID"`
	Username       string         `json:"username" db:"username"`
	Email          sql.NullString `json:"email" db:"email"` // Added Email field
	PasswordHash   sql.NullString `json:"-" db:"password_hash"`
	Name           string         `json:"name"`
	AuthProvider   sql.NullString `json:"authProvider,omitempty" db:"auth_provider"`
	ProviderUserID sql.NullString `json:"-" db:"provider_user_id"`
	AuditFields
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`

	// Refresh Token Fields
	RefreshTokenHash       sql.NullString `db:"refresh_token_hash"`        // Store hash of the refresh token
	RefreshTokenExpiryTime sql.NullTime   `db:"refresh_token_expiry_time"` // Expiry of the stored refresh token
}
