---
trigger: model_decision
description: when working on creating/updating/handling domain models
---


## 📦 Domain Models

```go
package domain

type User struct {
    UserID         string           `json:"userID"`
    Username       string           `json:"username"`
    Email          string           `json:"email"`
    PasswordHash   *string          `json:"-"` // Pointer for optionality
    Name           string           `json:"name"`
    AuthProvider   AuthProviderType `json:"auth_provider,omitempty"`
    ProviderUserID string           `json:"-"`
    IsVerified     bool             `json:"is_verified"`
    ProfilePicURL  string           `json:"profile_pic_url,omitempty"`
    AuditFields
    DeletedAt *time.Time `json:"deletedAt,omitempty"`
    
    RefreshTokenHash       string     `json:"-"`
    RefreshTokenExpiryTime *time.Time `json:"-"`
}

type AuditFields struct {
    CreatedAt     time.Time `json:"createdAt"`
    CreatedBy     string    `json:"createdBy"`
    LastUpdatedAt time.Time `json:"lastUpdatedAt"`
    LastUpdatedBy string    `json:"lastUpdatedBy"`
}

type AuthProviderType string

const (
    ProviderLocal  AuthProviderType = "local"
    ProviderGoogle AuthProviderType = "google"
)
```

**Domain Rules:**
- ✅ Embed `AuditFields`
- ✅ Use pointers for optional fields
- ✅ Use `json:"-"` for sensitive data
- ✅ Use typed constants for enums
- ✅ Include `DeletedAt *time.Time` for soft deletes
- ❌ Never use `sql.Null*` types
- ❌ Never import database packages
