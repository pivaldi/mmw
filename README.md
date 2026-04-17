# MMW — Modular Monolith Worskpace (POC)

A proof-of-concept of the [Go Modular Monolith White Paper](https://github.com/pivaldi/go-modular-monolith-white-paper)
that can evolve into a distributed system without rewriting application code.
All modules run in a single process in development; extracting a module to its own deployment requires only a
composition-root change.

Read the [Technical Implementation Reference](./TECHNICAL-IMPLEMENTATION-REFERENCE.md) for understanding in-deep the technical implementation.

---

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────┐
│  cmd/mmw/main.go  — composition root                │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌────────────────┐    │
│  │   auth   │  │   todo   │  │ notifications  │    │
│  │ :8091    │  │ :8090    │  │  (subscriber)  │    │
│  └──────────┘  └──────────┘  └────────────────┘    │
│       │              │                              │
│       └──── shared PostgreSQL :5435 ────────────────│
│       └──── in-process Watermill bus ───────────────│
└─────────────────────────────────────────────────────┘
```

| Module | Port | Description |
|---|---|---|
| `modules/auth` | 8091 | JWT auth — register, login, token validation |
| `modules/todo` | 8090 | Todo CRUD — protected by auth middleware |
| `modules/notifications` | — | Event consumer — logs / forwards to RocketChat |

---

## Prerequisites

All dependencies will be installed by the the script `./configure`.

| Tool | Purpose |
|---|---|
| **Go ≥ 1.26** | Build and run |
| **Docker** | PostgreSQL container |
| **mise** | Task runner (modes 2 & 3) |

Optional for the Angular frontend:

| Tool | Install |
|---|---|
| **Node ≥ 18 / npm** | [nodejs.org](https://nodejs.org) |
| **Angular CLI** | `npm install -g @angular/cli` |

---

## Environment Variables

All three run modes need a minimum set of environment variables. The simplest
way is to copy the example file:

```bash
cp .envrc.example .envrc   # sets APP_ENV=development
```

The remaining variables have safe defaults already baked into `mise.toml` for
development (see below). If you are **not** using mise, export them manually:

```bash
export APP_ENV=development
export APP_NAME=mmw
export DB_PASSWORD=postgres
export JWT_SECRET=fake-secret   # change in production
```

---

## Database

All three modes share the same PostgreSQL container:

```bash
# Start
docker-compose -f deployments/docker-compose.yml up postgres -d # or mise run db:up

# Stop
docker-compose -f deployments/docker-compose.yml down postgres # or mise run db:down

# Connection details (same as the embedded default.toml)
host=localhost  port=5435  db=mmw  user=postgres  password=postgres
```

---

## Run Mode 1 — Minimal, no configure (in-process)

**When to use:** you just cloned the repo and want to run the monolith immediately,
without installing mise, stow, or any other tooling. The root `go.mod` already
lists every module (`mmw-auth`, `mmw-todo`, `mmw-notifications`, `mmw-contracts`,
`piprim/mmw`) as pinned dependencies, so the Go toolchain resolves them without
a workspace.

### 1a — Set env vars

```bash
cp .envrc.example .envrc
source .envrc
export APP_NAME=mmw DB_PASSWORD=postgres JWT_SECRET=fake-secret
```

### 1b — Start the database

```bash
docker-compose -f deployments/docker-compose.yml up postgres -d
```

### 1c — Migrate both modules

Run migrations directly with `go run` from each module directory. Go resolves
the modules from `go.mod` when there is no `go.work` in scope.

```bash
# Ignore the workspace (or remove go.work if you prefer)
export GOWORK=off

# Migrate all module at once
DB_PASSWORD=postgres APP_ENV=development go run cmd/migrate/main.go
```

### 1d — Run the monolith

```bash
GOWORK=off DB_PASSWORD=postgres APP_ENV=development JWT_SECRET="fake-secret" go run ./cmd/mmw
```

All three modules (auth :8091, todo :8090, notifications) start in the same
process. Module communication is entirely in-process — zero network overhead.

The front-end is embedded and serve by the monolith, so opening http://localhost:8090/login in a web browser works as is.

---

## Run Mode 2 — Full developer setup (in-process)

**When to use:** day-to-day development. `./configure` installs mise, stow, and
all tooling; `go.work` enables local module resolution so edits to any module
are immediately reflected without publishing.

### 2a — One-time setup

```bash
./configure        # installs mise, stow, buf, golangci-lint, sets GOPRIVATE, syncs go.work
```

`configure` does, in order:

1. Updates git submodules
2. Installs GNU Stow (if absent)
3. Copies `.envrc.example` → `.envrc` (if needed) and sources it
4. Installs mise and all declared tools (`go`, `golangci-lint`, `buf`, `zellij`, …)
5. Runs `mise install` + `mise run setup` (sets `GOPRIVATE`, syncs `go.work`, tidies all modules)
6. Installs the pre-commit hook (`arch:check` + `buf:lint` on every commit)
7. Stows common scripts into every `modules/*/scripts/` directory via symlinks

### 2b — Start and migrate the database

```bash
# Start PostgreSQL
mise run db:up

# Migrate both modules in one command
mise run db:migrate
```

Individual migration targets are also available inside each module directory:

```bash
cd modules/auth && mise run db:migrate:up
cd modules/todo && mise run db:migrate:up
```

### 2c — Run the monolith

```bash
# Plain run
mise run run

# Or with hot-reload (requires air, installed by mise)
mise run run:dev
```

Equivalent without mise:

```bash
go run -tags no_clickhouse,no_mysql,no_mssql ./cmd/mmw
```

The monolith starts all modules in a single process. The `go.work` file ensures
that local module directories (`modules/auth`, `modules/todo`, …) take
precedence over the pinned versions in `go.mod`.

The front-end SPA must be started manually in day-to-day development:
```bash
cd modules/todo/web/todoapp/
mise run start # or the alias "mise run run"
```

### 2d — Useful daily tasks

```bash
mise run test              # unit tests across all modules
mise run test:contract     # contract tests (require postgres)
mise run test:integration  # integration tests (require postgres)
mise run lint              # golangci-lint on all modules
mise run arch:check        # architectural boundary validation
mise run buf:generate      # regenerate proto stubs + application contracts
mise run buf:lint          # lint proto files
```

---

## Run Mode 3 — Distributed processes (Connect RPC)

**When to use:** validating that a module can be extracted from the monolith and
run as a standalone service. Auth and todo each run in their own process and
communicate over HTTP/2 via Connect RPC — exactly as they would in a
microservices deployment.

> **Prerequisite:** the database must already be up and both schemas migrated
> (Mode 2, steps 2b — or Mode 1, steps 1b–1c).

The key difference from Modes 1 & 2 is in `modules/todo/cmd/todo/main.go`:
instead of receiving `authModule.PrivateService()` (an in-process call), the
todo module creates a Connect HTTP client that calls the auth service over the
network:

```go
// modules/todo/cmd/todo/main.go (already implemented)
authHttpClient := authv1connect.NewAuthPrivateServiceClient(
    &http.Client{},
    todoConf.AuthServer.URL("", nil), // http://localhost:8091
)
todoModule, _ := todo.New(todo.Infrastructure{
    AuthSvc: defauth.NewPrivateHTTPClient(authHttpClient),
    // …
})
```

No application code changes — only the composition root differs.

### 3a — Terminal 1: start the auth service

```bash
cd modules/auth
go run ./cmd/auth          # listens on http://localhost:8091
```

### 3b — Terminal 2: start the todo service

```bash
cd modules/todo
go run ./cmd/todo          # listens on http://localhost:8090
                           # calls auth at http://localhost:8091 for token validation
```

### 3c — Service endpoints

| Service | Address | Role |
|---|---|---|
| **auth** | `http://localhost:8091` | Register, login, validate tokens |
| **todo** | `http://localhost:8090` | Todo CRUD (calls auth for JWT validation) |

The `notifications` module is intentionally omitted from Mode 3 — in a real
distributed setup it would subscribe to a shared message broker (e.g. RabbitMQ)
instead of the in-process Watermill GoChannel.

---

## Angular Frontend (optional)

The Angular 17 SPA lives in `modules/todo/web/todoapp/`. It communicates with
the Go backend via the ConnectRPC TypeScript client and generated stubs from
`contracts/ts/`.

```bash
cd modules/todo/web/todoapp
npm install
npm start      # Angular dev server on http://localhost:4200
```

The `proxy.conf.json` forwards:
- `/api/auth.v1.*` → `http://localhost:8091`
- `/api/todo.v1.*` → `http://localhost:8090`

Works with both Mode 2 (single process) and Mode 3 (separate processes) — the
Angular app only cares about the ports.

---

## Project Structure

```
poc/
├── cmd/mmw/main.go              ← Mode 2 entry point (all modules in one process)
├── config/                      ← Root app config
├── contracts/
│   ├── proto/                   ← Protobuf definitions (source of truth)
│   ├── go/network/              ← Generated: proto structs + Connect stubs
│   ├── go/application/          ← Generated: service interfaces + events + errors
│   └── ts/                      ← Generated: TypeScript stubs
├── deployments/
│   └── docker-compose.yml       ← PostgreSQL container
├── libs/ogl/                    ← General-purpose Go utilities
├── mmw/                         ← Platform library (github.com/piprim/mmw)
│   ├── pkg/platform/            ← Module runner, UoW, outbox relay, HTTP server…
│   └── cmd/mmw-cli/             ← mmw check arch, mmw new module, mmw new contract
├── modules/
│   ├── auth/
│   │   ├── auth.go              ← Module wiring (New, Start, PrivateService)
│   │   ├── cmd/auth/main.go     ← Mode 3 standalone entry point
│   │   └── cmd/migration/       ← Database migrations
│   ├── todo/
│   │   ├── todo.go              ← Module wiring
│   │   ├── cmd/todo/main.go     ← Mode 3 standalone entry point
│   │   ├── cmd/migration/       ← Database migrations
│   │   └── web/todoapp/         ← Angular 17 SPA
│   └── notifications/
│       └── notifications.go     ← Event subscriber (RocketChat optional)
├── stow/common/                 ← Shared scripts symlinked into every module
├── go.work                      ← Workspace (local module resolution, Mode 2)
├── go.mod                       ← Root module with pinned versions (Mode 1)
├── mise.toml                    ← Task runner config
└── configure                    ← One-time environment setup script
```

---

## Key Design Decisions

**Contracts are generated, not hand-written.** A custom `protoc-gen-go-contracts`
buf plugin reads each proto `service` block and generates the Go application-layer
interface, a no-op stub, event topic constants, and error code constants into
`contracts/go/application/{domain}/`. You write the proto once; everything else
follows.

**Transport swap requires only a composition-root change.** In Modes 1 & 2,
`todo` receives `authModule.PrivateService()` — a zero-cost in-process call.
In Mode 3, the same `defauth.AuthPrivateService` interface is satisfied by
`defauth.NewPrivateHTTPClient(connectClient)`. The todo application layer is
identical in both cases.

**Outbox guarantees event delivery.** Domain events are written to a
`{schema}.event` table inside the same database transaction as the business
data. A background `EventsRelay` polls the table and publishes to the Watermill
bus. A crash between the two steps is safe — the relay retries on restart.

**Architecture boundaries are enforced in CI.** `mise run arch:check` (or
`go run ./mmw/cmd/mmw-cli check arch`) runs `arch-go` against every module's
`arch-go.yml` rules. The pre-commit hook blocks commits that violate layer
boundaries.
