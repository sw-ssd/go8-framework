package todo

import "time"

type Schema struct {
	ID        uint64
	Title     string
	Done      bool
	Priority  int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
