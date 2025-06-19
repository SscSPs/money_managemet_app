package models

import "time"

// APIToken represents an API token for user authentication
type APIToken struct {
	ID         string     `json:"id" db:"id"`
	UserID     string     `json:"userID" db:"user_id"`
	Name       string     `json:"name" db:"name"`
	TokenHash  string     `json:"-" db:"token_hash"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty" db:"last_used_at"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty" db:"expires_at"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`
}

// TableName specifies the table name for GORM
func (APIToken) TableName() string {
	return "api_tokens"
}

// IsExpired checks if the token has expired
func (t *APIToken) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return t.ExpiresAt.Before(time.Now())
}

// UpdateLastUsed updates the LastUsedAt timestamp to current time
func (t *APIToken) UpdateLastUsed() {
	now := time.Now()
	t.LastUsedAt = &now
}
