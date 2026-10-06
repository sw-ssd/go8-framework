package book

import (
	"net/url"

	"codeberg.org/gmhafiz/go8/internal/utility/filter"
)

type Filter struct {
	Base filter.Filter
	Title string `json:"title"`
	PublishedDate string `json:"published_date"`
	ImageURL string `json:"image_url"`
	Description string `json:"description"`
	AuthorID string `json:"author_id"`
}

func Filters(queries url.Values) *Filter {
	f := filter.New(queries)
	if queries.Has("title") {
		f.Search = true
	}
	if queries.Has("published_date") {
		f.Search = true
	}
	if queries.Has("image_url") {
		f.Search = true
	}
	if queries.Has("description") {
		f.Search = true
	}
	if queries.Has("author_id") {
		f.Search = true
	}
	return &Filter{
		Base: *f,
		Title: queries.Get("title"),
		PublishedDate: queries.Get("published_date"),
		ImageURL: queries.Get("image_url"),
		Description: queries.Get("description"),
		AuthorID: queries.Get("author_id"),
	}
}
