# AGENTS.md — go8-framework

Guidance for AI coding agents. Read before editing.

## What this is
- Go + Postgres API starter kit, extended with CONNECT RPC (`connect-go`) and a code-generation CLI (`go8cli`).
- Reference SolidJS web app under `web/` (connect-web + TanStack Query/Router).
- Chi is retained ONLY for infrastructure routes (health, version, swagger, auth). Business domains are CONNECT services.

## Architecture (verified)
- CONNECT service fully replaces Chi for business-domain API; Chi is infra/health only.
- Repo layer: `ent` + `sqlx` (DREL was evaluated and abandoned). Usecase layer preserved unchanged.
- `go8cli` scaffolds full-stack CRUD from one command; it appends one line to `internal/server/domains.go` (registry pattern — does NOT rewrite an init file).
- e2e topology: Postgres via testcontainers (Docker/Podman) → API in-process (`server.New→Init→Migrate` + `Run` on `0.0.0.0:8080`) → web built with `VITE_API_BASE` baked, served by `vite preview` on :3000 → browser is NATIVE headless Chromium via playwright-go (NOT containerized).

## Key conventions
- Migrations: goose SQL in `database/migrations`. Format MUST be `-- +goose Up` / `-- +goose Down`, wrapping SQL in `-- +goose StatementBegin` / `-- +goose StatementEnd` (match the `users`/`sessions` files). Version = file timestamp prefix; versions MUST be unique across files.
- Config: `envconfig` prefix is `NewAPI` → real env vars are `NEWAPI_*` (e.g. `NEWAPI_PORT`, default 3080). `env.example`'s `API_PORT` is WRONG and ignored.
- CONNECT routing: register with `r.Handle(path+"*", handler)` where `path` ends in `/` — chi otherwise strips the prefix and `/X/Create` 404s. Do NOT use `r.Handle(path, handler)`.
- Health route is `GET /api/health` (returns `{"status":200}`), NOT `/health`.
- `go8cli` templates: backend Go uses `{{ }}`; frontend TSX uses `[[ ]]` (TSX's literal `{{}}` would be parsed as a Go template action).

## Running
- API: `task run` (or `go run cmd/go8/main.go` with `NEWAPI_*` env).
- Web: `cd web && npm install && npm run dev` (Vite, port 3000), or `npm run build && npx vite preview --host --port 3000`.
- Migrate/seed: `task migrate && task seed`.
- e2e: `task test:e2e` — needs a container runtime (Docker or Podman). On Podman set `DOCKER_HOST=unix:///run/podman/podman.sock` and `TESTCONTAINERS_RYUK_DISABLED=true` (Podman lacks the `bridge` network ryuk needs). The browser is NATIVE Playwright Chromium on the host, not a container.
- Verify e2e locally without a runner is not possible; `go build ./...` / `go vet ./...` only prove compilation.

## e2e gotchas (verified the hard way)
- Containerized browsers do NOT work for this app:
  - Chromium binds its remote-debugging port to `127.0.0.1` (host can't reach it across the container boundary).
  - Lightpanda (`lightpanda/browser:nightly`) binds `0.0.0.0` but does NOT render the SolidJS/Vite ES-module SPA — the DOM stays an empty `<div id="root">` (~230 bytes). It works for simple/crawl pages (`campfire-commerce` demo) but not module SPAs.
  - chromedp v0.15.1 (the version Lightpanda's own demo uses) was also verified to return an empty DOM for this app.
  - Conclusion: keep the browser NATIVE (playwright-go on the host). Do NOT pursue containerized browsers.
- `server.start()` previously called `log.Fatal` on any `ListenAndServe` error, including `http.ErrServerClosed`, so a graceful `Shutdown` triggered `os.Exit(1)`. Fixed to ignore `ErrServerClosed`.
- `Register` used `r.Handle(path, handler)` and 404'd on every method; fixed to `r.Handle(path+"*", handler)`.

## Commit style
- Conventional commits: `feat:`, `fix:`, `docs:`, `chore:`, `test:`. Keep commits reviewable and scoped.

## Do NOT
- Do not containerize the e2e browser (see gotchas).
- Do not remove the `//go:build e2e` build tag on `e2e/e2e_test.go` — it keeps `go test ./...` free of Docker/Playwright deps.
- Do not commit `web/dist/` (gitignored; e2e regenerates it).
