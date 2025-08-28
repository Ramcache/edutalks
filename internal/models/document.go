package models

import "time"

type Document struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Filename    string    `json:"filename"`
	Filepath    string    `json:"filepath"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	Category    string    `json:"category"`
	SectionID   *int      `json:"section_id,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type DocumentPreviewResponse struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	UploadedAt  string `json:"uploaded_at"`
	Message     string `json:"message"`
}
