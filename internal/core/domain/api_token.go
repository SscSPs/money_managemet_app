package domain

import "time"

// APIToken represents an API token for authenticating API requests
type APIToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"` // Never expose the hash in JSON responses
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"-"` // For soft deletes
}

// IsExpired checks if the token has expired
func (t *APIToken) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return t.ExpiresAt.Before(time.Now())
}

// UpdateLastUsed updates the LastUsedAt timestamp to the current time
func (t *APIToken) UpdateLastUsed() {
	now := time.Now()
	t.LastUsedAt = &now
}
