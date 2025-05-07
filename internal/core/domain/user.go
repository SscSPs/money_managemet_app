package domain

import "time"

// User represents a user of the application in the domain.
type User struct {
	UserID       string     `json:"userID"`
	Username     string     `json:"username" db:"username"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Name         string     `json:"name"`
	AuditFields
	DeletedAt    *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

func (u *User) GetUserID() string    { return u.UserID }
func (u *User) GetUsername() string  { return u.Username }
func (u *User) GetName() string      { return u.Name }
