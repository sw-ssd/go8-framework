package book

import "time"

type Schema struct {
	ID        uint64
	Title string
	PublishedDate time.Time
	ImageURL string
	Description string
	AuthorID int64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
