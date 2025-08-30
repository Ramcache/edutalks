package models

import "time"

type Article struct {
	ID          int64      `db:"id"           json:"id"`
	AuthorID    *int64     `db:"author_id"    json:"authorId,omitempty"`
	Title       string     `db:"title"        json:"title"`
	Summary     *string    `db:"summary"      json:"summary,omitempty"`
	BodyHTML    string     `db:"body_html"    json:"bodyHtml"`
	Tags        []string   `db:"-"            json:"tags"`
	IsPublished bool       `db:"is_published" json:"isPublished"`
	PublishedAt *time.Time `db:"published_at" json:"publishedAt,omitempty"`
	CreatedAt   time.Time  `db:"created_at"   json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at"   json:"updatedAt"`
}

// swagger:model CreateArticleRequest
type CreateArticleRequest struct {
	Title       string   `json:"title"    example:"Как писать middleware в Go"`
	Summary     string   `json:"summary"  example:"Короткое описание для превью"`
	BodyHTML    string   `json:"bodyHtml" example:"<p>Контент</p>"`
	Tags        []string `json:"tags"     example:"go,backend,markdown"`
	Publish     bool     `json:"publish"`
	IsPublished *bool    `json:"isPublished,omitempty"`
}
