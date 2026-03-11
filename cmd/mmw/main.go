// cmd/mmw/main.go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ovya/ogl/oglcore"
	"github.com/ovya/ogl/oglevents"
	oglos "github.com/ovya/ogl/oglos"
	"github.com/ovya/ogl/oglslog"
	"github.com/ovya/ogl/platform"
	"github.com/ovya/ogl/platform/middleware"
	"github.com/ovya/ogl/platform/runner"
	"github.com/pivaldi/mmw/notifications"
	"github.com/pivaldi/mmw/todo"
	"github.com/rotisserie/eris"
)

const (
	outputChannelBufferSize = 1024
	minDatabaseURLLength    = 20
)

var errFormater = eris.ToJSON

var logger *slog.Logger
var exit = 0

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		cancel()
		os.Exit(exit)
	}()

	logger, err := setupLogger()
	if err != nil {
		exit = 1
		logError("boostraping logge", err)

		return
	}

	watermillLogger := watermill.NewSlogLogger(logger)
	rawBus := gochannel.NewGoChannel(
		gochannel.Config{
			// Output channel buffer size
			OutputChannelBuffer: outputChannelBufferSize,
			// Persistent guarantees the channel won't drop messages if no subscriber is attached yet
			Persistent: true,
		},
		watermillLogger,
	)

	defer rawBus.Close()
	// Wrap the raw infrastructure in the Adapter.
	systemBus := oglevents.NewWatermillBus(rawBus)

	conf, err := todo.GetConfig(ctx, oglos.EnvMap())
	if err != nil {
		exit = 1
		logError("todo app error", eris.Wrap(err, "app failed to load configuration"))

		return
	}

	logger.Info("todo config loaded", "conf", conf)

	dbPool, err := getDatabasePoolConnexion(ctx, conf)
	if err != nil {
		exit = 1
		logError("creating todo database pool", err)

		return
	}
	defer dbPool.Close()

	todoLogger := logger.With("module", "todo")
	notifLogger := logger.With("module", "notifications")
	modules := []oglcore.Module{
		todo.Build(dbPool, systemBus, todoLogger),
		notifications.Build(rawBus, notifLogger),
	}

	platformRuner := runner.New(
		conf, logger, dbPool, modules,
		middleware.LoggingMiddleware(logger, conf.Environment.IsDev()),
		middleware.CORSMiddleware(conf),
	)

	logger.Info("Starting the platform...")
	err = platformRuner.Run(ctx)
	if err != nil {
		logError("platform error", err)
		exit = 1

		return
	}
}

func logError(msg string, err error) {
	logger.Error(msg, "details", errFormater(err, true))
}

// setupLogger automatically configures the logger based on the environment
func setupLogger() (*slog.Logger, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		return nil, eris.New("Environment variable APP_ENV not set.")
	}

	isProd := appEnv == "production"
	replaceErr := func(_ []string, a slog.Attr) slog.Attr {
		// Detect if the attribute value is an `error` type
		if err, isError := a.Value.Any().(error); isError {
			if isProd {
				return slog.Any(a.Key, eris.ToJSON(err, true))
			}

			return slog.String(a.Key, "\n"+eris.ToString(err, true))
		}

		return a
	}

	var logger *slog.Logger
	if isProd {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelWarn, // TODO: from plateform config
			ReplaceAttr: replaceErr,
		})
		logger = slog.New(handler)
	} else {
		logger = slog.New(oglslog.StdoutTxtHandler(slog.LevelDebug, replaceErr))
	}

	logger.Info("logger setup ok")

	return logger, nil
}

func getDatabasePoolConnexion(ctx context.Context, conf platform.Config) (*pgxpool.Pool, error) {
	dbUrl := conf.GetDatabaseURL()
	logger.Info("connecting to todo database", "url", maskDatabaseURL(dbUrl))

	dbPool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		return nil, eris.Wrap(err, "connecting to database")
	}

	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, eris.Wrap(err, "pinging database")
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
