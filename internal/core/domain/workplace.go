package domain

import "time"

// Workplace represents an isolated environment containing accounts, journals, etc.
type Workplace struct {
	WorkplaceID         string  `json:"workplaceID"`         // Primary Key (e.g., UUID)
	Name                string  `json:"name"`                // User-defined name for the workplace
	Description         string  `json:"description"`         // Optional description
	DefaultCurrencyCode *string `json:"defaultCurrencyCode"` // Default currency code for this workplace (e.g., "USD")
	IsActive            bool    `json:"isActive"`            // Indicates whether the workplace is active or disabled
	AuditFields                 // Embed common audit fields
}

// UserWorkplaceRole defines the possible roles a user can have within a workplace.
type UserWorkplaceRole string

const (
	RoleAdmin    UserWorkplaceRole = "ADMIN"
	RoleMember   UserWorkplaceRole = "MEMBER"
	RoleReadOnly UserWorkplaceRole = "READONLY" // Users with read-only access to workplace data
	RoleRemoved  UserWorkplaceRole = "REMOVED"  // For users who have been removed from the workplace
)

// UserWorkplace represents the membership of a User in a Workplace.
type UserWorkplace struct {
	UserID      string            `json:"userID"`      // FK -> users.user_id
	UserName    string            `json:"userName"`    // Name of the user
	WorkplaceID string            `json:"workplaceID"` // FK -> workplaces.workplace_id
	Role        UserWorkplaceRole `json:"role"`        // Role of the user in this specific workplace
	JoinedAt    time.Time         `json:"joinedAt"`    // Timestamp when the user joined the workplace
}
