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
	"codeberg.org/gmhafiz/go8/internal/domain/book"
	"codeberg.org/gmhafiz/go8/internal/domain/book/repository"
	"codeberg.org/gmhafiz/go8/internal/domain/book/usecase"
	"codeberg.org/gmhafiz/go8/internal/utility/filter"
)

// Service implements the generated BookServiceHandler.
type Service struct {
	go8v1connect.UnimplementedBookServiceHandler
	uc usecase.Book
}

// New builds the book service (repository -> usecase -> service) from the ent client.
func New(ent *gen.Client) *Service {
	return &Service{uc: usecase.New(repository.New(ent))}
}

// go8cli:impl
func (s *Service) Create(ctx context.Context, req *connect.Request[go8v1.CreateBookRequest]) (*connect.Response[go8v1.Book], error) {
	schema, err := s.uc.Create(ctx, &book.CreateRequest{
		Title: req.Msg.GetTitle(),
		PublishedDate: req.Msg.GetPublishedDate().AsTime(),
		ImageURL: req.Msg.GetImageUrl(),
		Description: req.Msg.GetDescription(),
		AuthorID: req.Msg.GetAuthorId(),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) Get(ctx context.Context, req *connect.Request[go8v1.GetBookRequest]) (*connect.Response[go8v1.Book], error) {
	schema, err := s.uc.Read(ctx, uint64(req.Msg.GetId()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) List(ctx context.Context, req *connect.Request[go8v1.ListBooksRequest]) (*connect.Response[go8v1.ListBooksResponse], error) {
	f := &book.Filter{
		Base: filter.Filter{
			Page:    int(req.Msg.GetPage()),
			Limit:   int(req.Msg.GetPageSize()),
			Offset:  int(req.Msg.GetPageSize()) * (int(req.Msg.GetPage()) - 1),
			Search:  req.Msg.GetSearch() != "",
		},
		Title: req.Msg.GetSearch(),
		PublishedDate: req.Msg.GetSearch(),
		ImageURL: req.Msg.GetSearch(),
		Description: req.Msg.GetSearch(),
		AuthorID: req.Msg.GetSearch(),
	}
	schemas, total, err := s.uc.List(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]*go8v1.Book, 0, len(schemas))
	for _, sc := range schemas {
		items = append(items, toProto(sc))
	}
	return connect.NewResponse(&go8v1.ListBooksResponse{ Books: items, Total: int32(total) }), nil
}

func (s *Service) Update(ctx context.Context, req *connect.Request[go8v1.UpdateBookRequest]) (*connect.Response[go8v1.Book], error) {
	schema, err := s.uc.Update(ctx, &book.UpdateRequest{
		ID: uint64(req.Msg.GetId()),
		Title: req.Msg.GetTitle(),
		PublishedDate: req.Msg.GetPublishedDate().AsTime(),
		ImageURL: req.Msg.GetImageUrl(),
		Description: req.Msg.GetDescription(),
		AuthorID: req.Msg.GetAuthorId(),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toProto(schema)), nil
}

func (s *Service) Delete(ctx context.Context, req *connect.Request[go8v1.DeleteBookRequest]) (*connect.Response[emptypb.Empty], error) {
	if err := s.uc.Delete(ctx, uint64(req.Msg.GetId())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// Register mounts the CONNECT handler on the chi router.
func Register(ent *gen.Client, r chi.Router, opts ...connect.HandlerOption) {
	path, handler := go8v1connect.NewBookServiceHandler(New(ent), opts...)
	r.Handle(path, handler)
}

func toProto(s *book.Schema) *go8v1.Book {
	return &go8v1.Book{
		Id: int64(s.ID),
		Title: s.Title,
		PublishedDate: timestamppb.New(s.PublishedDate),
		ImageUrl: s.ImageURL,
		Description: s.Description,
		AuthorId: s.AuthorID,
		CreatedAt: timestamppb.New(s.CreatedAt),
	}
}
