package book

import "time"

type CreateRequest struct {
	Title         string    `json:"title" validate:"required"`
	PublishedDate time.Time `json:"published_date"`
	ImageURL      string    `json:"image_url"`
	Description   string    `json:"description"`
	AuthorID      int64     `json:"author_id"`
}

type UpdateRequest struct {
	ID            uint64    `json:"id"`
	Title         string    `json:"title"`
	PublishedDate time.Time `json:"published_date"`
	ImageURL      string    `json:"image_url"`
	Description   string    `json:"description"`
	AuthorID      int64     `json:"author_id"`
}
