package todo

import "time"

// NewSchema constructs a domain Schema from raw fields.
func NewSchema(id uint64, title string, done bool, priority int, createdAt, updatedAt time.Time, deletedAt *time.Time) *Schema {
	return &Schema{
		ID:        id,
		Title:     title,
		Done:      done,
		Priority:  priority,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}
}
