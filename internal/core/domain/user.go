package domain

import "time"

// User represents a user of the application in the domain.
type User struct {
	UserID string `json:"userID"` // Primary Key (e.g., UUID)
	Name   string `json:"name"`
	AuditFields
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"` // Used for soft delete
}
