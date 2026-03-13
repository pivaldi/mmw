// services/auth/module.go
package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ovya/ogl/platform/oglserver"
	"github.com/rotisserie/eris"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Module struct {
	server *oglserver.HTTPServer
}

func NewModule(logger *slog.Logger, port string) *Module {
	mux := http.NewServeMux()

	// Register Connect Handlers
	// path, handler := authv1connect.NewAuthServiceHandler(...)
	// mux.Handle(path, handler)

	// Wrap the mux in h2c for Connect, then pass it to our generic server
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})
	genericServer := oglserver.NewHTTPServer("auth-connect", port, h2cHandler, logger)

	return &Module{
		server: genericServer,
	}
}

// Start just delegates to the generic server!
func (m *Module) Start(ctx context.Context) error {
	return eris.Wrap(m.server.Start(ctx), "auth application failure")
}
