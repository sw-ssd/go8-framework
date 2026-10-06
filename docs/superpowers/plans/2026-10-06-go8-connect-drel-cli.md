# 實作計畫：Go8 CONNECT RPC + SolidJS 前端 + go8cli（放棄 DREL，保留 ent/sqlx，支援客製 templates/commands）

日期：2026-10-06
對應設計：docs/superpowers/specs/2026-10-06-go8-connect-drel-cli-design.md
修訂：放棄 DREL，資料層保留 ent/sqlx；go8cli 細分 model/create/read/update/delete/resource 子命令（互動+非互動）；go8cli 支援客製 templates 與 commands（檔案型，免重編譯）。
採用方案：A（垂直切片 todo 先行 → go8cli 模板化 → 對 author/book 套用達成全面替換 chi handler）

## Scope Check

設計含 3 子系統（buf/connect 後端、SolidJS 前端、go8cli）+ go8cli 擴充性，但彼此經「切片 → 模板」耦合，且已自拆為 P0–P4，每階段皆產出可獨立驗證的軟體。採**單一計畫、階段化任務**。

工程師假設：熟悉 Go 與前端，但不熟 buf/connect/TanStack/SolidJS/cobra。本計畫寫足工具安裝、指令、檔案職責、程式碼骨架、測試與驗證。

## 工具鏈安裝（P0 前置）

- **buf**：`go install github.com/bufbuild/buf/cmd/buf@latest`（或 `brew install bufbuild/buf/buf`）。驗證：`buf --version`。
- **connect-go**：`github.com/connectrpc/connect-go`、`google.golang.org/protobuf`（go get）。
- **ent**：既有 `go generate ./ent` 已可用。
- **前端**：Node 18+。`cd web && npm init -y` 後安裝依賴（見 §前端任務）。
- **Taskfile**：既有 `Taskfile.yml` 擴充 `gen` 目標串接 `buf generate`。
- **go8cli**：cobra（命令框架）+ promptui（互動提示）+ `text/template` + `//go:embed`（內嵌預設模板）。使用者客製模板/命令走檔案（見 T2.4/T2.5），不需重編譯。

## File Structure（本計畫觸及的檔案與職責）

新增：
- `buf.yaml` / `buf.gen.yaml` — buf 模組與生成設定。
- `api/go8/v1/todo.proto` — 參考 domain 的 proto。
- `gen/go8/v1/todo.pb.go`、`todo.connect.go` — buf 生成（納入版控）。
- `ent/schema/todo.go` — todo 的 ent schema（CLI model 子命令新增）。
- `internal/domain/todo/{service,usecase,repository,model.go,request.go,resource.go}` — 切片實作（repository 用 ent）。
- `internal/server/domains.go` — 已註冊 domain 清單（註冊表模式）。
- `web/` — SolidJS 參考 app。
- `cmd/go8cli/` — 程式碼生成 CLI（main.go + `templates/` 內嵌預設 + 模板解析/命令 manifest 邏輯）。
- `database/migrations/0000xx_todo.up.sql` / `.down.sql` — todo 的 goose migration。
- `.go8/templates/` — 使用者客製 .tmpl（覆寫內建或新增；可選，可不納入版控）。
- `.go8/commands/<cmd>.yaml` — 使用者客製命令 manifest（可選）。

修改：
- `internal/server/server.go` — 啟動時迭代 `domains.go` 掛載 connect handler（注入 ent client 等依賴）。
- `internal/server/initDomains.go` — 改為迭代 `domains.go` 清單。
- `go.mod` — 加 connect-go、protobuf、cobra、promptui。
- `Taskfile.yml` — 加 `gen` 目標（`buf generate`）。
- `ent/gen/` — 因新增 schema 重新生成（既有流程）。

P3 刪除（clean cutover，僅刪 chi handler，保留 ent/sqlx）：
- `internal/domain/author/handler/*` 與 `register.go`。
- `internal/domain/book/handler/*` 與 `register.go`。
- author/book 的 usecase/repository（ent/sqlx）保留不動。

## 任務（依階段）

### P0 — 工具鏈與 buf 配置

**T0.1 安裝 buf**
- 安裝 buf（見上）。`buf --version` 確認可用。

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

**T0.3 Taskfile 加 gen 目標**
- `Taskfile.yml` 加：
  ```yaml
  gen:
    desc: generate proto code
    cmds:
      - buf generate
  ```
- 驗證：`task gen` 可執行（縱使暫無 proto）。

### P1 — 垂直切片 `todo`（端到端，ent 資料層）

**T1.1 寫 todo.proto**
- `api/go8/v1/todo.proto`（見設計 §6 形狀），含 `Todo` message 與 `TodoService`（Create/Get/List/Update/Delete）。
- 驗證：`buf lint api/go8/v1/todo.proto` 通過；`buf generate` 產出 `gen/go8/v1/todo.pb.go`、`todo.connect.go`。

**T1.2 ent schema + 生成**
- `ent/schema/todo.go`（仿既有 schema）：
  ```go
  package schema
  import (
      "entgo.io/ent"
      "entgo.io/ent/schema/field"
  )
  type Todo struct { ent.Schema }
  func (Todo) Fields() []ent.Field {
      return []ent.Field{
          field.String("title"),
          field.Bool("done").Default(false),
          field.Int("priority").Default(0),
      }
  }
  ```
- 執行 `go generate ./ent`；確認 `ent/gen` 產出 `todo` 套件。
- 驗證：`go build ./ent/...` 通過。

**T1.3 todo migration**
- 由 ent 產出的 DDL（或手寫）新增 `database/migrations/0000xx_todo.up.sql` / `.down.sql`（goose 格式）：
  ```sql
  -- +up
  CREATE TABLE IF NOT EXISTS todos (
      id BIGSERIAL PRIMARY KEY,
      title TEXT NOT NULL,
      done BOOLEAN NOT NULL DEFAULT FALSE,
      priority INTEGER NOT NULL DEFAULT 0,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  -- +down
  DROP TABLE IF EXISTS todos;
  ```
- 驗證：`cmd/migrate` 對測試庫 up 成功；`\d todos` 存在。

**T1.4 usecase（沿用 author 模式）**
- `internal/domain/todo/usecase/usecase.go`：定義 `Todo` interface（Create/List/Read/Update/Delete），實作呼叫 repository。複製 `author/usecase` 結構，型別換成 todo 的 `Schema`/`Filter`/`*Request`。
- `internal/domain/todo/model.go`、`request.go`、`resource.go`：定義領域型別。
- 驗證：`go build ./internal/domain/todo/...`。

**T1.5 ent repository**
- `internal/domain/todo/repository/postgres.go`：仿 `author/repository/postgres.go`，使用 ent client：
  ```go
  type repository struct { ent *gen.Client }
  func New(ent *gen.Client) *repository { return &repository{ent: ent} }
  func (r *repository) Create(ctx context.Context, req *todo.CreateRequest) (*todo.Schema, error) {
      c, err := r.ent.Todo.Create().SetTitle(req.GetTitle()).SetPriority(int(req.GetPriority())).Save(ctx)
      if err != nil { return nil, err }
      return toSchema(c), nil
  }
  // List/Read/Update/Delete 用 r.ent.Todo.Query()/Get/Update/Delete
  ```
- 驗證：`go build ./internal/domain/todo/...`；單元測試 `postgres_test.go` 用 dockertest Postgres（仿 `author/repository/postgres_test.go`）驗證 Create/List。

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
- 驗證：`cd web && npm install && npm run build` 成功；`npm run dev` 後瀏覽 `/todos` 能看到由 connect 端點取回的資料（Playwright 或手測冒煙）。

**T1.8 P1 驗收**
- 驗證序列：`task gen` → `go generate ./ent` → `go build ./...` → `cmd/migrate` → 啟動 server → `curl` Create/List 回 JSON → `cd web && npm run build` 通過 → 前端列表頁渲染。
- 提交：`feat: todo vertical slice (connect+ent+solidjs)`。

### P2 — go8cli 程式碼生成（細分子命令，互動+非互動，支援客製）

**T2.1 CLI 骨架**
- `cmd/go8cli/main.go`：cobra root + `generate` 子命令；`generate` 下掛 `model`/`create`/`read`/`update`/`delete`/`resource`。
- 非互動：flags（`--field name:type`、結構化參數）。互動：缺 flag 且 `term.IsTerminal()` 時用 promptui 詢問；非 TTY 缺 flag 則 `os.Exit(2)` 報錯。
- 驗證：`go run ./cmd/go8cli generate --help` 印出 6 個子命令。

**T2.2 內嵌模板（按層拆分）**
- `cmd/go8cli/templates/` 放 P1 各檔的 `text/template` 版：`todo.proto.tmpl`、`ent_schema.tmpl`、`model.go.tmpl`、`usecase.go.tmpl`、`repository.go.tmpl`（ent）、`service.go.tmpl`、`TodoList.tsx.tmpl`、`TodoForm.tsx.tmpl` 等。以 `//go:embed templates` 載入。
- 驗證：`go build ./cmd/go8cli`。

**T2.3 generate resource / 各層子命令**
- `go8cli generate resource <name> [--field ...]` 依序呼叫 model+create+read+update+delete：
  - `model`：渲染 `ent/schema/<name>.go` + 觸發 `go generate ./ent` + 新增 goose migration + 產 `model.go`。
  - `create`/`read`/`update`/`delete`：分別渲染 proto 對應 rpc（首次建整個 service 骨架，之後增量追加）、service 方法、usecase 方法、repository 方法、前端頁/元件。
  - 在 `internal/server/domains.go` 的 `Domains` slice 追加一行（結構化替換）。
- 增量驗證：`go8cli generate resource widget` → `task gen` → `go build ./...` 通過 → server 啟動後 widget 端點可用 → `cd web && npm run build` 通過。
- 增量加層驗證：`go8cli generate delete widget`（假設 resource 已含 C/R/U）後 `go build ./...` 仍通過，且 widget service 含 Delete。
- 互動驗證：`go8cli generate resource thing`（無 flag，TTY）能依提示完成生成並編譯。
- 提交：`feat: go8cli generate (model/create/read/update/delete/resource, interactive+non-interactive)`。

**T2.4 客製 templates（檔案型）**
- 模板解析：內嵌預設 → 使用者目錄（`.go8/templates`，或 `--templates <dir>` / `GO8_TEMPLATES`）。同名使用者模板優先。
- 新增 `go8cli generate <name>` 通用渲染：對任意 `.tmpl`（內建或使用者目錄）以相同變數上下文（`Name`、`Fields` 等）渲染；使用者新增的 `.tmpl` 直接可被渲染（免重編譯）。
- 驗證：在 `.go8/templates/` 放 `hello.go.tmpl`，執行 `go8cli generate hello --name Foo` 產出預期檔；`--templates ''` 時忽略使用者目錄、只用內建。

**T2.5 客製 commands（manifest）**
- 讀取 `.go8/commands/<cmd>.yaml`（結構見設計 §9.1）：`name`/`description`/`prompts`/`steps`（`template`+`out`+`mode` 或 `shell`）。
- `go8cli <cmd>`（或 `go8cli run <cmd>`）依序執行 steps：渲染模板到 `out`（create/append），或執行 `shell`。`prompts` 在非互動時由 `--<name>` flag / 環境變數提供；非 TTY 缺值則報錯退出。
- 驗證：放 `.go8/commands/greeter.yaml`，執行 `go8cli greeter --label Hi` 產出 greeter 檔並（若有 shell 步）執行；非互動可於 CI 呼叫。
- 提交：`feat: go8cli extensibility (custom templates + command manifests)`。

### P3 — 全面替換 author/book 的 chi handler（保留 ent/sqlx）

**T3.1 對 author 套用 CLI**
- `go8cli generate resource author`（欄位對齊現有 `author/model.go`）。
- 比對新 usecase 與舊 `author/usecase` 業務邏輯（搜尋/快取），把 `searchRepo`/`cacheRedis`/`cacheLRU` 邏輯遷入新 usecase（或保留為新 usecase 的可選依賴）。
- 刪除 `internal/domain/author/handler/*` 與 `register.go`。
- 驗證：`go build ./...`；author connect 端點可用；舊 `/authors` chi REST 路徑不存在。

**T3.2 對 book 套用 CLI**
- 同上，處理 `book` ↔ `author` 關聯（FK）。
- 刪除 `internal/domain/book/handler/*` 與 `register.go`。
- 驗證：book connect 端點可用；`go build ./...` 通過。

**T3.3 確認 ent/sqlx 保留**
- 驗證：`grep -r "entgo.io/ent" internal` 仍有命中（author/book/todo repository）；`grep -r "chi.NewRouter\|router.Get" internal/domain` 資源型 domain 無命中。
- 提交：`feat: migrate author/book handlers to connect, keep ent/sqlx repos`。

### P4 — 前端參考 app 串接

**T4.1 TanStack Router/Query 串所有 resource**
- 在 `web/src/routes` 為 author/book/widget 加路由；`features/*` 加 list/form 頁，統一用 `lib/client.ts`。
- 驗證：`npm run build` 通過；`npm run dev` 各資源列表頁可渲染（Playwright 或手測冒煙）。
- 提交：`feat: web reference app wired to all connect resources`。

## 測試策略

- 後端：每 domain repository 單元測試用 dockertest Postgres（仿 `author/repository/postgres_test.go`）；usecase 測試用 mirip 產的 mock（`//go:generate mirip` 既有模式保留）。
- 前端：關鍵資料獲取用 `@tanstack/solid-query` 的 `QueryClient` 單元測試或元件測試（輕量，YAGNI：僅 list/create 一組）。
- 整合：P1/P2/P3 每階段以 `curl` connect JSON 端點作冒煙（非僅編譯）。
- go8cli：T2.4/T2.5 的客製模板與命令以實際 `go8cli` 執行驗證（產出可編譯/可執行）。
- 不寫：純轉發/mock 回聲/來源文字的重複測試。

## 提交節奏

- P0 完成：`chore: add buf toolchain and config`
- P1 完成：`feat: todo vertical slice (connect+ent+solidjs)`
- P2 完成：`feat: go8cli generate (model/create/read/update/delete/resource, interactive+non-interactive)` + `feat: go8cli extensibility (custom templates + command manifests)`
- P3 完成：`feat: migrate author/book handlers to connect, keep ent/sqlx repos`
- P4 完成：`feat: web reference app wired to all connect resources`

## 風險與因應

- ent 程式碼生成納入 CLI：`model` 子命令產 ent schema 並觸發 `go generate ./ent`；`task gen` 串接 `buf generate`。
- connect 與 chi 中介相容：connect handler 掛 chi 即可沿用 CORS/OTel/auth。
- CLI 模板漂移：模板源自 P1 切片，CLI 產出須通過相同編譯/冒煙。
- 互動模式 CI 行為：非 TTY 缺 flag/prompt 即報錯退出，確保 CI 可重現。
- 使用者模板覆寫內建：解析順序明確（使用者優先）；`--templates ''` 強制只用內建。
