# Go8 — CONNECT RPC + DREL ORM + SolidJS 前端 + go8cli 程式碼生成

日期：2026-10-06
狀態：已與使用者確認設計（採用方案 A：垂直切片先行 → CLI 模板化 → 套用舊 domain）

## 1. 背景與動機

`codeberg.org/gmhafiz/go8` 目前是一套 Go API starter kit，採用 **Chi router + sqlx + ent + Postgres + OpenTelemetry + Redis + goose** 的分層架構（handler → usecase → repository）。使用者希望將其升級為現代化全端 framework：

1. 後端採用 **CONNECT RPC** 及其生態（buf / connect-go / connect-web）。
2. 前端使用 **CONNECT RPC Web + SolidJS + TanStack（Query/Router）** 生態。
3. 新增 **DREL ORM**（`github.com/alternayte/drel`）作為資料層。
4. 建立 **go8cli** CRUD 程式碼模板（仿 `github.com/gobuffalo/cli` 的 resource 生成心智模型）。

已與使用者確認的三項閘門決策：

- **DREL** = `github.com/alternayte/drel`（基於程式碼生成的 Postgres/SQLite ORM）。
- **CONNECT 整合方式** = 全面替換：Chi REST 退為 mux + infra 中介；主要 transport 改為 CONNECT（自帶 HTTP/JSON + gRPC 雙協議）。
- **前端交付** = repo 內 `web/` 放參考 SolidJS 應用（connect-web + TanStack Query/Router），`go8cli` 從模板生成 CRUD 頁面/元件，並同時生成 Go 程式模板。

## 2. 目標

- 在既有 go8 骨架上建立可端到端運作的 CONNECT RPC 後端（proto 定義 → connect service → usecase → DREL repository）。
- 以 DREL 取代現有 ent/sqlx 資料層（於 CLI 套用後移除）。
- 提供 `web/` 參考前端：SolidJS + TanStack Query/Router，透過 connect-web client 呼叫同一組端點。
- 提供 `go8cli`：非互動式（flag 驅動）的 Go 程式碼生成器，一口氣產出前後端 CRUD 骨架。
- 達成「全面替換」：現有 `author`/`book` 等 domain 經 CLI 重新生成後，刪除 chi handler 與 ent/sqlx 程式碼（clean cutover）。

## 3. 非目標（本輪不做）

- 不替換 Redis / OpenTelemetry / CORS / 認證（argon2id + scs）等 infra 能力。
- 不實作多租戶、SaaS 計費、檔案儲存等未經要求的子系統。
- 不在本輪重寫 `authentication` domain（保留 chi 路徑，後續可獨立轉 connect）。
- 不支援 DREL 目前未涵蓋的資料庫（僅 Postgres；SQLite 為可選延伸，非本輪驗證重點）。

## 4. 採用方案

**方案 A — 垂直切片先行 → CLI 模板化 → 套用舊 domain（採用）**

1. 先做一個**全新參考 domain**（如 `todo`）跑通整條龍：`proto → connect service → usecase → DREL repo → 前端頁面 → migration`。
2. 把該切片固化為 `go8cli` 的內嵌模板。
3. 用 CLI 對 `author`/`book` 生成，達成全面替換並刪除舊程式碼。

理由：單一切面先證明可行，風險集中可回滾；CLI 模板即規格，保證 N 個 domain 一致；不重寫兩次。

（對照方案 B「大爆炸全改寫」已捨棄：改動面大、CLI 與手寫易漂移。）

## 5. 倉庫佈局（monorepo）

```
go8-framework/
├── api/go8/v1/*.proto            # protobuf 源（每 domain 一檔）
├── buf.yaml                       # buf 模組與 lint/breaking 設定
├── buf.gen.yaml                   # 生成 go + connect-go + connect-web(ts) + openapiv2
├── gen/go8/v1/                    # buf 生成的 Go stub（*.pb.go, *.connect.go）
├── internal/
│   ├── db/
│   │   ├── models/                # DREL 模型（embed drel.Model[T] + db: tag）
│   │   └── gen/                   # drel generate 產出（DB 聚合 + query builder）
│   ├── domain/<d>/
│   │   ├── service/               # connect service impl（= 舊 handler 角色）
│   │   ├── usecase/               # 業務邏輯（既有，精簡保留）
│   │   ├── repository/            # DREL 實作（取代 ent/sqlx）
│   │   ├── model.go               # domain 型別（Schema/Filter/Request）
│   │   ├── request.go
│   │   └── resource.go
│   └── server/                    # chi 作 mux + infra 中介（CORS/OTel/auth/health）
├── web/                           # SolidJS 參考 app（獨立 npm 專案，不在 Go module）
│   ├── package.json
│   └── src/
│       ├── gen/                   # buf 生成的 TS service/message client
│       ├── lib/client.ts          # createPromiseClient 工廠
│       ├── routes/                # TanStack Router 路由
│       └── features/<d>/          # list/create/edit/delete 頁與元件
├── cmd/
│   ├── go8/                       # server 入口（保留）
│   ├── go8cli/                    # 新增：程式碼生成 CLI
│   ├── migrate/                   # 保留（goose）
│   └── seed/                      # 保留
└── database/migrations/           # goose（保留）
```

- `web/` 為獨立 npm workspace，buf 產出的 TS 落至 `web/src/gen`。
- `gen/go8/v1/` 為 buf 生成的 Go 產物，納入版本控制（與 proto 同步）。

## 6. 後端 transport：protobuf/buf + CONNECT

- `.proto` 定義 service：`Create` / `Get` / `List`（含分頁與過濾）/ `Update` / `Delete`。
- `buf.gen.yaml` 生成：
  - Go messages（`protoc-gen-go`）
  - connect-go service stub（`protoc-gen-connect-go`，產生 `XServiceHandler` 介面與 `NewXServiceHandler`）
  - connect-web TS client（`protoc-gen-connect-web` + `protoc-gen-es`）
  - openapiv2（沿用現有 swagger 習慣，選用）
- **service impl** 位於 `internal/domain/<d>/service/service.go`，實作生成的 `XServiceHandler`，內部呼叫既有 `usecase`。
- **掛載**：connect handler 即 `http.Handler`，掛到 chi router：
  ```go
  path, h := bookv1.NewBookServiceHandler(svc, opts)
  s.router.Handle(path, h)
  ```
  chi 退為 mux + 全域中介（CORS / OTel / auth）；`/health`、`/version`、`/metrics`、`/swagger` 仍走 chi。CONNECT 自帶 HTTP/JSON + gRPC 雙協議。
- 現有 `internal/domain/*/handler`（chi 版）與 `register.go` 於 CLI 套用後刪除。

## 7. 資料層：DREL ORM

- 模型集中 `internal/db/models/`：
  ```go
  type Book struct {
      drel.Model[int]
      Title  string `db:"title"`
      AuthorID int   `db:"author_id"`
  }
  func NewBook(title string, authorID int) *Book { return &Book{Title: title, AuthorID: authorID} }
  ```
  每實體一檔；`drel.Model[T]` 提供 PK 與追蹤基礎。
- `drel generate` → `internal/db/gen/`：`DB` 聚合所有集合、`WithTx`、`Tx(ctx)`、型別安全 `Where().OrderBy().All`、snapshot 變更追蹤（僅更新異動欄位）。
- server 開 `drel.DB`（`db.Open(dsn)`）注入 repository；repository 使用：
  ```go
  books, err := database.Books.Where(models.Books.Done.IsFalse()).OrderBy(models.Books.Priority.Asc()).All(ctx)
  ```
  交易經 `database.WithTx(ctx, func(ctx){ database.Tx(ctx).Books.Add(...) })` 走 context。
- **migration 策略（實作階段 spike 確認）**：
  - 預設：DREL 由模型產 DDL，用 `drel` CLI 把 DDL 落進 `database/migrations/*.sql` 給 goose 跑（現有 `database/migrate.go` 保留）。
  - 若 DREL 不支援自動產 DDL：則手維 goose migration，DREL 模型作為 runtime 型別安全層。
  - 此決策不阻塞整體設計；P1 切片時一併確認。
- usecase 層的 redis/lru 快取（`author` 有）保留——DREL 不取代快取。
- **風險**：DREL 為較新 ORM，邊界案例可能不足。緩解：repository 仍走介面（如既有 `Author` interface），必要時可換實作；但本輪按使用者要求全面採用。

## 8. 前端 `web/`：SolidJS + TanStack + connect-web

- Vite + SolidJS + TypeScript；依賴：
  - `@tanstack/solid-query`（資料獲取/快取）
  - `@tanstack/solid-router`（路由）
  - `@connectrpc/connect`、`@connectrpc/connect-web`、`@bufbuild/protobuf`
- `src/gen/` = buf 生成的 TS service/message client。
- `src/lib/client.ts`：
  ```ts
  import { createPromiseClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import { BookService } from "../gen/book/v1/book_connect";
  const transport = createConnectTransport({ baseUrl: "/api" });
  export const bookClient = createPromiseClient(BookService, transport);
  ```
- `src/features/<d>/`：list / create / edit / delete 頁與元件，使用 `useQuery` / `useMutation`（TanStack Query）+ `src/routes`（TanStack Router）。
- CORS 已配置，前端經同一組 connect 端點（HTTP/JSON）取數。

## 9. `go8cli`：Go 程式碼生成（仿 buffalo/cli）

- `cmd/go8cli`：以 `text/template` + `//go:embed` 內嵌模板（與 §5/§6/§7/§8 切片同構）。
- 命令（非互動式為主，flag 驅動）：
  - `go8cli generate resource <name> [fields...]`
    - `api/go8/v1/<name>.proto`
    - `internal/db/models/<name>.go`（DREL 模型）+ 重跑 `drel generate`
    - `internal/domain/<name>/{service,usecase,repository,model.go,request.go,resource.go}`
    - `web/src/features/<name>/*`（TanStack 頁/元件）+ 重跑 buf 產 TS
    - 採用**註冊表模式**避免 CLI 改寫 `initDomains.go`：各 domain 於 `service` 套件暴露 `Register(r chi.Router, svc *XService, opts ...connect.HandlerOption)`，`internal/server/domains.go` 維護已註冊 domain 清單，server 啟動時迭代掛載；CLI 僅在該 slice 追加一行，不改寫 `initDomains.go` 本體。
  - `go8cli new <module>`（選用，scaffold 新 app）
- 設計原則：模板即規格。CLI 產出的程式碼必須可 `go build ./...` 且端點可用；產出結構與 P1 切片完全一致，確保 `author`/`book` 經 CLI 重新生成後與手寫切片同構。

## 10. 階段與驗證

- **P0 工具鏈**：buf、drel、web（npm/vite）就位；`buf.gen.yaml` 與 `buf.yaml`；`drel generate` 流程串接。
- **P1 垂直切片**：全新 `todo` domain 跑通 proto / connect service / usecase / DREL repo / migration / 前端頁。
  - 驗證：`buf generate` 乾淨 → `drel generate` 乾淨 → `go build ./...` → server 啟動 → `curl` connect JSON 端點回傳資料 → `vite build` 成功 → 列表頁冒煙渲染。
- **P2 `go8cli`**：把 P1 固化为模板；`go8cli generate resource widget` 驗證可編譯、端點可用、`vite build` 通過。
- **P3 全面替換**：用 CLI 對 `author`/`book` 生成，刪除 chi handler + ent/sqlx 相關程式碼（clean cutover，無 shim）。
- **P4 前端參考 app**：TanStack Query/Router 串接所有 generated resource。

每階段驗證準則：產生器輸出可編譯、server 啟動、connect JSON 端點可呼叫、前端建置通過、舊 REST 路徑在 P3 後不再存在。

## 11. 風險與緩解

| 風險 | 緩解 |
|------|------|
| DREL 較新，邊界案例不足 | repository 走介面；migration spike 先確認 DDL 產出能力 |
| DREL 不支援自動 migration | 退回手維 goose，DREL 僅作 runtime 層 |
| connect 與既有 chi 中介（CORS/OTel/auth）相容性 | connect handler 為 http.Handler，掛 chi 即可沿用全域中介 |
| CLI 模板與手寫切片漂移 | 模板直接源自 P1 切片；CLI 產出需通過相同編譯/冒煙驗證 |
| 全面替換波及 `authentication` 等 domain | 本輪僅轉 `author`/`book` 等資源型 domain；auth 保留 chi 路徑 |

## 12. 驗收標準（Definition of Done）

- `go8cli generate resource <name>` 一口氣產出前後端 CRUD 骨架，且 `go build ./...` 與 `vite build` 通過。
- 參考 `todo` 切片與 `author`/`book`（經 CLI 生成）皆以 CONNECT 暴露端點，舊 chi REST handler 已移除。
- 資料層統一走 DREL；ent/sqlx 在資源型 domain 不再被引用。
- 前端 `web/` 參考 app 能以 TanStack Query/Router 經 connect-web 完成 list/create/edit/delete。
- `buf generate` 與 `drel generate` 皆納入 `Taskfile.yml`，CI 可重現。
