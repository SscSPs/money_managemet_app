---
trigger: model_decision
description: when working on creating/updating/handling DTOs
---


## 📄 DTOs

**Request DTOs:**
```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name" binding:"required,min=1,max=100"`
}

type UpdateUserRequest struct {
    Name *string `json:"name"` // Pointer to distinguish omitted vs empty
}

type ListUsersParams struct {
    Limit  int `form:"limit,default=20"`
    Offset int `form:"offset,default=0"`
}
```

**Response DTOs:**
```go
type UserResponse struct {
    UserID        string    `json:"userId"`
    Username      string    `json:"username"`
    Email         string    `json:"email,omitempty"`
    Name          string    `json:"name"`
    IsVerified    bool      `json:"isVerified"`
    ProfilePicURL string    `json:"profilePicUrl,omitempty"`
    CreatedAt     time.Time `json:"createdAt"`
    UpdatedAt     time.Time `json:"updatedAt"`
}

func ToUserResponse(user *domain.User) UserResponse {
    return UserResponse{
        UserID:        user.UserID,
        Username:      user.Username,
        Email:         user.Email,
        Name:          user.Name,
        IsVerified:    user.IsVerified,
        ProfilePicURL: user.ProfilePicURL,
        CreatedAt:     user.CreatedAt,
        UpdatedAt:     user.LastUpdatedAt,
    }
}
```

**DTO Rules:**
- ✅ Use `binding` tags for validation
- ✅ Use `form` tags for query params
- ✅ Use pointers in update DTOs for partial updates
- ✅ Include converter functions (ToXxxResponse)
- ✅ Use camelCase for JSON field names
