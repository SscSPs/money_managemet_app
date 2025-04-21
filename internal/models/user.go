package models

// User represents a user of the application.
// Note: UserID type might be string (UUID) or int depending on final design.
type User struct {
	UserID string `json:"userID"` // Primary Key (e.g., UUID)
	Name   string `json:"name"`
	AuditFields
}
