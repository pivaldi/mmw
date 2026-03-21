// services/auth/module.go
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	ogloutbox "github.com/ovya/ogl/db/outbox"
	ogluow "github.com/ovya/ogl/pg/uow"
	oglevents "github.com/ovya/ogl/platform/events"
	oglserver "github.com/ovya/ogl/platform/server"
	"github.com/pivaldi/mmw/auth/config"
	"github.com/pivaldi/mmw/auth/internal/adapters/inbound/connect"
	outboxevents "github.com/pivaldi/mmw/auth/internal/adapters/outbound/events"
	"github.com/pivaldi/mmw/auth/internal/adapters/outbound/persistence/postgres"
	"github.com/pivaldi/mmw/auth/internal/application"
	defauth "github.com/pivaldi/mmw/contracts/definitions/auth"
	"github.com/pivaldi/mmw/contracts/gen/go/auth/v1/authv1connect"
	"github.com/rotisserie/eris"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

const relayTableName = "auth.event"
const AppName = "Auth"

// App implements oglcore.App for the auth service.
type App struct {
	relay       *ogloutbox.EventsRelay
	server      *oglserver.HTTPServer
	logger      *slog.Logger
	authService *application.AuthApplicationService
}

// The infrastructure need to start the Auth App.
type Infrastructure struct {
	DBPool   *pgxpool.Pool
	EventBus oglevents.SystemEventBus
	Logger   *slog.Logger
	cfg      *config.Config
}

func (i *Infrastructure) WithConfig(cfg *config.Config) Infrastructure {
	i.cfg = cfg
	return *i
}

// New wires the auth module with all its dependencies.
func New(infra Infrastructure) (*App, error) {
	var cfg = infra.cfg
	if cfg == nil {
		var err error
		cfg, err = config.Load(context.Background(), "")
		if err != nil {
			return nil, eris.Wrap(err, "app failed to load config")
		}
	}

	userRepo := postgres.NewUserRepository(infra.DBPool)
	sessionRepo := postgres.NewSessionRepository(infra.DBPool)
	uow := ogluow.NewUnitOfWork(infra.DBPool)
	dispatcher := outboxevents.NewOutboxDispatcher(infra.DBPool)

	authService := application.NewAuthService(userRepo, sessionRepo, uow, dispatcher, cfg.JWT.Secret)
	authHandler := connect.NewAuthHandler(authService)

	mux := http.NewServeMux()
	path, handler := authv1connect.NewAuthServiceHandler(authHandler)
	mux.Handle(path, handler)

	h2cHandler := h2c.NewHandler(mux, &http2.Server{})
	server := oglserver.NewHTTPServer(cfg.Environment.String(), cfg.Server, h2cHandler, infra.Logger)

	return &App{
		relay:       ogloutbox.NewEnventsRelay(infra.DBPool, infra.EventBus, infra.Logger, relayTableName),
		server:      server,
		logger:      infra.Logger,
		authService: authService,
	}, nil
}

// Start runs the HTTP server and the outbox relay concurrently.
func (m *App) Start(ctx context.Context) error {
	m.logger.Info("starting auth module")
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return m.server.Start(gCtx)
	})

	g.Go(func() error {
		m.relay.Start(gCtx)
		return nil
	})

	return eris.Wrapf(g.Wait(), "%s failure", AppName)
}

// Ensure Module implements defauth.AuthService so it can be passed directly to
// defauth.NewInprocClient without an intermediate wrapper.
var _ defauth.AuthService = (*App)(nil)

// GetUser delegates to the internal auth application service.
func (m *App) GetUser(ctx context.Context, id string) (*defauth.UserDTO, error) {
	u, err := m.authService.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return u, nil
}

// ValidateToken delegates to the internal auth application service.
func (m *App) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	id, err := m.authService.ValidateToken(ctx, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("validating token: %w", err)
	}

	return id, nil
}
