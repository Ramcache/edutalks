package models

import "time"

type News struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	ImageURL  string    `json:"image_url"`
	Color     string    `json:"color"`
	Sticker   string    `json:"sticker"`
	CreatedAt time.Time `json:"created_at"`
}
