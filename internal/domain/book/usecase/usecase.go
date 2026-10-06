package usecase

import (
	"context"

	"codeberg.org/gmhafiz/go8/internal/domain/book"
	"codeberg.org/gmhafiz/go8/internal/domain/book/repository"
)

// Book is the business logic boundary for the book resource.
type Book interface {
	// go8cli:iface
	Create(ctx context.Context, r *book.CreateRequest) (*book.Schema, error)
	List(ctx context.Context, f *book.Filter) ([]*book.Schema, int, error)
	Read(ctx context.Context, id uint64) (*book.Schema, error)
	Update(ctx context.Context, r *book.UpdateRequest) (*book.Schema, error)
	Delete(ctx context.Context, id uint64) error
}

// BookUseCase implements Book using an ent-backed repository.
type BookUseCase struct {
	repo repository.Book
}

func New(repo repository.Book) *BookUseCase {
	return &BookUseCase{repo: repo}
}

// go8cli:impl
func (u *BookUseCase) Create(ctx context.Context, r *book.CreateRequest) (*book.Schema, error) {
	return u.repo.Create(ctx, r)
}
func (u *BookUseCase) List(ctx context.Context, f *book.Filter) ([]*book.Schema, int, error) {
	return u.repo.List(ctx, f)
}
func (u *BookUseCase) Read(ctx context.Context, id uint64) (*book.Schema, error) {
	return u.repo.Read(ctx, id)
}
func (u *BookUseCase) Update(ctx context.Context, r *book.UpdateRequest) (*book.Schema, error) {
	return u.repo.Update(ctx, r)
}
func (u *BookUseCase) Delete(ctx context.Context, id uint64) error {
	return u.repo.Delete(ctx, id)
}
