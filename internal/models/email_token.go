package models

import "time"

type EmailVerificationToken struct {
	UserID    int
	Token     string
	ExpiresAt time.Time
	Confirmed bool
	CreatedAt time.Time
}
type EmailJob struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
}
