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
	"codeberg.org/gmhafiz/go8/internal/domain/author"
	"codeberg.org/gmhafiz/go8/internal/domain/author/repository"
	"codeberg.org/gmhafiz/go8/internal/domain/author/usecase"
	"codeberg.org/gmhafiz/go8/internal/utility/filter"
)

// Service implements the generated AuthorServiceHandler.
type Service struct {
	go8v1connect.UnimplementedAuthorServiceHandler
	uc usecase.Author
}

// New builds the author service (repository -> usecase -> service) from the ent client.
func New(ent *gen.Client) *Service {
	return &Service{uc: usecase.New(repository.New(ent))}
}

// go8cli:impl
func (s *Service) Create(ctx context.Context, req *connect.Request[go8v1.CreateAuthorRequest]) (*connect.Response[go8v1.Author], error) {
	schema, err := s.uc.Create(ctx, &author.CreateRequest{
		FirstName: req.Msg.GetFirstName(),
		MiddleName: req.Msg.GetMiddleName(),
		LastName: req.Msg.GetLastName(),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) Get(ctx context.Context, req *connect.Request[go8v1.GetAuthorRequest]) (*connect.Response[go8v1.Author], error) {
	schema, err := s.uc.Read(ctx, uint64(req.Msg.GetId()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) List(ctx context.Context, req *connect.Request[go8v1.ListAuthorsRequest]) (*connect.Response[go8v1.ListAuthorsResponse], error) {
	f := &author.Filter{
		Base: filter.Filter{
			Page:    int(req.Msg.GetPage()),
			Limit:   int(req.Msg.GetPageSize()),
			Offset:  int(req.Msg.GetPageSize()) * (int(req.Msg.GetPage()) - 1),
			Search:  req.Msg.GetSearch() != "",
		},
		FirstName: req.Msg.GetSearch(),
		MiddleName: req.Msg.GetSearch(),
		LastName: req.Msg.GetSearch(),
	}
	schemas, total, err := s.uc.List(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]*go8v1.Author, 0, len(schemas))
	for _, sc := range schemas {
		items = append(items, toProto(sc))
	}
	return connect.NewResponse(&go8v1.ListAuthorsResponse{ Authors: items, Total: int32(total) }), nil
}

func (s *Service) Update(ctx context.Context, req *connect.Request[go8v1.UpdateAuthorRequest]) (*connect.Response[go8v1.Author], error) {
	schema, err := s.uc.Update(ctx, &author.UpdateRequest{
		ID: uint64(req.Msg.GetId()),
		FirstName: req.Msg.GetFirstName(),
		MiddleName: req.Msg.GetMiddleName(),
		LastName: req.Msg.GetLastName(),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) Delete(ctx context.Context, req *connect.Request[go8v1.DeleteAuthorRequest]) (*connect.Response[emptypb.Empty], error) {
	if err := s.uc.Delete(ctx, uint64(req.Msg.GetId())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// Register mounts the CONNECT handler on the chi router.
func Register(ent *gen.Client, r chi.Router, opts ...connect.HandlerOption) {
	path, handler := go8v1connect.NewAuthorServiceHandler(New(ent), opts...)
	r.Handle(path, handler)
}

func toProto(s *author.Schema) *go8v1.Author {
	return &go8v1.Author{
		Id: int64(s.ID),
		FirstName: s.FirstName,
		MiddleName: s.MiddleName,
		LastName: s.LastName,
		CreatedAt: timestamppb.New(s.CreatedAt),
	}
}
