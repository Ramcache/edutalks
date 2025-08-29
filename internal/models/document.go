package models

import "time"

type Document struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	Filename    string    `json:"filename"`
	Filepath    string    `json:"-"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	Category    string    `json:"category"`
	SectionID   *int      `json:"section_id"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type DocumentPreviewResponse struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	SectionID   *int   `json:"section_id,omitempty"`
	UploadedAt  string `json:"uploaded_at"`
	Message     string `json:"message"`
}
