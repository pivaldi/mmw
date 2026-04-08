//go:build system

package system

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"connectrpc.com/connect"
	authdef "github.com/pivaldi/mmw-contracts/definitions/auth"
	authv1 "github.com/pivaldi/mmw-contracts/gen/go/auth/v1"
	"github.com/pivaldi/mmw-contracts/gen/go/auth/v1/authv1connect"
	todov1 "github.com/pivaldi/mmw-contracts/gen/go/todo/v1"
	"github.com/pivaldi/mmw-contracts/gen/go/todo/v1/todov1connect"
	pfevents "github.com/piprim/mmw/pkg/platform/events"
	auth "github.com/pivaldi/mmw-auth"
	todo "github.com/pivaldi/mmw-todo"
)

var (
	authServer *httptest.Server
	todoServer *httptest.Server
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Required by auth.New() and todo.New() config.Load() validation.
	os.Setenv("APP_ENV", "development")
	os.Setenv("APP_NAME", "test")
	os.Setenv("JWT_SECRET", "fake-secret")
	os.Setenv("DB_PASSWORD", "postgres")

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("mmw"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get connection string: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := auth.Migrate(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "auth migration failed: %v\n", err)
		os.Exit(1)
	}
	if err := todo.Migrate(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "todo migration failed: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	rawBus := gochannel.NewGoChannel(
		gochannel.Config{OutputChannelBuffer: 64, Persistent: false},
		watermill.NewSlogLogger(logger),
	)
	defer rawBus.Close()
	systemBus := pfevents.NewWatermillBus(rawBus)

	authModule, err := auth.New(auth.Infrastructure{
		DBPool:   pool,
		EventBus: systemBus,
		Logger:   logger.With("module", auth.ModuleName),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create auth module: %v\n", err)
		os.Exit(1)
	}

	todoModule, err := todo.New(todo.Infrastructure{
		DBPool:   pool,
		EventBus: systemBus,
		Logger:   logger.With("module", todo.ModuleName),
		AuthSvc:  authdef.NewInprocClient(authModule.CombinedService()),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create todo module: %v\n", err)
		os.Exit(1)
	}

	authServer = httptest.NewServer(authModule.Handler())
	defer authServer.Close()
	todoServer = httptest.NewServer(todoModule.Handler())
	defer todoServer.Close()

	os.Exit(m.Run())
}

func newAuthPublicClient() authv1connect.AuthPublicServiceClient {
	return authv1connect.NewAuthPublicServiceClient(http.DefaultClient, authServer.URL)
}
func newTodoClient(token string) todov1connect.TodoServiceClient {
	return todov1connect.NewTodoServiceClient(
		http.DefaultClient,
		todoServer.URL,
		connect.WithInterceptors(bearerInterceptor(token)),
	)
}

func bearerInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	}
}

func registerAndLogin(t *testing.T, login, password string) string {
	t.Helper()
	ctx := context.Background()
	ac := newAuthPublicClient()

	_, err := ac.Register(ctx, connect.NewRequest(&authv1.RegisterRequest{
		Login:    login,
		Password: password,
	}))
	require.NoError(t, err)

	loginResp, err := ac.Login(ctx, connect.NewRequest(&authv1.LoginRequest{
		Login:    login,
		Password: password,
	}))
	require.NoError(t, err)
	return loginResp.Msg.GetToken()
}

func TestSystem_Unauthenticated(t *testing.T) {
	tc := todov1connect.NewTodoServiceClient(http.DefaultClient, todoServer.URL)
	_, err := tc.ListTodos(context.Background(), connect.NewRequest(&todov1.ListTodosRequest{}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}

func TestSystem_TodoCRUDFlow(t *testing.T) {
	token := registerAndLogin(t, "crud@test.com", "password123")
	tc := newTodoClient(token)
	ctx := context.Background()

	createResp, err := tc.CreateTodo(ctx, connect.NewRequest(&todov1.CreateTodoRequest{
		Title:       "Buy milk",
		Description: "Full fat",
		Priority:    todov1.Priority_PRIORITY_MEDIUM,
	}))
	require.NoError(t, err)
	id := createResp.Msg.Todo.Id
	assert.NotEmpty(t, id)
	assert.Equal(t, "Buy milk", createResp.Msg.Todo.Title)
	assert.Equal(t, todov1.TaskStatus_TASK_STATUS_PENDING, createResp.Msg.Todo.Status)

	getResp, err := tc.GetTodo(ctx, connect.NewRequest(&todov1.GetTodoRequest{Id: id}))
	require.NoError(t, err)
	assert.Equal(t, "Buy milk", getResp.Msg.Todo.Title)

	newTitle := "Buy oat milk"
	updateResp, err := tc.UpdateTodo(ctx, connect.NewRequest(&todov1.UpdateTodoRequest{
		Id:    id,
		Title: &newTitle,
	}))
	require.NoError(t, err)
	assert.Equal(t, "Buy oat milk", updateResp.Msg.Todo.Title)

	completeResp, err := tc.CompleteTodo(ctx, connect.NewRequest(&todov1.CompleteTodoRequest{Id: id}))
	require.NoError(t, err)
	assert.Equal(t, todov1.TaskStatus_TASK_STATUS_COMPLETED, completeResp.Msg.Todo.Status)

	reopenResp, err := tc.ReopenTodo(ctx, connect.NewRequest(&todov1.ReopenTodoRequest{Id: id}))
	require.NoError(t, err)
	assert.Equal(t, todov1.TaskStatus_TASK_STATUS_PENDING, reopenResp.Msg.Todo.Status)

	_, err = tc.DeleteTodo(ctx, connect.NewRequest(&todov1.DeleteTodoRequest{Id: id}))
	require.NoError(t, err)

	_, err = tc.GetTodo(ctx, connect.NewRequest(&todov1.GetTodoRequest{Id: id}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeNotFound, connectErr.Code())
}

func TestSystem_ListTodos_StatusFilter(t *testing.T) {
	token := registerAndLogin(t, "listfilter@test.com", "password123")
	tc := newTodoClient(token)
	ctx := context.Background()

	for _, title := range []string{"Pending A", "Pending B"} {
		_, err := tc.CreateTodo(ctx, connect.NewRequest(&todov1.CreateTodoRequest{
			Title:    title,
			Priority: todov1.Priority_PRIORITY_MEDIUM,
		}))
		require.NoError(t, err)
	}
	doneResp, err := tc.CreateTodo(ctx, connect.NewRequest(&todov1.CreateTodoRequest{
		Title:    "Done C",
		Priority: todov1.Priority_PRIORITY_LOW,
	}))
	require.NoError(t, err)
	_, err = tc.CompleteTodo(ctx, connect.NewRequest(&todov1.CompleteTodoRequest{Id: doneResp.Msg.Todo.Id}))
	require.NoError(t, err)

	status := todov1.TaskStatus_TASK_STATUS_PENDING
	listResp, err := tc.ListTodos(ctx, connect.NewRequest(&todov1.ListTodosRequest{Status: &status}))
	require.NoError(t, err)
	assert.Equal(t, 2, len(listResp.Msg.Todos))
}

func TestSystem_TodoScopedToUser(t *testing.T) {
	tokenA := registerAndLogin(t, "usera@test.com", "passwordA")
	tokenB := registerAndLogin(t, "userb@test.com", "passwordB")
	ctx := context.Background()

	_, err := newTodoClient(tokenA).CreateTodo(ctx, connect.NewRequest(&todov1.CreateTodoRequest{
		Title:    "User A secret",
		Priority: todov1.Priority_PRIORITY_HIGH,
	}))
	require.NoError(t, err)

	listResp, err := newTodoClient(tokenB).ListTodos(ctx, connect.NewRequest(&todov1.ListTodosRequest{}))
	require.NoError(t, err)
	assert.Empty(t, listResp.Msg.Todos)
}
