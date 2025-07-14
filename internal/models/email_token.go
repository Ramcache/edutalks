package models

import "time"

type EmailVerificationToken struct {
	UserID    int
	Token     string
	ExpiresAt time.Time
	Confirmed bool
}
