package repository

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"codeberg.org/gmhafiz/go8/ent/gen"
	entBook "codeberg.org/gmhafiz/go8/ent/gen/book"
	"codeberg.org/gmhafiz/go8/internal/domain/book"
)

type repository struct {
	ent *gen.Client
}

// Book is the persistence boundary for the book resource (ent-backed).
type Book interface {
	// go8cli:iface
	Create(ctx context.Context, request *book.CreateRequest) (*book.Schema, error)
	List(ctx context.Context, f *book.Filter) ([]*book.Schema, int, error)
	Read(ctx context.Context, id uint64) (*book.Schema, error)
	Update(ctx context.Context, request *book.UpdateRequest) (*book.Schema, error)
	Delete(ctx context.Context, id uint64) error
}

func New(ent *gen.Client) *repository {
	return &repository{ent: ent}
}

// go8cli:impl
func (r *repository) Create(ctx context.Context, request *book.CreateRequest) (*book.Schema, error) {
	create := r.ent.Book.Create()
	create = create.SetTitle(request.Title)
	create = create.SetPublishedDate(request.PublishedDate)
	create = create.SetImageURL(request.ImageURL)
	create = create.SetDescription(request.Description)
	create = create.SetAuthorID(request.AuthorID)
	create = create.SetCreatedAt(time.Now())
	created, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSchema(created), nil
}

func (r *repository) List(ctx context.Context, f *book.Filter) ([]*book.Schema, int, error) {
	query := r.ent.Book.Query()
	if f.Title != "" {
		query = query.Where(entBook.TitleContains(f.Title))
	}
	if f.ImageURL != "" {
		query = query.Where(entBook.ImageURLContains(f.ImageURL))
	}
	if f.Description != "" {
		query = query.Where(entBook.DescriptionContains(f.Description))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if !f.Base.DisablePaging {
		query = query.Limit(f.Base.Limit).Offset(f.Base.Offset)
	}
	for field, order := range f.Base.Sort {
		if order == "DESC" {
			switch field {
			case "title":
				query = query.Order(entBook.ByTitle(sql.OrderDesc()))
			case "image_url":
				query = query.Order(entBook.ByImageURL(sql.OrderDesc()))
			case "description":
				query = query.Order(entBook.ByDescription(sql.OrderDesc()))
			}
		} else {
			switch field {
			case "title":
				query = query.Order(entBook.ByTitle())
			case "image_url":
				query = query.Order(entBook.ByImageURL())
			case "description":
				query = query.Order(entBook.ByDescription())
			}
		}
	}
	entities, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return toSchemas(entities), total, nil
}

func (r *repository) Read(ctx context.Context, id uint64) (*book.Schema, error) {
	entity, err := r.ent.Book.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSchema(entity), nil
}

func (r *repository) Update(ctx context.Context, request *book.UpdateRequest) (*book.Schema, error) {
	update := r.ent.Book.UpdateOneID(request.ID)
	update = update.SetTitle(request.Title)
	update = update.SetPublishedDate(request.PublishedDate)
	update = update.SetImageURL(request.ImageURL)
	update = update.SetDescription(request.Description)
	update = update.SetAuthorID(request.AuthorID)
	update = update.SetUpdatedAt(time.Now())
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSchema(updated), nil
}

func (r *repository) Delete(ctx context.Context, id uint64) error {
	return r.ent.Book.DeleteOneID(id).Exec(ctx)
}

func toSchema(e *gen.Book) *book.Schema {
	if e == nil {
		return nil
	}
	return &book.Schema{
		ID:        e.ID,
		Title:     e.Title,
		PublishedDate:     e.PublishedDate,
		ImageURL:     e.ImageURL,
		Description:     e.Description,
		AuthorID:     e.AuthorID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		DeletedAt: e.DeletedAt,
	}
}

func toSchemas(entities []*gen.Book) []*book.Schema {
	out := make([]*book.Schema, 0, len(entities))
	for _, e := range entities {
		out = append(out, toSchema(e))
	}
	return out
}
