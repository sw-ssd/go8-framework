package repository

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"codeberg.org/gmhafiz/go8/ent/gen"
	entTodo "codeberg.org/gmhafiz/go8/ent/gen/todo"
	"codeberg.org/gmhafiz/go8/internal/domain/todo"
)

type repository struct {
	ent *gen.Client
}

// Todo is the persistence boundary for the todo resource (ent-backed).
type Todo interface {
	Create(ctx context.Context, request *todo.CreateRequest) (*todo.Schema, error)
	List(ctx context.Context, f *todo.Filter) ([]*todo.Schema, int, error)
	Read(ctx context.Context, id uint64) (*todo.Schema, error)
	Update(ctx context.Context, request *todo.UpdateRequest) (*todo.Schema, error)
	Delete(ctx context.Context, id uint64) error
}

func New(ent *gen.Client) *repository {
	return &repository{ent: ent}
}

func (r *repository) Create(ctx context.Context, request *todo.CreateRequest) (*todo.Schema, error) {
	created, err := r.ent.Todo.Create().
		SetTitle(request.Title).
		SetPriority(request.Priority).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSchema(created), nil
}

func (r *repository) List(ctx context.Context, f *todo.Filter) ([]*todo.Schema, int, error) {
	query := r.ent.Todo.Query()
	if f.Title != "" {
		query = query.Where(entTodo.TitleContains(f.Title))
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
			case "priority":
				query = query.Order(entTodo.ByPriority(sql.OrderDesc()))
			case "title":
				query = query.Order(entTodo.ByTitle(sql.OrderDesc()))
			}
		} else {
			switch field {
			case "priority":
				query = query.Order(entTodo.ByPriority())
			case "title":
				query = query.Order(entTodo.ByTitle())
			}
		}
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return toSchemas(entities), total, nil
}

func (r *repository) Read(ctx context.Context, id uint64) (*todo.Schema, error) {
	entity, err := r.ent.Todo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSchema(entity), nil
}

func (r *repository) Update(ctx context.Context, request *todo.UpdateRequest) (*todo.Schema, error) {
	update := r.ent.Todo.UpdateOneID(request.ID)
	if request.Title != "" {
		update = update.SetTitle(request.Title)
	}
	update = update.SetDone(request.Done).SetPriority(request.Priority).SetUpdatedAt(time.Now())
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSchema(updated), nil
}

func (r *repository) Delete(ctx context.Context, id uint64) error {
	return r.ent.Todo.DeleteOneID(id).Exec(ctx)
}

func toSchema(e *gen.Todo) *todo.Schema {
	if e == nil {
		return nil
	}
	return &todo.Schema{
		ID:        e.ID,
		Title:     e.Title,
		Done:      e.Done,
		Priority:  e.Priority,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		DeletedAt: e.DeletedAt,
	}
}

func toSchemas(entities []*gen.Todo) []*todo.Schema {
	out := make([]*todo.Schema, 0, len(entities))
	for _, e := range entities {
		out = append(out, toSchema(e))
	}
	return out
}
