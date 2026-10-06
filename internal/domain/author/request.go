package author

type CreateRequest struct {
	FirstName  string `json:"first_name" validate:"required"`
	MiddleName string `json:"middle_name"`
	LastName   string `json:"last_name" validate:"required"`
}

type UpdateRequest struct {
	ID         uint64 `json:"id"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	LastName   string `json:"last_name"`
}
