# Go8 — CONNECT RPC + SolidJS 前端 + go8cli 程式碼生成

日期：2026-10-06
狀態：已與使用者確認設計（採用方案 A：垂直切片先行 → CLI 模板化 → 套用舊 domain）
修訂 1：2026-10-06 放棄 DREL ORM，資料層保留既有 ent/sqlx；go8cli 細分為 model/create/read/update/delete/resource 子命令，互動與非互動皆支援。
修訂 2：2026-10-06 go8cli 支援客製——使用者可新增 templates 與 commands（檔案型，免重編譯）。

## 1. 背景與動機

`codeberg.org/gmhafiz/go8` 目前是一套 Go API starter kit，採用 **Chi router + sqlx + ent + Postgres + OpenTelemetry + Redis + goose** 的分層架構（handler → usecase → repository）。使用者希望將其升級為現代化全端 framework：

1. 後端採用 **CONNECT RPC** 及其生態（buf / connect-go / connect-web）。
2. 前端使用 **CONNECT RPC Web + SolidJS + TanStack（Query/Router）** 生態。
3. 建立 **go8cli** CRUD 程式碼模板（仿 `github.com/gobuffalo/cli` 的 resource 生成心智模型），支援互動與非互動，細分為 model/create/read/update/delete/resource 子命令，且允許使用者客製新增 templates 與 commands。

**資料層決策**：原規劃的 DREL ORM 已放棄，資料層**保留既有 ent/sqlx**，不引入新 ORM。本輪只把 transport 從 chi REST 換成 CONNECT，repository 層（ent/sqlx）維持不變。

## 2. 目標

- 在既有 go8 骨架上建立可端到端運作的 CONNECT RPC 後端（proto 定義 → connect service → usecase → repository(ent/sqlx)）。
- 提供 `web/` 參考前端：SolidJS + TanStack Query/Router，透過 connect-web client 呼叫同一組端點。
- 提供 `go8cli`：仿 buffalo/cli 的程式碼生成器，細分 model/create/read/update/delete/resource 子命令；互動（終端提示）與非互動（flag 驅動）皆支援；並允許使用者以檔案方式客製 templates 與 commands（免重編譯）。
- 達成「全面替換 chi REST」：現有 author/book 等 domain 經過 connect 化後，刪除 chi handler（ent/sqlx repository 保留）。

## 3. 非目標（本輪不做）

- 不引入新的 ORM（DREL 已放棄；不換 gorm/bun/sqlc）。
- 不替換 Redis / OpenTelemetry / CORS / 認證（argon2id + scs）等 infra 能力。
- 不實作多租戶、SaaS 計費、檔案儲存等未經要求的子系統。
- 不在本輪重寫 authentication domain（保留 chi 路徑，後續可獨立轉 connect）。
- 不實作 Go plugin / 外部腳本式的命令擴充（採檔案型 manifest，見 §9 擴充性）。

## 4. 採用方案

**方案 A — 垂直切片先行 → CLI 模板化 → 套用舊 domain（採用）**

1. 先做一個**全新參考 domain**（如 todo）跑通整條龍：proto → connect service → usecase → repository(ent) → 前端頁面。
2. 把該切片固化為 go8cli 的內嵌模板（細分各層生成器）。
3. 用 CLI 對 author/book 生成 connect service/usecase/前端，達成全面替換 chi handler。

理由：單一切面先證明可行，風險集中可回滾；CLI 模板即規格，保證 N 個 domain 一致；不重寫兩次。

（對照方案 B「大爆炸全改寫」已捨棄：改動面大、CLI 與手寫易漂移。）

## 5. 倉庫佈局（monorepo）

```
go8-framework/
├── api/go8/v1/*.proto            # protobuf 源（每 domain 一檔）
├── buf.yaml                       # buf 模組與 lint/breaking 設定
├── buf.gen.yaml                   # 生成 go + connect-go + connect-web(ts) + openapiv2
├── gen/go8/v1/                    # buf 生成的 Go stub（*.pb.go, *.connect.go）
├── ent/
│   ├── schema/                    # ent schema（既有；CLI 的 model 子命令在此新增）
│   └── gen/                       # ent 程式碼生成（既有）
├── internal/
│   ├── domain/<d>/
│   │   ├── service/               # connect service impl（= 舊 handler 角色）
│   │   ├── usecase/               # 業務邏輯（既有，精簡保留）
│   │   ├── repository/            # ent/sqlx 實作（既有，不動）
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
├── .go8/                          # go8cli 擴充（使用者客製；可選，不強制納入版控）
│   ├── templates/                 # 使用者 .tmpl（覆寫內建或新增；go8cli generate <name> 渲染）
│   └── commands/                  # 使用者命令 manifest（<cmd>.yaml；go8cli <cmd> 執行）
└── database/migrations/           # goose（保留；CLI model 子命令可新增 migration）
```

- `web/` 為獨立 npm workspace，buf 產出的 TS 落至 `web/src/gen`。
- `gen/go8/v1/` 為 buf 生成的 Go 產物，納入版本控制（與 proto 同步）。
- `ent/` 維持既有程式碼生成流程（`go generate ./ent`）；CLI 的 model 子命令在 `ent/schema/` 新增 schema 並觸發生成。
- `.go8/` 為 go8cli 擴充目錄（見 §9 擴充性）：模板解析順序為「內嵌預設 → 使用者 `.go8/templates`（或 `--templates` 指定）」；使用者命令 manifest 於 `.go8/commands/<cmd>.yaml`。

## 6. 後端 transport：protobuf/buf + CONNECT（全面替換 chi REST）

- `.proto` 定義 service：Create / Get / List（含分頁與過濾）/ Update / Delete。
- `buf.gen.yaml` 生成：
  - Go messages（protoc-gen-go）
  - connect-go service stub（protoc-gen-connect-go，產生 XServiceHandler 介面與 NewXServiceHandler）
  - connect-web TS client（protoc-gen-connect-web + protoc-gen-es）
  - openapiv2（沿用現有 swagger 習慣，選用）
- **service impl** 位於 `internal/domain/<d>/service/service.go`，實作生成的 XServiceHandler，內部呼叫既有 usecase。
- **掛載**：connect handler 即 http.Handler，掛到 chi router：
  ```go
  path, h := bookv1.NewBookServiceHandler(svc, opts)
  s.router.Handle(path, h)
  ```
  chi 退為 mux + 全域中介（CORS / OTel / auth）；`/health`、`/version`、`/metrics`、`/swagger` 仍走 chi。CONNECT 自帶 HTTP/JSON + gRPC 雙協議。
- 現有 `internal/domain/*/handler`（chi 版）於 CLI 套用後刪除。
- **資料層不變**：usecase 與 repository（ent/sqlx）沿用既有實作，不重建。

## 7. 資料層：保留 ent/sqlx（DREL 已放棄）

- 維持現狀：domain repository 使用 `ent/gen` 的 `*gen.Client`（如 author/repository/postgres.go），或 sqlx（如 health）。
- 既有的 `ent/generate.go`、`go generate ./ent`、`database/migrate.go`（goose）與 `database/migrations/` 全部保留。
- CLI 的 model 子命令在 `ent/schema/` 新增 ent schema，並觸發 `go generate ./ent`；migration 由使用者依 ent 產出的 DDL 手維 goose（或沿用現有 migration 慣例）。
- 不改 repository 介面；connect service 只取代上層 handler。

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
- `src/features/<d>/`：list / create / edit / delete 頁與元件，使用 useQuery / useMutation（TanStack Query）+ `src/routes`（TanStack Router）。
- CORS 已配置，前端經同一組 connect 端點（HTTP/JSON）取數。

## 9. `go8cli`：Go 程式碼生成（仿 buffalo/cli，細分子命令）

- `cmd/go8cli`：以 `text/template` + `//go:embed` 內嵌模板（與 §5/§6/§7/§8 切片同構）。
- **互動與非互動**：
  - 非互動（flag 驅動）：`go8cli generate resource book --field title:string --field priority:int`。
  - 互動（終端提示）：當必要 flag 省略且為 TTY 時，依序詢問名稱、欄位（名稱:型別）、是否產前端等；非 TTY 時缺少 flag 則報錯（可被 CI 呼叫）。
- **內建子命令**（每個皆支援互動/非互動；可獨立對既有 resource 增量添加一層）：
  - `go8cli generate model <name> [fields...]`
    - 在 `ent/schema/<name>.go` 新增 ent schema
    - 觸發 `go generate ./ent`
    - 在 `database/migrations/` 新增 goose migration（DDL 來源：ent 產出或手寫）
    - 產出 `internal/domain/<name>/model.go`（Schema/Filter/Request 領域型別）
  - `go8cli generate create <name>`：proto 的 Create 部分 + service.Create + usecase.Create + repository.Create（ent）+ 前端 create 表單
  - `go8cli generate read <name>`：Get + List + usecase + repo + 前端 list 頁
  - `go8cli generate update <name>`：Update + usecase + repo + 前端 edit 頁
  - `go8cli generate delete <name>`：Delete + usecase + repo + 前端 delete 動作
  - `go8cli generate resource <name> [fields...]`：依序執行 model + create + read + update + delete（一鍵全產 C+R+U+D+model+service+前端）
- **註冊表模式**：各 domain 於 service 套件暴露 `Register(r chi.Router, svc *XService, opts ...connect.HandlerOption)`，`internal/server/domains.go` 維護已註冊 domain 清單，server 啟動時迭代掛載；CLI 僅在該 slice 追加一行，不改寫 `initDomains.go` 本體。
- 設計原則：模板即規格。CLI 產出的程式碼必須可 `go build ./...` 且端點可用；產出結構與 P1 切片完全一致，確保 author/book 經 CLI 重新生成後與手寫切片同構。

### 9.1 擴充性：客製 templates 與 commands（檔案型，免重編譯）

- **模板解析順序**：內嵌預設模板 → 使用者模板目錄（`.go8/templates`，或 `--templates <dir>` / `GO8_TEMPLATES` 環境變數）。同名使用者模板覆寫內建；使用者新增的 `.tmpl` 檔案可被 `go8cli generate <name>` 直接渲染（`<name>` 為檔名去副檔名），共用與內建相同的變數上下文（`Name`、`Fields` 等）。
- **客製命令**：使用者在 `.go8/commands/<cmd>.yaml` 宣告一個命令 manifest，結構：
  ```yaml
  name: widget
  description: scaffold a widget bundle
  prompts:                       # 互動欄位；非互動時由 --<name> flag / 環境變數提供
    - name: label
      prompt: Widget display label
  steps:                         # 依序執行
    - template: widget/go.go.tmpl
      out: internal/domain/{{.Name}}/widget.go
      mode: create               # create | append
    - template: widget/page.tsx.tmpl
      out: web/src/features/{{.Name}}/Widget.tsx
      mode: create
    - shell: task gen            # 可選：執行外部指令
  ```
  - 執行：`go8cli <cmd>`（或 `go8cli run <cmd>`）讀取 manifest 並逐一執行 steps；互動與非互動行為與內建子命令一致（非 TTY 缺 prompt 值則報錯）。
- 此機制讓使用者擴充 templates 與 commands 不需修改或重編譯 go8cli 原始碼。

## 10. 階段與驗證

- **P0 工具鏈**：buf、web（npm/vite）就位；`buf.gen.yaml` 與 `buf.yaml`；`Taskfile.yml` 加 `gen`（=`buf generate`）。
- **P1 垂直切片**：全新 todo domain 跑通 proto / connect service / usecase / repository(ent) / 前端頁。
  - 驗證：`buf generate` 乾淨 → `go generate ./ent` 乾淨 → `go build ./...` → server 啟動 → `curl` connect JSON 端點回傳資料 → `vite build` 成功 → 列表頁冒煙渲染。
- **P2 go8cli**：把 P1 固化为模板；實作 model/create/read/update/delete/resource 子命令（互動+非互動）；並實作 §9.1 的模板目錄解析與命令 manifest 執行（T2.4/T2.5）。`go8cli generate resource widget` 驗證可編譯、端點可用、`vite build` 通過；`go8cli generate delete widget`（增量）驗證可加層。
- **P3 全面替換 chi handler**：用 CLI 對 author/book 生成 connect service/usecase/前端（repository/ent 保留），刪除 chi handler + register.go。不移除 ent/sqlx。
  - 驗證：`go build ./...`；author/book connect 端點可用；舊 `/authors`、`/books` chi REST 路徑不存在；`grep -r "chi.NewRouter\|router.Get" internal/domain` 無資源型 domain 命中。
- **P4 前端參考 app**：TanStack Query/Router 串接所有 generated resource。

每階段驗證準則：產生器輸出可編譯、server 啟動、connect JSON 端點可呼叫、前端建置通過、舊 REST 路徑在 P3 後不再存在。

## 11. 風險與緩解

| 風險 | 緩解 |
|------|------|
| connect 與既有 chi 中介（CORS/OTel/auth）相容性 | connect handler 為 http.Handler，掛 chi 即可沿用全域中介 |
| ent 程式碼生成納入 CLI 流程的複雜度 | model 子命令產 ent schema 並觸發 `go generate ./ent`；`task gen` 串接 `buf generate` |
| CLI 模板與手寫切片漂移 | 模板直接源自 P1 切片；CLI 產出需通過相同編譯/冒煙驗證 |
| 全面替換波及 authentication 等 domain | 本輪僅轉 author/book 等資源型 domain；auth 保留 chi 路徑 |
| 互動模式在非 TTY（CI）下行為 | 非 TTY 缺少必要 flag/prompt 即報錯退出，確保 CI 可重現 |
| 使用者模板覆寫內建導致行為偏移 | 模板解析順序明確（使用者優先）；可用 `--templates ''` 強制只用內建 |

## 12. 驗收標準（Definition of Done）

- `go8cli generate resource <name>` 一口氣產出前後端 CRUD 骨架，且 `go build ./...` 與 `vite build` 通過；`go8cli generate {model,create,read,update,delete}` 各子命令可獨立/增量產出並通過編譯。
- 互動與非互動兩種模式皆可用（非互動可於 CI 呼叫）。
- 使用者可在 `.go8/templates` 放自訂 `.tmpl` 並以 `go8cli generate <name>` 渲染；可在 `.go8/commands/<cmd>.yaml` 定義客製命令並以 `go8cli <cmd>` 執行，且互動/非互動皆可用。
- 參考 todo 切片與 author/book（經 CLI 生成）皆以 CONNECT 暴露端點，舊 chi REST handler 已移除；ent/sqlx 資料層保留不變。
- 前端 web/ 參考 app 能以 TanStack Query/Router 經 connect-web 完成 list/create/edit/delete。
- `buf generate` 納入 `Taskfile.yml`，CI 可重現。
