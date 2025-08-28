package models

import "time"

type Tab struct {
	ID        int       `json:"id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Position  int       `json:"position"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Section struct {
	ID          int       `json:"id"`
	TabID       int       `json:"tab_id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Position    int       `json:"position"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SectionWithCount struct {
	Section   Section `json:"section"`
	DocsCount int     `json:"docs_count"`
}

type TabTree struct {
	Tab      Tab                `json:"tab"`
	Sections []SectionWithCount `json:"sections"`
}
