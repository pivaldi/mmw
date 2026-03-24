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
	"github.com/ovya/ogl/platform"
	oglcore "github.com/ovya/ogl/platform/core"
	oglevents "github.com/ovya/ogl/platform/events"
	oglslog "github.com/ovya/ogl/slog"
	auth "github.com/pivaldi/mmw-auth"
	defauth "github.com/pivaldi/mmw-contracts/definitions/auth"
	notifications "github.com/pivaldi/mmw-notifications"
	todo "github.com/pivaldi/mmw-todo"
	mmwConfig "github.com/pivaldi/mmw/config"
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

	config, err := mmwConfig.Load(ctx)
	if err != nil {
		exitCode = 1
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	logger, err := oglslog.New(oglslog.HandlerText, config.LogLevel.SlogLevel())
	if err != nil {
		exitCode = 1
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	dbPool, err = getDatabasePoolConnexion(ctx, logger, config.TodoConfig.Database.URL())
	if err != nil {
		logError(logger, "creating database pool", err)
		return
	}

	rawBus := getRawbus(logger)
	defer rawBus.Close()
	systemBus := oglevents.NewWatermillBus(rawBus)

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
		AuthSvc:  defauth.NewInprocClient(authModule),
	})
	if err != nil {
		logError(logger, "failed to initialize todo module", err)
		return
	}

	// Create the notifications module
	notifEvents := todo.NotifyEvents
	notifEvents = append(notifEvents, auth.NotifyEvents...)
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
	modules := []oglcore.Module{
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
