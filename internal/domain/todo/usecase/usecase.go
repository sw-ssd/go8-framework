package usecase

import (
	"context"

	"codeberg.org/gmhafiz/go8/internal/domain/todo"
	"codeberg.org/gmhafiz/go8/internal/domain/todo/repository"
)

// Todo is the business logic boundary for the todo resource.
type Todo interface {
	Create(ctx context.Context, r *todo.CreateRequest) (*todo.Schema, error)
	List(ctx context.Context, f *todo.Filter) ([]*todo.Schema, int, error)
	Read(ctx context.Context, id uint64) (*todo.Schema, error)
	Update(ctx context.Context, r *todo.UpdateRequest) (*todo.Schema, error)
	Delete(ctx context.Context, id uint64) error
}

// TodoUseCase implements Todo using an ent-backed repository.
type TodoUseCase struct {
	repo repository.Todo
}

func New(repo repository.Todo) *TodoUseCase {
	return &TodoUseCase{repo: repo}
}

func (u *TodoUseCase) Create(ctx context.Context, r *todo.CreateRequest) (*todo.Schema, error) {
	return u.repo.Create(ctx, r)
}

func (u *TodoUseCase) List(ctx context.Context, f *todo.Filter) ([]*todo.Schema, int, error) {
	return u.repo.List(ctx, f)
}

func (u *TodoUseCase) Read(ctx context.Context, id uint64) (*todo.Schema, error) {
	return u.repo.Read(ctx, id)
}

func (u *TodoUseCase) Update(ctx context.Context, r *todo.UpdateRequest) (*todo.Schema, error) {
	return u.repo.Update(ctx, r)
}

func (u *TodoUseCase) Delete(ctx context.Context, id uint64) error {
	return u.repo.Delete(ctx, id)
}
