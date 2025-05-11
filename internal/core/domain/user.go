package domain

import "time"

// AuthProviderType represents the type of authentication provider.
type AuthProviderType string

// Constants for AuthProviderType
const (
	ProviderLocal  AuthProviderType = "local"
	ProviderGoogle AuthProviderType = "google"
	// Add other providers like Facebook, GitHub etc. as needed
)

// User represents a user of the application in the domain.
type User struct {
	UserID         string           `json:"userID"`
	Username       string           `json:"username" db:"username"`         // Consider if this should also be optional/nullable
	Email          string           `json:"email" db:"email"`               // Added Email field
	PasswordHash   *string          `json:"-" db:"password_hash,omitempty"` // Made pointer for optionality
	Name           string           `json:"name" db:"name"`                 // Changed from pointer
	AuthProvider   AuthProviderType `json:"auth_provider,omitempty" db:"auth_provider,omitempty"`
	ProviderUserID string           `json:"-" db:"provider_user_id,omitempty"` // User's ID from the external provider, non-pointer
	IsVerified     bool             `json:"is_verified" db:"is_verified"`
	ProfilePicURL  string           `json:"profile_pic_url,omitempty" db:"profile_pic_url,omitempty"`
	AuditFields
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`

	// Refresh Token Fields
	RefreshTokenHash       string     `json:"-" db:"refresh_token_hash"`        // Store hash of the refresh token
	RefreshTokenExpiryTime *time.Time `json:"-" db:"refresh_token_expiry_time"` // Expiry of the stored refresh token
}

func (u *User) GetUserID() string   { return u.UserID }
func (u *User) GetUsername() string { return u.Username }
func (u *User) GetName() string     { return u.Name }

// UpdateUserProviderDetails is a DTO for updating a user's provider-specific information.
// Fields are pointers to allow for partial updates (only updating fields that are non-nil).
type UpdateUserProviderDetails struct {
	AuthProvider   AuthProviderType `json:"auth_provider,omitempty"`
	ProviderUserID string           `json:"provider_user_id,omitempty"`
	Name           *string          `json:"name,omitempty"`  // Pointer to allow selective update
	Email          *string          `json:"email,omitempty"` // Added Email field
	IsVerified     *bool            `json:"is_verified,omitempty"`
	ProfilePicURL  *string          `json:"profile_pic_url,omitempty"`
}

// GoogleUserInfo represents the user information obtained from Google.
// The field names match the typical JSON response from Google's userinfo endpoint.
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}
