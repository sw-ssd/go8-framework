# 實作計畫：Go8 CONNECT RPC + DREL ORM + SolidJS 前端 + go8cli

日期：2026-10-06
對應設計：docs/superpowers/specs/2026-10-06-go8-connect-drel-cli-design.md
採用方案：A（垂直切片 `todo` 先行 → go8cli 模板化 → 對 `author`/`book` 套用達成全面替換）

## Scope Check

設計含 4 子系統（buf/connect 後端、DREL 資料層、SolidJS 前端、go8cli），但彼此經「切片 → 模板」耦合，且已自拆為 P0–P4，每階段皆產出可獨立驗證的軟體。故採**單一計畫、階段化任務**，不拆 4 份。

工程師假設：熟悉 Go 與前端，但不熟 buf/connect/drel/TanStack/SolidJS。本計畫寫足工具安裝、指令、檔案職責、程式碼骨架、測試與驗證。

## 工具鏈安裝（P0 前置，全員需知）

- **buf**（proto 管理 + 程式碼生成）：
  `go install github.com/bufbuild/buf/cmd/buf@latest`（或 `brew install bufbuild/buf/buf`）。驗證：`buf --version`。
- **drel**（ORM 程式碼生成）：
  `go install github.com/alternayte/drel/cmd/drel@latest`。驗證：`drel --help`（確認 generate 子命令與設定檔格式）。
- **connect-go**（後端依賴，go get）：
  `github.com/connectrpc/connect-go`、`google.golang.org/protobuf`。
- **前端**（npm）：Node 18+。`cd web && npm init -y` 後安裝依賴（見 §前端任務）。
- **Taskfile**：既有 `Taskfile.yml` 擴充 `gen` 目標串接 `buf generate` 與 `drel generate`。

## File Structure（本計畫觸及的檔案與職責）

新增：
- `buf.yaml` — buf 模組、deps（googleapis、connect）、lint/breaking。
- `buf.gen.yaml` — 生成 go + connect-go（→ `gen/`）+ connect-web/es（→ `web/src/gen`）。
- `api/go8/v1/todo.proto` — 參考 domain 的 proto（後續由 CLI 複製此模式）。
- `gen/go8/v1/todo.pb.go`、`todo.connect.go` — buf 生成（納入版控）。
- `internal/db/models/todo.go` — DREL 模型（embed `drel.Model[T]` + `db:` tag）。
- `internal/db/gen/...` — `drel generate` 產出（`DB` 聚合 + query builder），納入版控。
- `internal/domain/todo/{service,usecase,repository,model.go,request.go,resource.go}` — 切片實作。
- `internal/server/domains.go` — 已註冊 domain 清單（註冊表模式）。
- `web/` — SolidJS 參考 app（package.json、vite.config、src/gen、src/lib、src/features/todo、src/routes）。
- `cmd/go8cli/` — 程式碼生成 CLI（main.go + `templates/` 內嵌）。
- `drel.yaml`（或 drel 既定設定檔名）— 指向 `internal/db/models` → `internal/db/gen`。

修改：
- `internal/server/server.go` — 新增 `newDrel()` 開 `drel.DB`，注入 repository；啟動時迭代 `domains.go` 掛載 connect handler。
- `internal/server/initDomains.go` — 改為迭代 `domains.go` 清單（不再逐一 `initX()`）。
- `go.mod` — 加 connect-go、protobuf、drel。
- `Taskfile.yml` — 加 `gen` 目標（`buf generate && drel generate`）。
- `database/migrations/` — 加 todo 的 goose migration（DDL 來源見 P1 spike）。

P3 刪除（clean cutover）：
- `internal/domain/author/handler/*`、`internal/domain/author/repository/postgres.go`（ent 版）、`internal/domain/author/repository/redis.go`、`lru.go`、`search.go`（ent 版）及對應 mock。
- `internal/domain/book/handler/*`、`internal/domain/book/repository/postgres.go`（ent 版）及 mock。
- `ent/`、`ent/gen/` 若無其他引用則移除；`go.mod` 移除 `entgo.io/ent`。

## 任務（依階段）

### P0 — 工具鏈與 buf/drel 配置

**T0.1 安裝工具**
- 步驟：安裝 buf、drel（見上）。`buf --version`、`drel --help` 確認可用。
- 驗證：兩指令皆印出版本/說明。

**T0.2 建立 buf 配置**
- `buf.yaml`：
  ```yaml
  version: v1
  deps:
    - buf.build/googleapis/googleapis
    - buf.build/connectrpc/connect
  breaking:
    use: [FILE]
  lint:
    use: [DEFAULT]
  ```
- `buf.gen.yaml`：
  ```yaml
  version: v1
  managed:
    enabled: true
    go_package_prefix:
      default: codeberg.org/gmhafiz/go8/gen
  plugins:
    - plugin: buf.build/protocolbuffers/go
      out: gen
      opt: paths=source_relative
    - plugin: buf.build/connectrpc/go
      out: gen
      opt: paths=source_relative
    - plugin: buf.build/connectrpc/es
      out: web/src/gen
    - plugin: buf.build/connectrpc/connect-web
      out: web/src/gen
  ```
- 驗證：`buf mod update` 成功；`buf lint` 對空 proto 目錄通過。

**T0.3 建立 drel 配置**
- 依 `drel --help` / README 建立設定檔，指向 `internal/db/models` → 輸出 `internal/db/gen`。
- 驗證：空模型目錄下 `drel generate` 成功產出 `internal/db/gen/db`（含 `Open`）。

**T0.4 Taskfile 加 gen 目標**
- `Taskfile.yml` 加：
  ```yaml
  gen:
    desc: generate proto + drel code
    cmds:
      - buf generate
      - drel generate
  ```
- 驗證：`task gen` 可執行（縱使暫無 proto/model）。

### P1 — 垂直切片 `todo`（端到端）

**T1.1 寫 todo.proto**
- `api/go8/v1/todo.proto`（見設計 §6 形狀），含 `Todo` message 與 `TodoService`（Create/Get/List/Update/Delete）。
- 驗證：`buf lint api/go8/v1/todo.proto` 通過；`buf generate` 產出 `gen/go8/v1/todo.pb.go`、`todo.connect.go`。

**T1.2 DREL 模型 + 生成**
- `internal/db/models/todo.go`：
  ```go
  package models
  import "github.com/alternayte/drel"
  type Todo struct {
      drel.Model[int]
      Title    string `db:"title"`
      Done     bool   `db:"done"`
      Priority int    `db:"priority"`
  }
  func NewTodo(title string, priority int) *Todo { return &Todo{Title: title, Priority: priority} }
  ```
- 執行 `drel generate`；確認 `internal/db/gen` 產出 `Todos` 集合與 `DB.Open`。
- 驗證：`go build ./internal/db/...` 通過。

**T1.3 todo migration（spike：確認 DREL DDL 產出）**
- 先 `drel --help` / README 確認是否產 DDL：
  - 若有：將 DDL 轉為 `database/migrations/0000xx_todo.up.sql` / `.down.sql`（goose 格式）。
  - 若無：手寫 `CREATE TABLE todo (id BIGSERIAL PK, title TEXT, done BOOLEAN, priority INT, created_at TIMESTAMPTZ)` 的 goose migration。
- 驗證：`task migrate` 或 `cmd/migrate` 對測試庫 up 成功；`\d todo` 存在。

**T1.4 usecase（沿用 author 模式）**
- `internal/domain/todo/usecase/usecase.go`：定義 `Todo` interface（Create/List/Read/Update/Delete），實作呼叫 repository。複製 `author/usecase` 結構，型別換成 todo 的 `Schema`/`Filter`/`*Request`。
- `internal/domain/todo/model.go`、`request.go`、`resource.go`：定義領域型別。
- 驗證：`go build ./internal/domain/todo/...`。

**T1.5 DREL repository**
- `internal/domain/todo/repository/repository.go`：
  ```go
  type Repository struct { db *drel.DB }
  func New(db *drel.DB) *Repository { return &Repository{db: db} }
  func (r *Repository) Create(ctx context.Context, req *todo.CreateRequest) (*todo.Schema, error) {
      m := models.NewTodo(req.GetTitle(), int(req.GetPriority()))
      if err := r.db.WithTx(ctx, func(ctx context.Context) error {
          r.db.Tx(ctx).Todos.Add(m); return nil
      }); err != nil { return nil, err }
      return toSchema(m), nil
  }
  // List/Read/Update/Delete 用 r.db.Todos.Where(...).All(ctx) / Find / WithTx
  ```
- 驗證：`go build ./internal/domain/todo/...`；單元測試 `repository_test.go` 用 dockertest Postgres（仿 `author/repository/postgres_test.go`）驗證 Create/List。

**T1.6 connect service + 註冊表**
- `internal/domain/todo/service/service.go`：實作 `todo_v1.TodoServiceHandler`，內部呼叫 usecase；提供 `Register(r chi.Router, svc *Service, opts ...connect.HandlerOption)`。
- `internal/server/domains.go`：
  ```go
  package server
  import "github.com/go-chi/chi/v5"
  type domainRegistration struct {
      register func(r chi.Router, opts ...connect.HandlerOption)
  }
  var Domains = []domainRegistration{ /* todo 於 P1 手填；P2 後由 CLI 追加 */ }
  ```
- `internal/server/initDomains.go`：改為 `for _, d := range Domains { d.register(s.router, connectOpts...) }`。
- 驗證：`go build ./...`；server 啟動後 `curl -X POST localhost:3080/go8.v1.TodoService/Create -d '{"title":"x","priority":1}'` 回傳 JSON todo。

**T1.7 前端 todo 頁（參考 app）**
- `web/package.json` 依賴：`solid-js`、`vite`、`vite-plugin-solid`、`@tanstack/solid-query`、`@tanstack/solid-router`、`@bufbuild/protobuf`、`@connectrpc/connect`、`@connectrpc/connect-web`。
- `web/src/lib/client.ts`：`createConnectTransport({ baseUrl: "/api" })` + `createPromiseClient(TodoService, transport)`。
- `web/src/features/todo/TodoList.tsx`：用 `useQuery` 呼叫 `client.list(...)` 渲染列表；`TodoForm.tsx` 用 `useMutation` 呼叫 `create`。
- `web/src/routes`：TanStack Router 掛 `/todos`。
- 驗證：`cd web && npm install && npm run build` 成功；`npm run dev` 後瀏覽 `/todos` 能看到由 connect 端點取回的資料（可用 Playwright 冒煙或手測）。

**T1.8 P1 驗收**
- 驗證序列：`task gen` → `go build ./...` → `task migrate` → 啟動 server → `curl` Create/List 回 JSON → `cd web && npm run build` 通過 → 前端列表頁渲染。
- 提交：`git commit` P1 全部（訊息 `feat: todo vertical slice (connect+drel+solidjs)`）。

### P2 — go8cli 程式碼生成

**T2.1 CLI 骨架**
- `cmd/go8cli/main.go`：用 `github.com/spf13/cobra` 或標準 `flag` 解析 `generate resource <name> [fields...]`。
- 驗證：`go run ./cmd/go8cli --help` 印出子命令。

**T2.2 內嵌模板**
- `cmd/go8cli/templates/` 放 P1 各檔的 `text/template` 版：`todo.proto.tmpl`、`model.go.tmpl`（DREL）、`usecase.go.tmpl`、`repository.go.tmpl`（DREL）、`service.go.tmpl`、`TodoList.tsx.tmpl` 等。以 `//go:embed templates` 載入。
- 驗證：`go build ./cmd/go8cli`。

**T2.3 generate resource 實作**
- 依 `<name>` 與 `[fields]` 渲染模板 → 寫入：
  - `api/go8/v1/<name>.proto`
  - `internal/db/models/<name>.go` + 觸發 `drel generate`
  - `internal/domain/<name>/{service,usecase,repository,model.go,request.go,resource.go}`
  - `web/src/features/<name>/*` + 觸發 `buf generate`
  - 在 `internal/server/domains.go` 的 `Domains` slice 追加一行（字串替換或結構化追加）
- 驗證：`go8cli generate resource widget` → `task gen` → `go build ./...` 通過 → server 啟動後 `curl` widget Create/List 可用 → `cd web && npm run build` 通過。
- 提交：`feat: go8cli generate resource`。

### P3 — 全面替換 author/book（clean cutover）

**T3.1 對 author 套用 CLI**
- `go8cli generate resource author`（欄位對齊現有 `author/model.go`：`name`、`books` 關聯等）。
- 比對新 `usecase` 與舊 `author/usecase` 業務邏輯（搜尋/快取），把 `searchRepo`/`cacheRedis`/`cacheLRU` 邏輯遷入新 usecase（或保留為新 usecase 的可選依賴）。
- 刪除 `internal/domain/author/handler/*`、`author/repository/postgres.go`(ent)、`redis.go`、`lru.go`、`search.go`(ent) 與對應 mock、`register.go`。
- 驗證：`go build ./...`；author connect 端點可用；舊 `/authors` REST 路徑不存在。

**T3.2 對 book 套用 CLI**
- 同上，處理 `book` ↔ `author` 關聯（FK）。
- 刪除 `internal/domain/book/handler/*`、`book/repository/postgres.go`(ent) 及 mock。
- 驗證：book connect 端點可用；`go build ./...` 通過。

**T3.3 清理 ent**
- 若 `ent/`、`ent/gen/` 與 `entgo.io/ent` 已無引用：`rm -rf ent ent/gen`；`go mod tidy` 移除 ent 依賴。
- 驗證：`go build ./...` 與 `go mod tidy` 乾淨；`grep -r "entgo.io/ent" internal` 無命中。
- 提交：`feat: migrate author/book to connect+drel, drop ent/chi-rest`。

### P4 — 前端參考 app 串接

**T4.1 TanStack Router/Query 串所有 resource**
- 在 `web/src/routes` 為 author/book/widget 加路由；`features/*` 加 list/form 頁，統一用 `lib/client.ts`。
- 驗證：`npm run build` 通過；`npm run dev` 各資源列表頁可渲染（Playwright 或手測冒煙）。
- 提交：`feat: web reference app wired to all connect resources`。

## 測試策略

- 後端：每 domain repository 單元測試用 dockertest Postgres（仿 `author/repository/postgres_test.go`）；usecase 測試用 mirip 產的 mock（`//go:generate mirip` 既有模式保留）。
- 前端：關鍵資料獲取用 `@tanstack/solid-query` 的 `QueryClient` 單元測試或元件測試（輕量，YAGNI：僅 list/create 一組）。
- 整合：P1/P2/P3 每階段以 `curl` connect JSON 端點作冒煙（非僅編譯）。
- 不寫：純轉發/mock 回聲/來源文字的重複測試。

## 提交節奏

- P0 完成：`chore: add buf/drel toolchain and config`
- P1 完成：`feat: todo vertical slice (connect+drel+solidjs)`
- P2 完成：`feat: go8cli generate resource`
- P3 完成：`feat: migrate author/book to connect+drel, drop ent/chi-rest`
- P4 完成：`feat: web reference app wired to all connect resources`

## 風險與因應（詳設計 §11）

- DREL migration：T1.3 spike 先確認 DDL 產出能力；無則手維 goose。
- DREL 邊界案例：repository 走介面，必要時可換實作。
- connect 與 chi 中介相容：connect handler 掛 chi 即可沿用 CORS/OTel/auth。
- CLI 模板漂移：模板源自 P1 切片，CLI 產出須通過相同編譯/冒煙。
