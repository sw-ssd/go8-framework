package usecase

import (
	"context"

	"codeberg.org/gmhafiz/go8/internal/domain/author"
	"codeberg.org/gmhafiz/go8/internal/domain/author/repository"
)

// Author is the business logic boundary for the author resource.
type Author interface {
	// go8cli:iface
	Create(ctx context.Context, r *author.CreateRequest) (*author.Schema, error)
	List(ctx context.Context, f *author.Filter) ([]*author.Schema, int, error)
	Read(ctx context.Context, id uint64) (*author.Schema, error)
	Update(ctx context.Context, r *author.UpdateRequest) (*author.Schema, error)
	Delete(ctx context.Context, id uint64) error
}

// AuthorUseCase implements Author using an ent-backed repository.
type AuthorUseCase struct {
	repo repository.Author
}

func New(repo repository.Author) *AuthorUseCase {
	return &AuthorUseCase{repo: repo}
}

// go8cli:impl
func (u *AuthorUseCase) Create(ctx context.Context, r *author.CreateRequest) (*author.Schema, error) {
	return u.repo.Create(ctx, r)
}
func (u *AuthorUseCase) List(ctx context.Context, f *author.Filter) ([]*author.Schema, int, error) {
	return u.repo.List(ctx, f)
}
func (u *AuthorUseCase) Read(ctx context.Context, id uint64) (*author.Schema, error) {
	return u.repo.Read(ctx, id)
}
func (u *AuthorUseCase) Update(ctx context.Context, r *author.UpdateRequest) (*author.Schema, error) {
	return u.repo.Update(ctx, r)
}
func (u *AuthorUseCase) Delete(ctx context.Context, id uint64) error {
	return u.repo.Delete(ctx, id)
}
