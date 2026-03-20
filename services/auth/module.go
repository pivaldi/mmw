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
	"github.com/pivaldi/mmw/auth/internal/adapters/inbound/connect"
	outboxevents "github.com/pivaldi/mmw/auth/internal/adapters/outbound/events"
	"github.com/pivaldi/mmw/auth/internal/adapters/outbound/persistence/postgres"
	"github.com/pivaldi/mmw/auth/internal/application"
	"github.com/pivaldi/mmw/auth/internal/infra/config"
	defauth "github.com/pivaldi/mmw/contracts/definitions/auth"
	"github.com/pivaldi/mmw/contracts/gen/go/auth/v1/authv1connect"
	"github.com/rotisserie/eris"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

const relayTableName = "auth.event"

// Module implements oglcore.Module for the auth service.
type Module struct {
	appName     string
	relay       *ogloutbox.EventsRelay
	server      *oglserver.HTTPServer
	logger      *slog.Logger
	authService *application.AuthApplicationService
}

var conf *config.Config

// GetConfig loads the auth service configuration (cached after first call).
func GetConfig(ctx context.Context, envprefix string, envs map[string]string) (*config.Config, error) {
	if conf != nil {
		return conf, nil
	}
	var err error
	conf, err = config.Load(ctx, envprefix, envs)
	if err != nil {
		return nil, eris.Wrap(err, "failed to load auth configuration")
	}

	return conf, nil
}

// New wires the auth module with all its dependencies.
func New(cfg *config.Config, dbPool *pgxpool.Pool, eventBus oglevents.SystemEventBus, logger *slog.Logger) *Module {
	userRepo := postgres.NewUserRepository(dbPool)
	sessionRepo := postgres.NewSessionRepository(dbPool)
	uow := ogluow.NewUnitOfWork(dbPool)
	dispatcher := outboxevents.NewOutboxDispatcher(dbPool)

	authService := application.NewAuthService(userRepo, sessionRepo, uow, dispatcher, cfg.JWT.Secret)
	authHandler := connect.NewAuthHandler(authService)

	mux := http.NewServeMux()
	path, handler := authv1connect.NewAuthServiceHandler(authHandler)
	mux.Handle(path, handler)

	h2cHandler := h2c.NewHandler(mux, &http2.Server{})
	server := oglserver.NewHTTPServer(cfg.AppName, cfg.Environment.String(), cfg.Server, h2cHandler, logger)

	return &Module{
		appName:     cfg.AppName,
		relay:       ogloutbox.NewEnventsRelay(dbPool, eventBus, logger, relayTableName),
		server:      server,
		logger:      logger,
		authService: authService,
	}
}

// Start runs the HTTP server and the outbox relay concurrently.
func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("starting auth module")
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return m.server.Start(gCtx)
	})

	g.Go(func() error {
		m.relay.Start(gCtx)
		return nil
	})

	return eris.Wrapf(g.Wait(), "%s failure", m.GetName())
}

// GetName returns the module name.
func (m *Module) GetName() string {
	return "auth"
}

// Ensure Module implements defauth.AuthService so it can be passed directly to
// defauth.NewInprocClient without an intermediate wrapper.
var _ defauth.AuthService = (*Module)(nil)

// GetUser delegates to the internal auth application service.
func (m *Module) GetUser(ctx context.Context, id string) (*defauth.UserDTO, error) {
	u, err := m.authService.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return u, nil
}

// ValidateToken delegates to the internal auth application service.
func (m *Module) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	id, err := m.authService.ValidateToken(ctx, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("validating token: %w", err)
	}

	return id, nil
}
