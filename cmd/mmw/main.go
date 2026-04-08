// cmd/mmw/main.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/piprim/mmw/pkg/platform"
	pfcore "github.com/piprim/mmw/pkg/platform/core"
	pfevents "github.com/piprim/mmw/pkg/platform/events"
	pfslog "github.com/piprim/mmw/pkg/platform/slog"
	auth "github.com/pivaldi/mmw-auth"
	authdef "github.com/pivaldi/mmw-contracts/definitions/auth"
	tododef "github.com/pivaldi/mmw-contracts/definitions/todo"
	notifications "github.com/pivaldi/mmw-notifications"
	todo "github.com/pivaldi/mmw-todo"
	mmwconfig "github.com/pivaldi/mmw/config"
	"github.com/rotisserie/eris"
)

const (
	outputChannelBufferSize = 1024
	minDatabaseURLLength    = 20
)

var exitCode = 0

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	var dbPool *pgxpool.Pool

	defer func() {
		if dbPool != nil {
			dbPool.Close()
		}
		cancel()
		os.Exit(exitCode)
	}()

	config, err := mmwconfig.Load(ctx)
	if err != nil {
		exitCode = 1
		_, _ = fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	if config.Environment.IsDev() {
		go startPprofServer()
	}

	logger, err := pfslog.New(pfslog.HandlerText, config.LogLevel.SlogLevel())
	if err != nil {
		exitCode = 1
		_, _ = fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	dbPool, err = getDatabasePoolConnexion(ctx, logger, config.MainDatabase.URL())
	if err != nil {
		logError(logger, "failed to create database pool", err)
		return
	}

	rawBus := getRawbus(logger)
	defer rawBus.Close()
	systemBus := pfevents.NewWatermillBus(rawBus)

	// Create authModule first, todo depends on it.
	authModule, err := auth.New(auth.Infrastructure{
		DBPool:   dbPool,
		EventBus: systemBus,
		Logger:   logger.With("module", auth.ModuleName),
	})
	if err != nil {
		logError(logger, "failed to initialize auth module", err)
		return
	}

	// Create the todo module
	todoModule, err := todo.New(todo.Infrastructure{
		DBPool:   dbPool,
		EventBus: systemBus,
		Logger:   logger.With("module", todo.ModuleName),
		AuthSvc:  authdef.NewInprocClient(authModule.CombinedService()),
	})
	if err != nil {
		logError(logger, "failed to initialize todo module", err)
		return
	}

	// Create the notifications module
	notifEvents := tododef.Topics
	notifEvents = append(notifEvents, authdef.Topics...)
	notifInfra := notifications.Infrastructure{
		Subscriber:  rawBus,
		Logger:      logger.With("module", notifications.ModuleName),
		Topics:      notifEvents,
		WithNotifer: true,
	}
	notifModule, err := notifications.New(notifInfra)
	if err != nil {
		logError(logger, "failed to initialyze notifications module", err)
		return
	}

	// Platform startup
	logger.Info("Platform startup…")
	modules := []pfcore.Module{
		todoModule,
		authModule,
		notifModule,
	}

	err = platform.New(logger, modules).Run(ctx) // Blocks until shutdown
	if err != nil {
		logError(logger, "platform error", err)
		return
	}
}

func logError(logger *slog.Logger, msg string, err error) {
	exitCode = 1
	logger.Error(msg, "err", err)
}

func getDatabasePoolConnexion(ctx context.Context, logger *slog.Logger, dbUrl string) (*pgxpool.Pool, error) {
	logger.Info("connecting to database", "url", maskDatabaseURL(dbUrl))

	dbPool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		return nil, eris.Wrap(err, "connecting to database")
	}

	if err := dbPool.Ping(ctx); err != nil {
		return dbPool, eris.Wrap(err, "pinging database")
	}

	logger.Info("database connection established")

	return dbPool, nil
}

func getRawbus(logger *slog.Logger) *gochannel.GoChannel {
	watermillLogger := watermill.NewSlogLogger(logger)
	rawBus := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: outputChannelBufferSize,
			Persistent:          true,
		},
		watermillLogger,
	)

	return rawBus
}

// maskDatabaseURL masks sensitive parts of database URL for logging
func maskDatabaseURL(url string) string {
	// Simple masking - in production use more robust URL parsing
	if len(url) < minDatabaseURLLength {
		return "***"
	}

	return url[:10] + "***" + url[len(url)-10:]
}

func startPprofServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	err := func() error {
		server := &http.Server{
			Addr:        "localhost:6060",
			ReadTimeout: time.Minute,
			Handler:     mux,
		}

		return server.ListenAndServe()
	}()
	if err != nil {
		panic(err)
	}
}
