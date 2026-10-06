package service

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	go8v1 "codeberg.org/gmhafiz/go8/gen/api/go8/v1"
	go8v1connect "codeberg.org/gmhafiz/go8/gen/api/go8/v1/go8v1connect"
	"codeberg.org/gmhafiz/go8/internal/domain/todo"
	"codeberg.org/gmhafiz/go8/internal/domain/todo/usecase"
)

type fakeRepo struct{}

func (fakeRepo) Create(_ context.Context, r *todo.CreateRequest) (*todo.Schema, error) {
	return &todo.Schema{ID: 7, Title: r.Title, Done: false, Priority: r.Priority}, nil
}
func (fakeRepo) List(_ context.Context, _ *todo.Filter) ([]*todo.Schema, int, error) {
	return []*todo.Schema{{ID: 7, Title: "x", Priority: 1}}, 1, nil
}
func (fakeRepo) Read(_ context.Context, id uint64) (*todo.Schema, error) {
	return &todo.Schema{ID: id, Title: "x"}, nil
}
func (fakeRepo) Update(_ context.Context, r *todo.UpdateRequest) (*todo.Schema, error) {
	return &todo.Schema{ID: r.ID, Title: r.Title, Done: r.Done, Priority: r.Priority}, nil
}
func (fakeRepo) Delete(_ context.Context, _ uint64) error { return nil }

func TestServiceMapping(t *testing.T) {
	s := &Service{uc: usecase.New(fakeRepo{})}
	ctx := context.Background()

	resp, err := s.Create(ctx, connect.NewRequest(&go8v1.CreateTodoRequest{Title: "hi", Priority: 3}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetId() != 7 || resp.Msg.GetTitle() != "hi" || resp.Msg.GetPriority() != 3 {
		t.Fatalf("create mapping: %+v", resp.Msg)
	}

	lr, err := s.List(ctx, connect.NewRequest(&go8v1.ListTodosRequest{Page: 1, PageSize: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if lr.Msg.GetTotal() != 1 || len(lr.Msg.GetTodos()) != 1 {
		t.Fatalf("list mapping: %+v", lr.Msg)
	}

	g, err := s.Get(ctx, connect.NewRequest(&go8v1.GetTodoRequest{Id: 9}))
	if err != nil {
		t.Fatal(err)
	}
	if g.Msg.GetId() != 9 {
		t.Fatalf("get mapping: %+v", g.Msg)
	}

	u, err := s.Update(ctx, connect.NewRequest(&go8v1.UpdateTodoRequest{Id: 9, Title: "y", Done: true, Priority: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if u.Msg.GetDone() != true || u.Msg.GetPriority() != 2 {
		t.Fatalf("update mapping: %+v", u.Msg)
	}

	if _, err := s.Delete(ctx, connect.NewRequest(&go8v1.DeleteTodoRequest{Id: 9})); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(go8v1connect.TodoServiceCreateProcedure, "TodoService/Create") {
		t.Fatalf("unexpected connect procedure path: %s", go8v1connect.TodoServiceCreateProcedure)
	}
}
