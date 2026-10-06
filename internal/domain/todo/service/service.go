package service

import (
	"context"

	"github.com/go-chi/chi/v5"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	go8v1 "codeberg.org/gmhafiz/go8/gen/api/go8/v1"
	go8v1connect "codeberg.org/gmhafiz/go8/gen/api/go8/v1/go8v1connect"
	"codeberg.org/gmhafiz/go8/ent/gen"
	"codeberg.org/gmhafiz/go8/internal/domain/todo"
	"codeberg.org/gmhafiz/go8/internal/domain/todo/repository"
	"codeberg.org/gmhafiz/go8/internal/domain/todo/usecase"
	"codeberg.org/gmhafiz/go8/internal/utility/filter"
)

// Service implements the generated TodoServiceHandler.
type Service struct {
	go8v1connect.UnimplementedTodoServiceHandler
	uc usecase.Todo
}

// New builds the todo service (repository -> usecase -> service) from the ent client.
func New(ent *gen.Client) *Service {
	return &Service{uc: usecase.New(repository.New(ent))}
}

func (s *Service) Create(ctx context.Context, req *connect.Request[go8v1.CreateTodoRequest]) (*connect.Response[go8v1.Todo], error) {
	schema, err := s.uc.Create(ctx, &todo.CreateRequest{
		Title:    req.Msg.GetTitle(),
		Priority: int(req.Msg.GetPriority()),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) Get(ctx context.Context, req *connect.Request[go8v1.GetTodoRequest]) (*connect.Response[go8v1.Todo], error) {
	schema, err := s.uc.Read(ctx, uint64(req.Msg.GetId()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) List(ctx context.Context, req *connect.Request[go8v1.ListTodosRequest]) (*connect.Response[go8v1.ListTodosResponse], error) {
	f := &todo.Filter{
		Base: filter.Filter{
			Page:    int(req.Msg.GetPage()),
			Limit:   int(req.Msg.GetPageSize()),
			Offset:  int(req.Msg.GetPageSize()) * (int(req.Msg.GetPage()) - 1),
			Search:  req.Msg.GetSearch() != "",
		},
		Title: req.Msg.GetSearch(),
	}
	schemas, total, err := s.uc.List(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]*go8v1.Todo, 0, len(schemas))
	for _, sc := range schemas {
		items = append(items, toProto(sc))
	}
	return connect.NewResponse(&go8v1.ListTodosResponse{Todos: items, Total: int32(total)}), nil
}

func (s *Service) Update(ctx context.Context, req *connect.Request[go8v1.UpdateTodoRequest]) (*connect.Response[go8v1.Todo], error) {
	schema, err := s.uc.Update(ctx, &todo.UpdateRequest{
		ID:       uint64(req.Msg.GetId()),
		Title:    req.Msg.GetTitle(),
		Done:     req.Msg.GetDone(),
		Priority: int(req.Msg.GetPriority()),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) Delete(ctx context.Context, req *connect.Request[go8v1.DeleteTodoRequest]) (*connect.Response[emptypb.Empty], error) {
	if err := s.uc.Delete(ctx, uint64(req.Msg.GetId())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// Register mounts the CONNECT handler on the chi router.
func Register(ent *gen.Client, r chi.Router, opts ...connect.HandlerOption) {
	path, handler := go8v1connect.NewTodoServiceHandler(New(ent), opts...)
	r.Handle(path, handler)
}

func toProto(s *todo.Schema) *go8v1.Todo {
	return &go8v1.Todo{
		Id:        int64(s.ID),
		Title:     s.Title,
		Done:      s.Done,
		Priority:  int32(s.Priority),
		CreatedAt: timestamppb.New(s.CreatedAt),
	}
}
