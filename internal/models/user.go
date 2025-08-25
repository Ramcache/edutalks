package models

import "time"

type User struct {
	ID                    int        `json:"id"`
	Username              string     `json:"username"`
	FullName              string     `json:"full_name"`
	Phone                 string     `json:"phone"`
	Email                 string     `json:"email"`
	Address               string     `json:"address"`
	PasswordHash          string     `json:"-"`
	Role                  string     `json:"role"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	HasSubscription       bool       `json:"has_subscription"`
	EmailSubscription     bool       `json:"email_subscription"`
	EmailVerified         bool       `json:"email_verified"`
}

type UpdateUserRequest struct {
	FullName *string `json:"full_name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Address  *string `json:"address,omitempty"`
	Role     *string `json:"role,omitempty"`
}

type UserProfileResponse struct {
	ID                    int        `json:"id"`
	Username              string     `json:"username"`
	FullName              string     `json:"full_name"`
	Phone                 string     `json:"phone"`
	Email                 string     `json:"email"`
	Address               string     `json:"address"`
	Role                  string     `json:"role"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	HasSubscription       bool       `json:"has_subscription"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	EmailSubscription     bool       `json:"email_subscription"`
	EmailVerified         bool       `json:"email_verified"`
}
