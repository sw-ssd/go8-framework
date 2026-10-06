package todo

import (
	"net/url"

	"codeberg.org/gmhafiz/go8/internal/utility/filter"
)

type Filter struct {
	Base filter.Filter

	Title string `json:"title"`
}

func Filters(queries url.Values) *Filter {
	f := filter.New(queries)
	if queries.Has("title") {
		f.Search = true
	}
	return &Filter{
		Base: *f,
		Title: queries.Get("title"),
	}
}
