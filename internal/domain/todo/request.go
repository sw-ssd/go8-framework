package todo

type CreateRequest struct {
	Title    string `json:"title" validate:"required"`
	Priority int    `json:"priority"`
}

type UpdateRequest struct {
	ID       uint64 `json:"id"`
	Title    string `json:"title"`
	Done     bool   `json:"done"`
	Priority int    `json:"priority"`
}
