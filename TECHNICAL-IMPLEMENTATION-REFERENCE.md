# Technical Implementation Reference

Proof of concept of the [Go Modular Monolith Workspaces with Contract Definitions](https://github.com/pivaldi/go-modular-monolith-white-paper) — The Technical Documentation.

This proof of concept relies heavily on the [mmw framework](https://github.com/piprim/mmw) written for the occasion.

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Technical Implementation Reference](#technical-implementation-reference)
  - [1. Workspace Architecture](#1-workspace-architecture)
    - [Go Workspace Configuration](#go-workspace-configuration)
    - [Module Organization](#module-organization)
    - [Module Dependency Rules](#module-dependency-rules)
  - [2. Contract Definition Layer](#2-contract-definition-layer)
    - [Contract Structure](#contract-structure)
    - [Contract Components](#contract-components)
    - [Contract Dependency Rules](#contract-dependency-rules)
  - [3. Module Structure](#3-module-structure)
    - [Directory Layout](#directory-layout)
    - [go.mod Configuration](#gomod-configuration)
    - [Internal Package Organization](#internal-package-organization)
  - [4. Hexagonal Architecture Layers](#4-hexagonal-architecture-layers)
    - [Layer Dependencies](#layer-dependencies)
    - [Domain Layer](#domain-layer)
    - [Application Layer](#application-layer)
    - [Adapters Layer](#adapters-layer)
      - [Inbound Adapters](#inbound-adapters)
      - [Outbound Adapters](#outbound-adapters)
    - [Infrastructure Layer](#infrastructure-layer)
  - [5. Runtime Orchestration](#5-runtime-orchestration)
    - [mmw Platform: Module Lifecycle](#mmw-platform-module-lifecycle)
    - [Composition Root (main.go)](#composition-root-maingo)
    - [Supervision with errgroup](#supervision-with-errgroup)
  - [6. Protobuf Contracts](#6-protobuf-contracts)
    - [Contract Generation Pipeline](#contract-generation-pipeline)
    - [Directory Structure](#directory-structure)
    - [Protobuf Definition](#protobuf-definition)
    - [Code Generation](#code-generation)
    - [Using Generated Contracts](#using-generated-contracts)
  - [7. Testing & Operations](#7-testing--operations)
    - [Test Organization](#test-organization)
    - [Operational Commands](#operational-commands)
    - [Architecture Validation](#architecture-validation)
  - [8. CLI Tooling (`mmw-cli`)](#8-cli-tooling-mmw-cli)
    - [`mmw new module`](#mmw-new-module)
    - [`mmw new contract <name>`](#mmw-new-contract-name)
    - [`mmw check arch`](#mmw-check-arch)
    - [`mmw test coverage`](#mmw-test-coverage)
  - [Deployment](#deployment)

<!-- markdown-toc end -->

## 1. Workspace Architecture

### Go Workspace Configuration

**File:** `go.work`

```
go 1.26.1

use (
    .                        # github.com/pivaldi/mmw (root: config + cmd/mmw)
    ./contracts              # github.com/pivaldi/mmw-contracts
    ./libs/ogl               # ogl (file, os, string utilities)
    ./mmw                    # github.com/piprim/mmw (platform + CLI)
    ./modules/auth           # github.com/pivaldi/mmw-auth
    ./modules/notifications  # github.com/pivaldi/mmw-notifications
    ./modules/todo           # github.com/pivaldi/mmw-todo
)
```

**Purpose:** Coordinates multiple independent Go modules in a single repository.

**Key Properties:**
- Each `use` entry is an independent Go module with its own `go.mod`
- Workspace makes cross-module development seamless (no publishing required)
- `go test ./...` works across the entire workspace
- IDE jump-to-definition works across modules

### Module Organization

```
.
├── go.work                               # Workspace coordinator
├── go.mod                                # Root module: github.com/pivaldi/mmw
├── cmd/
│   └── mmw/
│       └── main.go                       # Composition root: wires all modules
├── config/
│   └── configs/                          # Root-level TOML config files
├── contracts/                            # go module: github.com/pivaldi/mmw-contracts
│   ├── go.mod                            # Only depends on connect + protobuf
│   ├── buf.yaml                          # Buf workspace config
│   ├── buf.gen.yaml                      # Common buf generation config
│   ├── buf.gen.auth.yaml                 # Auth-specific buf generation
│   ├── buf.gen.todo.yaml                 # Todo-specific buf generation
│   ├── cmd/
│   │   └── protoc-gen-go-contracts/      # Custom protoc plugin (generates contracts)
│   │       └── main.go
│   ├── proto/                            # Source of truth — proto definitions
│   │   ├── auth/v1/auth.proto
│   │   ├── todo/v1/todo.proto
│   │   ├── common/v1/common.proto        # Shared error types (DomainError)
│   │   └── options/v1/options.proto      # Custom proto options (e.g. topic)
│   ├── go/                               # Generated Go code (do not edit)
│   │   ├── application/
│   │   │   ├── auth/                     # Generated application contracts for auth
│   │   │   │   ├── auth_private_service_contract_gen.go
│   │   │   │   ├── auth_public_service_contract_gen.go
│   │   │   │   ├── connect_client.go     # HTTP clients wrapping Connect transport
│   │   │   │   ├── errors_gen.go         # Error code constants
│   │   │   │   ├── events_gen.go         # Topic constants + event type aliases
│   │   │   │   └── types.go
│   │   │   └── todo/                     # Generated application contracts for todo
│   │   │       ├── todo_service_contract_gen.go
│   │   │       ├── errors_gen.go
│   │   │       └── events_gen.go
│   │   └── network/                      # Standard protobuf + connect generated code
│   │       ├── auth/v1/                  # Protobuf types + Connect handler interfaces
│   │       └── todo/v1/
│   └── ts/                               # Generated TypeScript types
├── libs/
│   └── ogl/                              # go module: ogl
│       ├── file/                         # File utilities (zip, lock)
│       ├── os/                           # OS helpers
│       └── string/                       # String helpers
├── mmw/                                  # go module: github.com/piprim/mmw
│   ├── go.mod
│   ├── pkg/
│   │   ├── platform/                     # Runtime platform library
│   │   │   ├── core/app.go               # Module interface
│   │   │   ├── runner.go                 # platform.App + Run()
│   │   │   ├── server/                   # HTTP server with health, pprof, gRPC reflection
│   │   │   ├── events/                   # SystemEventBus (Watermill adapter)
│   │   │   ├── connect/                  # Connect interceptors (error logging)
│   │   │   ├── middleware/               # HTTP middleware (logger, recovery, CORS, auth)
│   │   │   ├── authctx/                  # Auth context propagation
│   │   │   ├── config/                   # Layered config loader (TOML + env)
│   │   │   ├── db/                       # StructArgs, migrator, outbox relay
│   │   │   ├── pg/uow/                   # Unit of Work (pgxpool + pgx.Tx)
│   │   │   └── slog/                     # Structured logging setup
│   │   ├── archtest/                     # Architecture boundary validators
│   │   └── scaffold/                     # Cookiecutter-style module generator
│   └── cmd/
│       └── mmw-cli/                      # mmw CLI tool (new, check, test)
│           └── cmd/
│               ├── new/                  # mmw new module / mmw new contract
│               ├── check/                # mmw check arch
│               └── test/                 # mmw test coverage
├── modules/
│   ├── auth/                             # go module: github.com/pivaldi/mmw-auth
│   │   ├── go.mod
│   │   ├── auth.go                       # Module factory + Infrastructure struct
│   │   └── internal/
│   ├── todo/                             # go module: github.com/pivaldi/mmw-todo
│   │   ├── go.mod
│   │   ├── todo.go                       # Module factory + Infrastructure struct
│   │   └── internal/
│   └── notifications/                    # go module: github.com/pivaldi/mmw-notifications
│       ├── go.mod
│       ├── notifications.go              # Module factory + Infrastructure struct
│       └── internal/
└── deployments/
    └── README.md
```

### Module Dependency Rules

**Enforced by Go compiler:**

1. **Contracts module:** Only depends on `connectrpc/connect` and `google.golang.org/protobuf`
   ```go
   // contracts/go.mod
   module github.com/pivaldi/mmw-contracts
   go 1.25.0
   require (
       connectrpc.com/connect v1.19.1
       google.golang.org/protobuf v1.36.11
   )
   ```

2. **Feature modules:** Import the shared contracts and platform — **never another module** !
   ```go
   // modules/todo/go.mod
   module github.com/pivaldi/mmw-todo
   require (
       github.com/pivaldi/mmw-contracts v0.0.0   // ← shared contracts
       github.com/piprim/mmw v0.0.0              // ← platform
       // Cannot import github.com/pivaldi/mmw-auth (architecture check error)
   )
   ```

3. **Cross-module calls:** Only through contract interfaces, never via concrete types
   - `todo` receives `defauth.AuthPrivateService` (interface) — not `*auth.Module`
   - Wiring happens exclusively in `cmd/mmw/main.go`

4. **Internal Packages:** Cannot be imported across module boundaries
   - `modules/auth/internal/` is inaccessible to `todo`
   - Enforced by Go's `internal` package rules + separate Go modules

## 2. Contract Definition Layer

### Contract Structure

**Location:** `contracts/go/application/{domain}/`

**Purpose:** Defines the public API contract for inter-module and client communication. Generated automatically from proto definitions using the custom `protoc-gen-go-contracts` plugin.

The contracts module replaces the hand-written "definitions" pattern by **auto-generating boilerplate** from the proto service definitions that already describe your API. You write the proto once; the plugin produces the Go interface, the noop stub, the event topics, and the error code constants.

### Contract Components

Each domain in `contracts/go/application/{domain}/` contains:

**1. Service Interface** (`*_service_contract_gen.go`) — generated by `protoc-gen-go-contracts`

```go
// contracts/go/application/auth/auth_private_service_contract_gen.go
// Code generated by protoc-gen-go-contracts. DO NOT EDIT.

package auth

type AuthPrivateService interface {
    ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error)
}

var ErrAuthPrivateServiceUnavailable = fmt.Errorf("auth private service unavailable")

// NoopAuthPrivateService satisfies the interface with always-error stubs.
type NoopAuthPrivateService struct{}
```

**2. Event Topics** (`events_gen.go`) — generated by `protoc-gen-go-contracts`

```go
// contracts/go/application/auth/events_gen.go
// Code generated by protoc-gen-go-contracts. DO NOT EDIT.

const (
    TopicUserRegistered  = "auth.user.registered.v1"
    TopicUserDeleted     = "auth.user.deleted.v1"
    TopicPasswordChanged = "auth.user.password_changed.v1"
    TopicUserLoggedIn    = "auth.user.logged_in.v1"
)

var Topics = []string{TopicUserRegistered, TopicUserDeleted, TopicPasswordChanged, TopicUserLoggedIn}

// Type aliases re-export proto message types under the application package.
type UserRegisteredEvent = authv1.UserRegisteredEvent
type UserDeletedEvent    = authv1.UserDeletedEvent
```

**3. Error Codes** (`errors_gen.go`) — generated by `protoc-gen-go-contracts`

```go
// contracts/go/application/todo/errors_gen.go
// Code generated by protoc-gen-go-contracts. DO NOT EDIT.

const (
    ErrorCodeInvalidTitle            = todov1.TodoErrorCode_TODO_ERROR_CODE_INVALID_TITLE
    ErrorCodeNotFound                = todov1.TodoErrorCode_TODO_ERROR_CODE_NOT_FOUND
    ErrorCodeCannotCompleteCancelled = todov1.TodoErrorCode_TODO_ERROR_CODE_CANNOT_COMPLETE_CANCELLED
    // ...
)
```

**4. HTTP Clients** (`connect_client.go`) — wraps the generated Connect transport

```go
// contracts/go/application/auth/connect_client.go

// PrivateHTTPClient adapts the Connect transport to AuthPrivateService.
// Use when the auth module runs in a separate process.
type PrivateHTTPClient struct {
    client authv1connect.AuthPrivateServiceClient
}

var _ AuthPrivateService = (*PrivateHTTPClient)(nil)

func NewPrivateHTTPClient(client authv1connect.AuthPrivateServiceClient) *PrivateHTTPClient {
    return &PrivateHTTPClient{client: client}
}

func (c *PrivateHTTPClient) ValidateToken(
    ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
    resp, err := c.client.ValidateToken(ctx, connect.NewRequest(req))
    // ...
}
```

### Contract Dependency Rules

**What contracts contain:**
- Generated service interfaces (with `NoopService` stubs)
- Event topic constants and proto message type aliases
- Error code constants (aliases over proto enum values)
- HTTP clients wrapping the Connect generated transport

**What contracts NEVER contain:**
- Business logic or validation
- Imports of any module's `internal/` packages
- Dependencies beyond `connectrpc/connect` and `google.golang.org/protobuf`

**Enforcement:**
- `ContractPurityValidator` in `mmw/pkg/archtest` validates zero forbidden dependencies
- `mmw check arch` runs this as a CI gate


## 3. Module Structure

### Directory Layout

```
modules/todo/
├── go.mod                            # Independent module: github.com/pivaldi/mmw-todo
├── todo.go                           # Module factory + Infrastructure struct
├── cmd/
│   └── migration/
│       └── config.go                 # Standalone migration entry point
└── internal/                         # Private implementation
    ├── domain/                       # Domain layer (aggregate, value objects, events)
    │   ├── todo.go                   # Aggregate root
    │   ├── value_objects.go          # Title, Status, Priority, etc.
    │   ├── events.go                 # Domain events (TodoCreated, TodoCompleted, …)
    │   ├── errors.go                 # Domain error sentinels
    │   └── snapshot.go               # ToSnapshot / FromSnapshot for persistence
    ├── application/                  # Application layer (use cases, ports)
    │   ├── service.go                # TodoService interface + application service
    │   ├── command/                  # Write operations (with UoW)
    │   ├── query/                    # Read operations (no UoW)
    │   ├── ports/                    # Secondary port interfaces
    │   │   ├── repository.go         # TodoRepository interface
    │   │   ├── events.go             # EventDispatcher interface
    │   │   └── mocks/                # Mockery-generated mocks (ports only)
    │   ├── dto/                      # Application-level DTOs
    │   └── authctx/                  # Application-layer auth context helpers
    ├── adapters/                     # Adapters layer (I/O boundaries)
    │   ├── inbound/
    │   │   ├── connect/              # Connect RPC handler + token validator
    │   │   └── events/               # Watermill inbound event handlers
    │   └── outbound/
    │       ├── persistence/postgres/ # PostgreSQL repository (pgx, no ORM)
    │       └── events/               # Outbox dispatcher (writes to todo.event)
    └── infra/
        ├── config/                   # Module-specific TOML config loader
        └── persistence/migrations/   # Goose SQL migration files (embedded FS)
```

### go.mod Configuration

```go
module github.com/pivaldi/mmw-todo

go 1.26.1

require (
    // Shared application contracts (interfaces, events, error codes)
    github.com/pivaldi/mmw-contracts v0.0.0

    // Runtime platform (server, events, UoW, middleware, config, …)
    github.com/piprim/mmw v0.0.0

    // DB driver
    github.com/jackc/pgx/v5 v5.9.1

    // Connect RPC
    connectrpc.com/connect v1.19.1

    // For integration tests
    github.com/testcontainers/testcontainers-go v0.41.0
)
```

### Internal Package Organization

**Domain Layer:** `internal/domain/`
- Pure business logic, zero external dependencies
- Aggregate root with private fields, public methods, domain event emission
- Value objects with validation in constructors
- `ToSnapshot()` / `FromSnapshot()` for persistence without exposing private state

**Application Layer:** `internal/application/`
- `command/` — write operations (use UoW + repo + event dispatcher)
- `query/` — read operations (skip UoW, read-only)
- `ports/` — secondary port interfaces (`TodoRepository`, `EventDispatcher`, `UnitOfWork`)
- `dto/` — application-level data transfer objects
- `ports/mocks/` — mockery-generated mocks for **ports only** (not services)

**Adapters Layer:** `internal/adapters/`
- `inbound/connect/` — Connect RPC handler implementing the generated interface
- `inbound/events/` — Watermill consumer handlers (e.g. `HandleUserDeleted`)
- `outbound/persistence/postgres/` — repository using `pgx` + UoW executor
- `outbound/events/` — outbox dispatcher writing to `todo.event` table

**Infrastructure Layer:** `internal/infra/`
- `config/` — module-specific config loading (embedded TOML + env)
- `persistence/migrations/` — embedded Goose SQL migration files


## 4. Hexagonal Architecture Layers

### Layer Dependencies

```
Domain (pure business logic, zero external deps)
    ↑
Application (use cases, ports)
    ↑
Adapters (I/O boundaries: Connect handler, Postgres repo, event handlers)
    ↑
todo.go (Infrastructure struct + Module factory)
    ↑
cmd/mmw/main.go (composition root)
```

**Rule:** Dependencies point inward. Inner layers never depend on outer layers.

### Domain Layer

**Location:** `modules/todo/internal/domain/`

**Aggregate Root** — enforces all business invariants:
```go
// modules/todo/internal/domain/todo.go
type Todo struct {
    id          TodoID
    title       Title
    description Description
    status      Status
    dueDate     *DueDate
    userID      uuid.UUID
    createdAt   time.Time
    updatedAt   time.Time
    events      []DomainEvent
}

func NewTodo(title Title, desc Description, dueDate *DueDate, userID uuid.UUID) (*Todo, error) {
    if userID == uuid.Nil {
        return nil, ErrInvalidUserID
    }
    now := time.Now().UTC()
    t := &Todo{
        id: NewTodoID(), title: title, description: desc,
        status: StatusPending, dueDate: dueDate,
        userID: userID, createdAt: now, updatedAt: now,
    }
    t.events = append(t.events, TodoCreated{TodoID: t.id, UserID: userID, At: now})

    return t, nil
}

func (t *Todo) Complete() error {
    if t.status == StatusCompleted { return ErrCannotModifyCompleted }
    if t.status == StatusCancelled { return ErrCannotCompleteCancelled }
    t.status = StatusCompleted
    t.updatedAt = time.Now().UTC()
    t.events = append(t.events, TodoCompleted{TodoID: t.id, At: t.updatedAt})

    return nil
}

func (t *Todo) Events() []DomainEvent { return t.events }
```

**Properties:**
- No external dependencies (standard library only)
- Fully testable without infrastructure
- State changes only through methods that validate invariants and emit events

### Application Layer

**Location:** `modules/todo/internal/application/`

**Secondary Ports** (`ports/`):
```go
type TodoRepository interface {
    Save(ctx context.Context, todo *domain.Todo) error
    FindByID(ctx context.Context, id domain.TodoID) (*domain.Todo, error)
    Delete(ctx context.Context, id domain.TodoID) error
    ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Todo, error)
    Health(ctx context.Context) (any, error)
}

type EventDispatcher interface {
    Dispatch(ctx context.Context, events []domain.DomainEvent) error
}

type UnitOfWork interface {
    WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
    Executor(ctx context.Context) db.DBExecutor
}
```

**Command Handler** (write operation with UoW):
```go
// modules/todo/internal/application/command/create_todo.go
func (c *CreateTodoCommand) Execute(ctx context.Context, req *proto.CreateTodoRequest) (*proto.CreateTodoResponse, error) {
    var result *proto.CreateTodoResponse
    err := c.unitOfWork.WithTransaction(ctx, func(txCtx context.Context) error {
        title, err := domain.NewTitle(req.GetTitle())
        if err != nil { return err }
        // ... build other value objects ...

        todo, err := domain.NewTodo(title, desc, dueDate, userID)
        if err != nil { return err }

        if err := c.repository.Save(txCtx, todo); err != nil { return err }
        if err := c.eventDispatcher.Dispatch(txCtx, todo.Events()); err != nil { return err }

        result = mapToProto(todo)
        return nil
    })
    return result, err
}
```

**Properties:**
- Orchestrates domain objects and ports
- No knowledge of HTTP, database drivers, or frameworks
- UoW ensures transactional safety for write operations
- Query handlers are read-only and skip UoW

### Adapters Layer

**Location:** `modules/todo/internal/adapters/`

#### Inbound Adapters

**Connect RPC Handler** (`adapters/inbound/connect/todo_handler.go`):

```go
package connect

// TodoHandler implements the generated todov1connect.TodoServiceHandler interface.
type TodoHandler struct {
    svc application.TodoService
}

// Compile-time assertion
var _ todov1connect.TodoServiceHandler = (*TodoHandler)(nil)

func (h *TodoHandler) CreateTodo(
    ctx context.Context,
    req *connect.Request[todov1.CreateTodoRequest],
) (*connect.Response[todov1.CreateTodoResponse], error) {
    resp, err := h.svc.CreateTodo(ctx, req.Msg)
    if err != nil {
        return nil, connectErrorFrom(err)
    }
    return connect.NewResponse(resp), nil
}
```

**Error mapping (`connectErrorFrom`):**

| Platform error code | Connect status |
|---|---|
| `NotFound` | `connect.CodeNotFound` |
| `InvalidArgument` | `connect.CodeInvalidArgument` |
| `FailedPrecondition` | `connect.CodeFailedPrecondition` |
| `Unauthenticated` | `connect.CodeUnauthenticated` |
| other | `connect.CodeInternal` |

**Inbound Event Handler** (`adapters/inbound/events/`):

```go
// HandleUserDeleted is a Watermill handler: when auth publishes "auth.user.deleted.v1",
// the todo module deletes all tasks belonging to that user.
func HandleUserDeleted(cmd *command.DeleteUserTasksCommand) message.HandlerFunc {
    return func(msg *message.Message) ([]*message.Message, error) {
        // unmarshal event, extract userID, execute command
        msg.Ack()
        return nil, nil
    }
}
```

#### Outbound Adapters

**Postgres Repository** (`adapters/outbound/persistence/postgres/`):

```go
type PostgresTodoRepository struct {
    uow ports.UnitOfWork
}

var _ ports.TodoRepository = (*PostgresTodoRepository)(nil)

func (r *PostgresTodoRepository) Save(ctx context.Context, todo *domain.Todo) error {
    snap := todo.Snapshot()
    query := `INSERT INTO todo.todo (id, title, description, status, due_date, user_id, ...)
              VALUES (@id, @title, @description, @status, @due_date, @user_id, ...)`
    _, err := r.uow.Executor(ctx).Exec(ctx, query, pgx.NamedArgs(pfdb.StructArgs(snap)))
    return err
}
```

Key points:
- No ORM — direct `pgx` queries with named args via `pfdb.StructArgs`
- `uow.Executor(ctx)` returns the active transaction executor or falls back to pool
- `todo.Snapshot()` bridges domain private fields and the DB

**Outbox Dispatcher** (`adapters/outbound/events/`):

```go
// PostgresOutboxDispatcher writes events to todo.event in the same transaction.
func (d *PostgresOutboxDispatcher) Dispatch(ctx context.Context, events []domain.DomainEvent) error {
    b := &pgx.Batch{}
    for _, evt := range events {
        payload, _ := proto.Marshal(evt.ProtoMessage())
        b.Queue(`INSERT INTO todo.event (aggregate_id, event_type, payload, occurred_at)
                 VALUES (@aggregate_id, @event_type, @payload, @occurred_at)`,
            pgx.NamedArgs{...})
    }
    return d.uow.Executor(ctx).SendBatch(ctx, b).Close()
}
```

Events are written **in the same transaction** as the business data. A background `EventsRelay` polls the outbox table and publishes to Watermill. This prevents silent event loss on crash between DB commit and publish.

### Infrastructure Layer

**Location:** `modules/todo/internal/infra/`

Each module loads its own configuration:
```go
cfg, err := config.Load(ctx, "")
```

**Environment Variables:**
```bash
# Required (secrets — never in config files)
DB_PASSWORD=secret_password

# Optional (runtime overrides)
APP_ENV=production         # Selects production.toml
SERVER_DEBUG_ENABLED=true  # Enables pprof + gRPC reflection
```

## 5. Runtime Orchestration

### mmw Platform: Module Lifecycle

**Platform module** (`mmw/pkg/platform`) provides `core.Module` and the `platform.App` runner:

```go
// mmw/pkg/platform/core/app.go
type Module interface {
    Start(ctx context.Context) error
}
```

Every feature module must implement this interface. Each module exposes:

- `Infrastructure` struct — ALL shared resources the module needs (DB pool, event bus, logger, other module contracts)
- `Module` struct — holds internal components (HTTP server, outbox relay, event router, logger)
- `New(Infrastructure) (*Module, error)` — wires all internals, returns ready-to-run module
- `Start(ctx) error` — starts HTTP server + outbox relay + event router via internal errgroup
- `Close() error` — releases module-specific resources
- `var _ core.Module = (*Module)(nil)` — compile-time interface assertion

### Composition Root (main.go)

**Location:** `cmd/mmw/main.go`

```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer func() { dbPool.Close(); cancel(); os.Exit(exitCode) }()

    // 1. Observability (config + logger + optional pprof server)
    config, logger, err := initObservability(ctx)

    // 2. Shared resources
    dbPool, err = getDatabasePoolConnexion(ctx, logger, config.MainDatabase.URL())
    rawBus := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 1024, Persistent: true}, ...)
    eventBus := pfevents.NewWatermillBus(rawBus)

    // 3. Module factories in dependency order
    modules, err := initModules(logger, dbPool, rawBus, eventBus)

    // 4. Run all modules (shared-fate errgroup)
    platform.New(logger, modules).Run(ctx)
}

func initModules(...) ([]pfcore.Module, error) {
    // 1. Auth — no inter-module dependencies
    authModule, _ := auth.New(auth.Infrastructure{
        DBPool: dbPool, EventBus: eventBus, Logger: logger.With("module", auth.ModuleName),
    })

    // 2. Todo — depends on auth's private service to validate JWT tokens
    todoModule, _ := todo.New(todo.Infrastructure{
        DBPool: dbPool, EventBus: eventBus, Subscriber: rawBus,
        Logger:  logger.With("module", todo.ModuleName),
        AuthSvc: authModule.PrivateService(), // ← AuthPrivateService interface (not *auth.Module)
    })

    // 3. Notifications — subscribes to events from both auth and todo
    notifModule, _ := notifications.New(notifications.Infrastructure{
        Subscriber: rawBus,
        Logger:     logger.With("module", notifications.ModuleName),
        Topics:     append(tododef.Topics, authdef.Topics...),
    })

    return []pfcore.Module{todoModule, authModule, notifModule}, nil
}
```

**Wiring flow:**
1. Shared resources (DB pool, event bus, logger) created once
2. Module factories called in dependency order (providers before consumers)
3. Cross-module access uses the contract interface: `authModule.PrivateService()` returns `defauth.AuthPrivateService`
4. `platform.New(logger, modules).Run(ctx)` starts all modules under a shared-fate errgroup

### Supervision with errgroup

**Pattern:** Shared fate architecture — if one module fails, all shut down together.

Example when the auth module fails:

- **WITHOUT shared fate**
  - auth: Crashed (database connection fails)
  - todo: Running (token validation fails silently)
  - Result: Zombie monolith in undefined state

- **WITH shared fate (platform.App)**
  - auth `Start()` returns error
  - platform errgroup cancels context for all modules
  - todo/notifications `Start()` detect `ctx.Done()` → graceful shutdown
  - Process exits; Kubernetes restarts with clean state

`platform.App.Run()` internally uses `errgroup.WithContext` across all module `Start()` calls.


## 6. Protobuf Contracts

### Contract Generation Pipeline

The [POC](https://github.com/pivaldi/mmw) uses **two levels** of protobuf generation:

1. **Standard generation** (buf + protoc-gen-go + protoc-gen-connect-go): produces raw protobuf types and Connect handler interfaces in `contracts/go/network/`.

2. **Custom `protoc-gen-go-contracts` plugin**: reads the proto service definitions and produces application-layer boilerplate in `contracts/go/application/`:
   - `*_contract_gen.go` — Go service interface + NoopService stub
   - `errors_gen.go` — error code constants (aliases over proto enum values)
   - `events_gen.go` — event topic constants + proto message type aliases

This replaces hand-written contract definitions: you write the proto once, and the contracts are generated automatically.

### Directory Structure

```
contracts/
├── proto/                  # Source of truth (edit these)
│   ├── auth/v1/auth.proto
│   ├── todo/v1/todo.proto
│   ├── common/v1/common.proto   # DomainError, shared types
│   └── options/v1/options.proto # (options.v1.topic) extension for event topics
└── go/                     # Generated code (do not edit)
    ├── application/
    │   ├── auth/           # Generated Go contracts for auth
    │   └── todo/           # Generated Go contracts for todo
    └── network/
        ├── auth/v1/        # Protobuf types + Connect service interfaces
        └── todo/v1/
```

### Protobuf Definition

**File:** `contracts/proto/todo/v1/todo.proto`

```protobuf
syntax = "proto3";
package todo.v1;
option go_package = "github.com/pivaldi/mmw-contracts/go/network/todo/v1;todov1";

import "options/v1/options.proto";

// DomainEvent messages carry (options.v1.topic) so protoc-gen-go-contracts
// can extract the Watermill topic string.
message UserTaskCreatedEvent {
    option (options.v1.topic) = "todo.userTasks.created.v1";
    string todo_id = 1;
    string user_id = 2;
    // ...
}

enum TodoErrorCode {
    TODO_ERROR_CODE_UNSPECIFIED = 0;
    TODO_ERROR_CODE_INVALID_TITLE = 1;
    TODO_ERROR_CODE_NOT_FOUND = 2;
    // ...
}

service TodoService {
    rpc CreateTodo(CreateTodoRequest) returns (CreateTodoResponse);
    rpc GetTodo(GetTodoRequest) returns (GetTodoResponse);
    rpc UpdateTodo(UpdateTodoRequest) returns (UpdateTodoResponse);
    rpc CompleteTodo(CompleteTodoRequest) returns (CompleteTodoResponse);
    rpc ReopenTodo(ReopenTodoRequest) returns (ReopenTodoResponse);
    rpc DeleteTodo(DeleteTodoRequest) returns (DeleteTodoResponse);
    rpc ListTodos(ListTodosRequest) returns (ListTodosResponse);
}
```

### Code Generation

```bash
# Build the custom plugin first (once):
cd contracts
go build -o $(go env GOPATH)/bin/protoc-gen-go-contracts ./cmd/protoc-gen-go-contracts

# Generate network types + connect handlers (buf.gen.yaml):
buf generate

# Generate auth contracts (buf.gen.auth.yaml):
buf generate --template buf.gen.auth.yaml

# Generate todo contracts (buf.gen.todo.yaml):
buf generate --template buf.gen.todo.yaml

# Lint proto files
buf lint

# Check for breaking changes
buf breaking --against '.git#branch=main'
```

The `buf.gen.{domain}.yaml` files invoke both the standard `protoc-gen-go` / `protoc-gen-connect-go` plugins AND the custom `protoc-gen-go-contracts` plugin.

### Using Generated Contracts

Modules import the application-layer package (never the network package directly):

```go
// modules/todo/internal/adapters/inbound/connect/token_validator.go
import defauth "github.com/pivaldi/mmw-contracts/go/application/auth"

func NewTokenValidator(svc defauth.AuthPrivateService) pfmiddleware.TokenValidator {
    return func(ctx context.Context, token string) (uuid.UUID, error) {
        resp, err := svc.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
        // ...
    }
}
```

The notifications module subscribes to topics from the `events_gen.go` constants:

```go
// cmd/mmw/main.go
notifModule, _ := notifications.New(notifications.Infrastructure{
    Topics: append(tododef.Topics, authdef.Topics...),
    // tododef.Topics = []string{"todo.userTasks.created.v1", "todo.userTask.deleted.v1", ...}
    // authdef.Topics = []string{"auth.user.registered.v1", "auth.user.deleted.v1", ...}
})
```

**Transport swapping:** The same `TodoHandler` works for both network transport (Connect over HTTP) and — in a future distributed setup — in-process via a `NoopTodoService` shim. The composition root decides; no application code changes.

**After (Distributed/Network):**
```go
// todo deployed separately; auth exposed over HTTP
todoModule, _ := todo.New(todo.Infrastructure{
    AuthSvc: defauth.NewPrivateHTTPClient(
        authv1connect.NewAuthPrivateServiceClient(&http.Client{}, "https://auth.internal"),
    ),
})
```


## 7. Testing & Operations

### Test Organization

**Domain unit tests:** Co-located with domain code
- Location: `modules/todo/internal/domain/*_test.go`
- Purpose: Business rules, value object validation, state transitions
- Dependencies: None (pure functions, no mocks needed)

**Application unit tests:** Co-located with command/query handlers
- Location: `modules/todo/internal/application/**/*_test.go`
- Purpose: Command/query handlers with mocked ports
- Dependencies: Mockery-generated mocks of `TodoRepository`, `EventDispatcher`, `UnitOfWork`
- **Do NOT mock `application.TodoService` itself** — use the real service with mocked ports

**Adapter integration tests:** Co-located with adapters
- Location: `modules/todo/internal/adapters/**/*_test.go`
- Purpose: Repository + real DB (testcontainers), Connect handler behaviour
- Dependencies: Real PostgreSQL via testcontainers

**Contract tests:** Verify module implements its own public contract
- Location: `modules/todo/test/contract/contract_test.go`
- Purpose: Validates that the todo module correctly implements `defTodo.TodoService`
- Lives in the **provider** module, not the consumer

**System tests:** Cross-module scenarios
- Location: `./test/system/`
- Purpose: Full HTTP → DB → event → response flows
- Dependencies: Full system (testcontainers for DB)

### Operational Commands

```bash
# Run full workspace tests
go test ./...

# Build the monolith binary
go build -o bin/mmw ./cmd/mmw

# Run database migrations (standalone)
go run ./modules/auth/cmd/migration
go run ./modules/todo/cmd/migration

# Verify module dependencies
go mod verify

# Tidy all dependencies (run in each module directory)
go mod tidy

# Generate protobuf code
cd contracts
buf generate
buf generate --template buf.gen.auth.yaml
buf generate --template buf.gen.todo.yaml

# Lint proto files
cd contracts && buf lint

# Test coverage table (via mmw-cli)
cd modules/todo && mmw test coverage

# Via mise (from repo root)
mise run test:all
mise run arch:check
mise run stow:sync   # sync shared scripts into module directories
```

### Architecture Validation

**Tool:** `mmw/pkg/archtest` — invoked via `mmw check arch`

```bash
# Validate all architectural boundaries
mmw check arch
# OR via mise:
mise run arch:check
```

Built-in validators:

| Validator | Rule |
|---|---|
| `ContractPurityValidator` | Contracts (`go/application/`) must not import application or infrastructure packages |
| `LibDependencyValidator` | Shared libs (`libs/ogl`, `mmw/`) must not import module-specific code |
| `DomainPurityValidator` | Domain layer must not import adapters, infra, or application packages |
| `ApplicationPurityValidator` | Application layer must not import adapters or infra packages |

Per-module arch checks are discovered and run via `mise run arch:check` in each module directory.

**Example output:**
```
Architecture Validation
────────────────────────────────────────────
✓ todo         Validating service architecture boundaries
✓ auth         Validating service architecture boundaries
✓ ContractPurity  Contract definitions must not import application or infra packages
✓ LibDependency   Shared libs must not import module-specific code
✓ DomainPurity    Domain layer must not import adapters/infra/application
✓ ApplicationPurity  Application layer must not import adapters/infra
────────────────────────────────────────────
All checks passed.
```


## 8. CLI Tooling (`mmw-cli`)

**Binary:** `mmw` (built from `mmw/cmd/mmw-cli/`)

```
mmw new module [--template <path>]    Scaffold a new module interactively
mmw new contract <name>               Generate contract definition files
mmw check arch                        Validate architectural boundaries
mmw test coverage [flags] [packages]  Print a test coverage table
```

### `mmw new module`

Runs an interactive `huh` form built dynamically from the `_templates/template.toml` manifest:

```toml
# mmw/pkg/scaffold/_templates/template.toml
[variables]
name          = ""                                              # required text input
org-prefix    = "github.com/acme"                              # text with default
with-connect  = true                                           # bool confirm
with-contract = true                                           # bool confirm
with-database = true                                           # bool confirm
license       = ["MIT", "BSD-3", "GNU GPL v3.0", "Apache Software License 2.0"]  # select

[conditions]
"modules/{{.Name}}/internal/adapters/inbound/connect"     = "{{if .WithConnect}}true{{end}}"
"modules/{{.Name}}/internal/adapters/inbound/inproc"      = "{{if .WithContract}}true{{end}}"
"modules/{{.Name}}/internal/infra/persistence/migrations" = "{{if .WithDatabase}}true{{end}}"
"modules/{{.Name}}/cmd/migration"                         = "{{if .WithDatabase}}true{{end}}"
"contracts/definitions"                                   = "{{if .WithContract}}true{{end}}"
"contracts/proto"                                         = "{{if and .WithContract .WithConnect}}true{{end}}"
```

Variable names normalise automatically: `with-connect`, `with_connect`, and `withConnect` all map to `.WithConnect` in templates.

After collecting inputs, `mmw new module`:
1. Generates the full module tree under `modules/<name>/`
2. Generates contract definitions (when `with-contract = true`)
3. Adds the new module to `go.work`
4. Registers the module in the root `mise.toml`

Pass `--template <path>` to use a custom template directory instead of the embedded defaults.

### `mmw new contract <name>`

Generates only the contract definition files for an existing module.

### `mmw check arch`

Runs all architectural boundary validators and exits non-zero on failure.

### `mmw test coverage`

Runs `go test -cover` and prints a formatted table:

```
┌─────────────────────────────────┬──────────┬──────────────┐
│ Package                         │ Coverage │ Status       │
├─────────────────────────────────┼──────────┼──────────────┤
│ internal/domain                 │ 91.4%    │ Good         │
├─────────────────────────────────┼──────────┼──────────────┤
│ internal/application/command    │ 78.2%    │ Good         │
├─────────────────────────────────┼──────────┼──────────────┤
│ pkg/scaffold                    │ 87.3%    │ Good         │
└─────────────────────────────────┴──────────┴──────────────┘
```

## Deployment

**Monolith Mode (default):**
```bash
# Build single binary running all modules in one process
go build -o bin/mmw ./cmd/mmw

# Run
./bin/mmw
```

**Distributed Mode:**

To distribute modules, swap the in-process contract accessor for a network `HTTPClient` in `main.go`. Only the wiring changes; application code stays unchanged.

**Before (In-Process — monolith):**
```go
// cmd/mmw/main.go
authModule, _ := auth.New(...)
todoModule, _ := todo.New(todo.Infrastructure{
    AuthSvc: authModule.PrivateService(), // in-process: returns AuthPrivateService interface
})
```

**After (Network — distributed):**
```go
// auth deployed separately; todo calls it over HTTP
todoModule, _ := todo.New(todo.Infrastructure{
    AuthSvc: defauth.NewPrivateHTTPClient(
        authv1connect.NewAuthPrivateServiceClient(&http.Client{}, "https://auth.internal"),
    ),
})
```

The `TodoHandler` inside the todo module calls the same `defauth.AuthPrivateService` interface — it never knows whether the transport is in-process or network.
