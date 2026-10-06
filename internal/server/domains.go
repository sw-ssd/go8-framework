package server

import (
	"github.com/go-chi/chi/v5"
	"connectrpc.com/connect"

	"codeberg.org/gmhafiz/go8/ent/gen"
	todoService "codeberg.org/gmhafiz/go8/internal/domain/todo/service"
)

// ConnectDomain is a CONNECT service that wires itself given the ent client and router.
// go8cli appends an entry here for each generated resource.
type ConnectDomain struct {
	Name    string
	Register func(ent *gen.Client, r chi.Router, opts ...connect.HandlerOption)
}

// ConnectDomains is the registry go8cli appends a line to.
var ConnectDomains = []ConnectDomain{
	{Name: "todo", Register: todoService.Register},
}
