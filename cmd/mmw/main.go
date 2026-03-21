// cmd/mmw/main.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	oglcore "github.com/ovya/ogl/platform/core"
	oglevents "github.com/ovya/ogl/platform/events"
	oglrunner "github.com/ovya/ogl/platform/runner"
	oglslog "github.com/ovya/ogl/slog"
	"github.com/pivaldi/mmw/auth"
	defauth "github.com/pivaldi/mmw/contracts/definitions/auth"
	"github.com/pivaldi/mmw/notifications"
	"github.com/pivaldi/mmw/todo"
	todoConfig "github.com/pivaldi/mmw/todo/config"
	"github.com/rotisserie/eris"
)

const (
	outputChannelBufferSize = 1024
	minDatabaseURLLength    = 20
)

var errFormater = eris.ToJSON

var exit = 0

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	var dbPool *pgxpool.Pool

	defer func() {
		if dbPool != nil {
			dbPool.Close()
		}
		cancel()
		os.Exit(exit)
	}()

	todoConf, err := todoConfig.Load(ctx, "TODO_")
	if err != nil {
		exit = 1
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	// TODO: Add logLevel in the mmw config
	logger, err := oglslog.New(todoConf.Environment.String(), todoConf.LogLevel.SlogLevel())
	if err != nil {
		exit = 1
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	todoLogger := logger.With("app", todo.AppName)
	authLogger := logger.With("app", auth.AppName)
	notifLogger := logger.With("app", "notifications")

	watermillLogger := watermill.NewSlogLogger(todoLogger)
	rawBus := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: outputChannelBufferSize,
			Persistent:          true,
		},
		watermillLogger,
	)
	defer rawBus.Close()
	systemBus := oglevents.NewWatermillBus(rawBus)

	dbPool, err = getDatabasePoolConnexion(ctx, logger, todoConf.Database.URL())
	if err != nil {
		logError(logger, "creating database pool", err)
		return
	}

	// Create authApp first, todo depends on it.
	authApp, err := auth.New(auth.Infrastructure{
		DBPool:   dbPool,
		EventBus: systemBus,
		Logger:   authLogger,
	})
	if err != nil {
		logError(todoLogger, "failed to initialize auth app", err)
		return
	}
	authSvc := defauth.NewInprocClient(authApp)

	todoApp, err := todo.New(todo.Infrastructure{
		DBPool:   dbPool,
		EventBus: systemBus,
		Logger:   todoLogger,
		AuthSvc:  authSvc,
	})
	if err != nil {
		logError(todoLogger, "failed to initialize todo app", err)
		return
	}

	modules := []oglcore.App{
		todoApp,
		authApp,
		notifications.New(rawBus, notifLogger),
	}

	platformRuner := oglrunner.New(logger, modules)

	logger.Info("Starting the platform...")
	err = platformRuner.Run(ctx)
	if err != nil {
		logError(logger, "platform error", err)
		return
	}
}

func logError(logger *slog.Logger, msg string, err error) {
	exit = 1
	logger.Error(msg, "details", errFormater(err, true))
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

// maskDatabaseURL masks sensitive parts of database URL for logging
func maskDatabaseURL(url string) string {
	// Simple masking - in production use more robust URL parsing
	if len(url) < minDatabaseURLLength {
		return "***"
	}

	return url[:10] + "***" + url[len(url)-10:]
}
