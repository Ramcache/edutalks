package models

import "time"

type Document struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Filename   string    `json:"filename"`
	Filepath   string    `json:"filepath"`
	UploadedAt time.Time `json:"uploaded_at"`
}
