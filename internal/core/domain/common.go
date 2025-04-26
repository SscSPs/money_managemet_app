package domain

import "time"

// AuditFields holds standard audit information for domain entities.
// Note: UserID type might be string (UUID) or int depending on final design.
type AuditFields struct {
	CreatedAt     time.Time `json:"createdAt"`
	CreatedBy     string    `json:"createdBy"` // UserID Reference
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	LastUpdatedBy string    `json:"lastUpdatedBy"` // UserID Reference
}
