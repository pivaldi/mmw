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
	oglos "github.com/ovya/ogl/os"
	oglcore "github.com/ovya/ogl/platform/core"
	oglevents "github.com/ovya/ogl/platform/events"
	oglrunner "github.com/ovya/ogl/platform/runner"
	oglslog "github.com/ovya/ogl/slog"
	"github.com/pivaldi/mmw/auth"
	"github.com/pivaldi/mmw/todo"
	"github.com/rotisserie/eris"
)

const (
	outputChannelBufferSize = 1024
	minDatabaseURLLength    = 20
)

var errFormater = eris.ToJSON

var logger *slog.Logger
var dbPool *pgxpool.Pool
var exitCode = 0

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		cancel()
		dbPool.Close()
		os.Exit(exitCode)
	}()

	var err error

	envMap := oglos.EnvMap()

	todoConf, err := todo.GetConfig(ctx, "", envMap)
	if err != nil {
		exitCode = 1
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	authConf, err := auth.GetConfig(ctx, "AUTH_", envMap)
	if err != nil {
		exitCode = 1
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	logger, err = oglslog.New(todoConf.Environment.String(), todoConf.LogLevel.SlogLevel())
	if err != nil {
		exitCode = 1
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	todoLogger := logger.With("module", "todo")
	authLogger := logger.With("app", "auth")

	watermillLogger := watermill.NewSlogLogger(todoLogger)
	rawBus := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: outputChannelBufferSize,
			// Persistent guarantees the channel won't drop messages if no subscriber is attached yet
			Persistent: true,
		},
		watermillLogger,
	)

	defer rawBus.Close()
	// Wrap the raw infrastructure in the Adapter.
	systemBus := oglevents.NewWatermillBus(rawBus)

	// When extracted, you might swap Watermill's GoChannel for RabbitMQ here!
	// systemBus := setupRabbitMQ()
	// Wrap the raw infrastructure in the Adapter.
	// systemBus := oglevents.NewWatermillBus(rawBus)

	dbPool, err = getDatabasePoolConnexion(ctx, todoLogger, todoConf.Database.URL())
	if err != nil {
		logError("creating database pool", err)

		return
	}

	// Create authApp first (todo depends on it)
	authApp := auth.New(authConf, dbPool, systemBus, authLogger)

	// notifLogger := logger.With("module", "notifications")
	modules := []oglcore.Module{
		authApp,
		// Use RabitMQ consummer instead
		// notifications.Build(rawBus, notifLogger),
	}

	platformRuner := oglrunner.New(logger, modules)

	err = platformRuner.Run(ctx)
	if err != nil {
		logError("platform error", err)
		exitCode = 1

		return
	}
}

func logError(msg string, err error) {
	l := slog.New(oglslog.StderrTxtHandler(slog.LevelDebug, nil))
	l.Error(msg, "details", errFormater(err, true))
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
