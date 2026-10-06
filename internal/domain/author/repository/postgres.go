package repository

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"codeberg.org/gmhafiz/go8/ent/gen"
	entAuthor "codeberg.org/gmhafiz/go8/ent/gen/author"
	"codeberg.org/gmhafiz/go8/internal/domain/author"
)

type repository struct {
	ent *gen.Client
}

// Author is the persistence boundary for the author resource (ent-backed).
type Author interface {
	// go8cli:iface
	Create(ctx context.Context, request *author.CreateRequest) (*author.Schema, error)
	List(ctx context.Context, f *author.Filter) ([]*author.Schema, int, error)
	Read(ctx context.Context, id uint64) (*author.Schema, error)
	Update(ctx context.Context, request *author.UpdateRequest) (*author.Schema, error)
	Delete(ctx context.Context, id uint64) error
}

func New(ent *gen.Client) *repository {
	return &repository{ent: ent}
}

// go8cli:impl
func (r *repository) Create(ctx context.Context, request *author.CreateRequest) (*author.Schema, error) {
	create := r.ent.Author.Create()
	create = create.SetFirstName(request.FirstName)
	create = create.SetMiddleName(request.MiddleName)
	create = create.SetLastName(request.LastName)
	create = create.SetCreatedAt(time.Now())
	created, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSchema(created), nil
}

func (r *repository) List(ctx context.Context, f *author.Filter) ([]*author.Schema, int, error) {
	query := r.ent.Author.Query()
	if f.FirstName != "" {
		query = query.Where(entAuthor.FirstNameContains(f.FirstName))
	}
	if f.MiddleName != "" {
		query = query.Where(entAuthor.MiddleNameContains(f.MiddleName))
	}
	if f.LastName != "" {
		query = query.Where(entAuthor.LastNameContains(f.LastName))
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
			case "first_name":
				query = query.Order(entAuthor.ByFirstName(sql.OrderDesc()))
			case "middle_name":
				query = query.Order(entAuthor.ByMiddleName(sql.OrderDesc()))
			case "last_name":
				query = query.Order(entAuthor.ByLastName(sql.OrderDesc()))
			}
		} else {
			switch field {
			case "first_name":
				query = query.Order(entAuthor.ByFirstName())
			case "middle_name":
				query = query.Order(entAuthor.ByMiddleName())
			case "last_name":
				query = query.Order(entAuthor.ByLastName())
			}
		}
	}
	entities, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return toSchemas(entities), total, nil
}

func (r *repository) Read(ctx context.Context, id uint64) (*author.Schema, error) {
	entity, err := r.ent.Author.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSchema(entity), nil
}

func (r *repository) Update(ctx context.Context, request *author.UpdateRequest) (*author.Schema, error) {
	update := r.ent.Author.UpdateOneID(request.ID)
	update = update.SetFirstName(request.FirstName)
	update = update.SetMiddleName(request.MiddleName)
	update = update.SetLastName(request.LastName)
	update = update.SetUpdatedAt(time.Now())
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSchema(updated), nil
}

func (r *repository) Delete(ctx context.Context, id uint64) error {
	return r.ent.Author.DeleteOneID(id).Exec(ctx)
}

func toSchema(e *gen.Author) *author.Schema {
	if e == nil {
		return nil
	}
	return &author.Schema{
		ID:        e.ID,
		FirstName:     e.FirstName,
		MiddleName:     e.MiddleName,
		LastName:     e.LastName,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		DeletedAt: e.DeletedAt,
	}
}

func toSchemas(entities []*gen.Author) []*author.Schema {
	out := make([]*author.Schema, 0, len(entities))
	for _, e := range entities {
		out = append(out, toSchema(e))
	}
	return out
}
