# Introduction

```
            .,*/(#####(/*,.                               .,*((###(/*.
        .*(%%%%%%%%%%%%%%#/.                           .*#%%%%####%%%%#/.
      ./#%%%%#(/,,...,,***.           .......          *#%%%#*.   ,(%%%#/.
     .(#%%%#/.                    .*(#%%%%%%%##/,.     ,(%%%#*    ,(%%%#*.
    .*#%%%#/.    ..........     .*#%%%%#(/((#%%%%(,     ,/#%%%#(/#%%%#(,
    ./#%%%(*    ,#%%%%%%%%(*   .*#%%%#*     .*#%%%#,      *(%%%%%%%#(,.
    ./#%%%#*    ,(((##%%%%(*   ,/%%%%/.      .(%%%#/   .*#%%%#(*/(#%%%#/,
     ,#%%%#(.        ,#%%%(*   ,/%%%%/.      .(%%%#/  ,/%%%#/.    .*#%%%(.
      *#%%%%(*.      ,#%%%(*   .*#%%%#*     ./#%%%#,  ,(%%%#*      .(%%%#*
       ,(#%%%%%##(((##%%%%(*    .*#%%%%#(((##%%%%(,   .*#%%%##(///(#%%%#/.
         .*/###%%%%%%%###(/,      .,/##%%%%%##(/,.      .*(##%%%%%%##(*,
              .........                ......                .......
```

A starter kit for Go API development, extended with [CONNECT RPC](https://connectrpc.com) and a code-generation CLI.

Inspired by [How I write HTTP services after eight years](https://pace.dev/blog/2018/05/09/how-I-write-http-services-after-eight-years.html).

The kit is a **Go + Postgres + CONNECT RPC (connect-go) + ent + sqlx + go8cli** API starter. [Chi](https://github.com/go-chi/chi) is retained for infrastructure routes (health, version, swagger, authentication). Business domains are exposed as CONNECT services and generated end-to-end with `go8cli`.

# Motivation

On the topic of API development there are two camps: using a framework (echo, gin, buffalo) versus starting small and adding only what you need. The second option requires clear separation between controller, business logic, and database operations, with dependencies injected from outside in. Being modular makes swapping a library (router, database) much easier.

This kit adds CONNECT RPC on top of that layered architecture: each domain exposes a typed service over HTTP/JSON and gRPC, and `go8cli` scaffolds the full stack (proto → connect service → use case → ent repository → frontend) from one command.

# Features

- [x] [CONNECT RPC](https://connectrpc.com) services (HTTP/JSON + gRPC) via [connect-go](https://github.com/connectrpc/connect-go)
- [x] [Chi router](https://github.com/go-chi/chi) for infrastructure routes (health, version, swagger, auth)
- [x] Layered architecture: connect service → use case → repository
- [x] Database operations with [sqlx](https://github.com/jmoiron/sqlx) and [ent](https://entgo.io/docs/getting-started)
- [x] Database migration with [goose](https://github.com/pressly/goose)
- [x] Input [validation](https://github.com/go-playground/validator)
- [x] Single `.env` file or environment-variable configuration
- [x] CORS, request logging, OpenTelemetry
- [x] Cache layer (LRU / Redis)
- [x] Cookie-based session authentication
- [x] [Swagger](https://github.com/swaggo/swag) docs for infrastructure routes
- [x] [Task](https://taskfile.dev) task runner for migrate, generate, lint, test, run
- [x] `go8cli` code generator: scaffold a full CRUD resource (proto, connect service, use case, ent schema, migration, frontend) in one command
- [x] Unit testing of use case and repository with mocks

# Quick Start

Use the latest supported [Go](https://go.dev/dl/go1.27.0.linux-amd64.tar.gz) (>= 1.26). A database is required. This repo uses **podman** (docker-compose files are drop-in compatible with `docker-compose` too).

Get it:

```shell
git clone https://codeberg.com/gmhafiz/go8
cd go8
```

## Database

Bring up Postgres with podman-compose (the `docker-compose-infra.yml` reads DB credentials from `.env` or the environment):

```sh
podman-compose -f docker-compose-infra.yml up -d postgres
# docker-compose -f docker-compose-infra.yml up -d postgres   # if you use docker
```

Create tables and seed data (the seed user is needed for authentication):

```shell
go run cmd/migrate/main.go
go run cmd/seed/main.go
# or: task migrate && task seed
```

Run the API:

```shell
go run cmd/go8/main.go
# or: task run
```

The server serves infrastructure routes on Chi and CONNECT services mounted on the same router. To list every registered route (infra + connect):

```shell
go run cmd/route/main.go
# or: task routes
```

## Calling a CONNECT service

CONNECT services accept HTTP/JSON (and gRPC). The path is `/go8.v1.<Service>/<Method>`. Field names use camelCase in JSON. `uint64` ids are strings in JSON.

Create a book:

```shell
curl -X POST 'http://localhost:3080/go8.v1.BookService/Create' \
  --header 'Content-Type: application/json' \
  --data-raw '{
    "title": "Test title",
    "imageUrl": "https://example.com",
    "publishedDate": "2020-07-31T15:04:05Z",
    "description": "test description",
    "authorId": "1"
  }'
```

List books:

```shell
curl -X POST 'http://localhost:3080/go8.v1.BookService/List' \
  --header 'Content-Type: application/json' \
  --data-raw '{"page": 1, "pageSize": 30}'
```

The same services are callable from the generated TypeScript client in `web/` (SolidJS + TanStack Query/Router + connect-web).

# go8cli

`go8cli` scaffolds a full CRUD resource across the stack from one command.

```shell
# one-shot resource (model + create + read + update + delete + frontend)
go run cmd/go8cli/main.go resource book \
  --field title:string --field published_date:date --field image_url:string? --field description:string --field author_id:int64
```

Per-step subcommands (each interactive or non-interactive):

```shell
go run cmd/go8cli/main.go model    author --field first_name:string --field last_name:string
go run cmd/go8cli/main.go create   author --field first_name:string --field last_name:string
go run cmd/go8cli/main.go read     author
go run cmd/go8cli/main.go update   author --field first_name:string --field last_name:string
go run cmd/go8cli/main.go delete   author
go run cmd/go8cli/main.go generate author   # render any template / custom command
```

Field types: `string | int | int64 | bool | float | date`. Append `?` for an optional (nullable) column, e.g. `image_url:string?`. Edges are expressed as `int64` foreign-key fields (no native ent edges in generated code).

After generating, regenerate code:

```shell
go generate ./ent   # ent models
task gen            # buf -> Go + TS connect stubs
```

## Extensibility

- **Custom templates**: `go8cli generate <name>` renders any `.tmpl` from `--templates` / `GO8_TEMPLATES`, falling back to embedded defaults. Same-name user templates win.
- **Custom commands**: declare `*.yaml` manifests under `.go8/commands/` with `prompts` + `steps` (`template`+`out`+`mode` or `shell`). Run with `go8cli <cmd>` / `go8cli run <cmd>`.

# Architecture

```
proto (api/go8/v1/*.proto)
  -> buf generate -> connect service interface + Go/TS stubs
connect service (internal/domain/<d>/service)   // validates, maps proto <-> domain
  -> use case (internal/domain/<d>/usecase)     // business logic
    -> repository (internal/domain/<d>/repository) // ent (or sqlx)
ent client (s.ent) injected from server
```

Chi handles only infrastructure: `/health`, `/version`, `/swagger`, authentication. Each CONNECT service is mounted on the Chi router as an `http.Handler`.

# Structure

```
cmd/
  go8/      server entrypoint
  go8cli/   code generator
  migrate/  goose runner
  seed/     seed data
  route/    prints registered routes
api/go8/v1/        protobuf definitions
ent/                ent schemas + generated code (ent/gen)
gen/                buf-generated Go (proto + connect)
internal/
  server/           Server wiring (ConnectDomains registry)
  domain/<d>/       service / usecase / repository / model
web/                SolidJS + TanStack + connect-web reference app
database/migrations goose SQL
```

`internal/server/server.go` holds the `Server` struct and `Init()` (config, database, validator, router, middleware, domains). `s.InitDomains()` wires infrastructure (Chi) and then iterates `ConnectDomains` to mount each generated CONNECT service.

## Initialize a domain

Domains are registered in `internal/server/domains.go` as a `ConnectDomains` slice. `go8cli` appends one line per generated resource:

```go
var ConnectDomains = []ConnectDomain{
    {Name: "todo",   Register: todoService.Register},
    {Name: "author", Register: authorService.Register},
    {Name: "book",   Register: bookService.Register},
    // go8cli:domains
}
```

A `Register` function mounts the CONNECT handler on the Chi router:

```go
func Register(ent *gen.Client, r chi.Router, opts ...connect.HandlerOption) {
    path, handler := go8v1connect.NewBookServiceHandler(New(ent), opts...)
    r.Handle(path, handler)
}
```

# Database

Migrations live in `database/migrations` (goose). Current resources: `users`, `sessions`, `todos`, `authors`, `books`.

## Migrate

```sh
task migrate          # up all
task migrate:create NAME=create_a_table
task migrate:rollback
```

Or directly:

```sh
go run cmd/migrate/main.go
```

# Tooling

[Task](https://github.com/go-task/task) wraps common operations. Install:

```sh
sudo ./scripts/install-task.sh
task -l
```

## Tasks

| Task | What it does |
|------|--------------|
| `task run` | run the API (`go run cmd/go8/main.go`) |
| `task gen` | buf generate (Go + TS connect stubs) |
| `task generate` | `go generate ./...` (regenerate ent mocks etc.) |
| `task migrate` | goose up |
| `task seed` | seed super-admin user |
| `task routes` | print registered routes |
| `task fmt` / `task vet` / `task lint` | format / vet / golangci-lint |
| `task test` | `go test ./...` |
| `task dev` | hot reload (air) |
| `task build` | static linux binary into `./bin` |

# OpenTelemetry

Infra (Grafana, otel-collector, Prometheus, Loki, Jaeger) starts with podman-compose:

```sh
podman-compose -f docker-compose-infra.yml up -d
# docker-compose -f docker-compose-infra.yml up -d
```

Enable OTel in the server via `OTEL_ENABLE=true` (env or `.env`), then restart the API with `task run` / `docker-compose up -d`.

# Authentication

Cookie-based session auth (library [alexedwards/scs](https://github.com/alexedwards/scs)) on Chi infrastructure routes. Migrate + seed first:

```shell
go run cmd/migrate/main.go
go run cmd/seed/main.go
```

Register, then login (a session cookie is returned):

```shell
curl -vX POST -H 'content-type: application/json' 'http://localhost:3080/api/v1/register' \
  -d '{"first_name":"Hafiz","last_name":"Shafruddin","email":"email@example.com","password":"highEntropyPassword"}'

curl -vX POST -H 'content-type: application/json' 'http://localhost:3080/api/v1/login' \
  -d '{"email":"email@example.com","password":"highEntropyPassword"}'
```

Attach the `session` cookie for protected routes. Logout via `/v1/logout`.

# Cache

A cache layer (LRU in-memory or Redis) sits between use case and repository to avoid repeated DB hits. See `internal/domain/author/repository/{lru,redis}.go` for the original pattern; generated connect resources use the plain ent repository.

# Swagger docs

Infrastructure routes (health, version, auth) are documented with swag. Enable with `API_RUN_SWAGGER=true`, run `task swagger`, and open `http://localhost:3080/swagger/`.

# Testing

```sh
task test        # go test ./...
task test:unit   # go test -short ./...
```

Unit tests cover use case and repository with mocks. Integration tests (`internal/domain/authentication`) need a live Postgres (via `ory/dockertest`). Real end-to-end coverage lives in `e2e/` (behind the `e2e` build tag): it spins up a Testcontainers Postgres, runs the API in-process, builds and previews the SolidJS app, and drives the Todo CRUD in a native headless Chromium via Playwright. Run it with `task test:e2e` (`go test -tags e2e ./e2e`). The container runtime is selected via `DOCKER_HOST` (works with Docker or Podman; on Podman also set `TESTCONTAINERS_RYUK_DISABLED=true`). Connect service unit tests additionally cover the proto ↔ domain mapping.

# Build

```sh
task build                       # static binary -> ./bin
CGO_ENABLED=0 GOOS=linux go build -o ./server ./cmd/go8/main.go
```

# Run with podman

```sh
podman-compose up -d postgres
task run
# or full stack:
podman-compose up -d
```

# TODO

- Connect-client CLI examples and generated docs.
- gRPC gateway / REST bridge for CONNECT services.
- Native ent edge support in `go8cli`.

# Acknowledgements

Based on [gmhafiz/go8](https://codeberg.org/gmhafiz/go8). CONNECT RPC by [connectrpc](https://connectrpc.com).

## Testing

Unit tests cover use cases and repositories with mocks (`internal/domain/.../service/*_test.go`). There is **no real e2e in `go test ./...`** — the e2e suite sits behind the `//go:build e2e` tag so it does not pull Docker/Playwright.

Run the full web+API e2e:

```shell
# Podman (needs the socket + ryuk disabled: Podman lacks the bridge network ryuk requires)
export DOCKER_HOST=unix:///run/podman/podman.sock
TESTCONTAINERS_RYUK_DISABLED=true task test:e2e

# Docker
task test:e2e
```

The e2e spins up Postgres via testcontainers, runs the API in-process (`server.New→Init→Migrate` then `Run` on `0.0.0.0:8080`), builds the web with `VITE_API_BASE` baked in and serves it with `vite preview` on :3000, then drives the Todo CRUD page with a NATIVE headless Chromium via playwright-go. It does NOT use a containerized browser — see Constraints.

## Constraints

- **Env vars are `NEWAPI_*`** (`envconfig` prefix `NewAPI`): `NEWAPI_PORT` (default 3080), `NEWAPI_HOST`, `DB_*`, `CORS_ALLOWED_ORIGINS`, `API_RUN_SWAGGER`. `env.example`'s `API_PORT` is wrong and ignored.
- **Migrations** (`database/migrations`) use goose: `-- +goose Up/Down` wrapping SQL in `-- +goose StatementBegin/End`; version = unique timestamp prefix. `goose` panics on duplicate versions or unparsed SQL.
- **CONNECT health** is `GET /api/health` → `{"status":200}` (not `/health`).
- **CONNECT routing**: register with `r.Handle(path+"*", handler)` where `path` ends in `/`; `r.Handle(path, handler)` strips the prefix and 404s every method.
- **Browser for e2e is native** (playwright-go on the host). Containerized browsers do not work for this SPA: Chromium binds its debug port to `127.0.0.1`; Lightpanda (`lightpanda/browser:nightly`) binds `0.0.0.0` but does not render the Vite/SolidJS ES-module app (DOM stays an empty `div#root`). chromedp was also verified to return an empty DOM. Keep the browser native.
