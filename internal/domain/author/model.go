package author

import "time"

type Schema struct {
	ID        uint64
	FirstName string
	MiddleName string
	LastName string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
