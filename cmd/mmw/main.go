// cmd/mmw/main.go
package main

import (
	"context"
	"log"
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
	"github.com/ovya/ogl/platform/config"
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

	logger, err := setupLogger()
	if err != nil {
		exit = 1
		log.Default().Printf("boostraping logger failed: %s", err)

		return
	}

	todoLogger := logger.With("app", "todo")
	notifLogger := logger.With("app", "notifications")

	watermillLogger := watermill.NewSlogLogger(todoLogger)
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
		logError(logger, "app error", eris.Wrap(err, "app failed to load configuration"))

		return
	}

	todoLogger.Info("todo config loaded")

	dbPool, err = getDatabasePoolConnexion(ctx, todoLogger, conf)
	if err != nil {
		logError(todoLogger, "creating database pool", err)

		return
	}

	todoApp, err := todo.New(dbPool, systemBus, todoLogger)
	if err != nil {
		logError(todoLogger, "creating app failed", err)
		return
	}

	modules := []oglcore.Module{
		todoApp,
		notifications.New(rawBus, notifLogger),
	}

	// platformRuner := runner.New(
	// 	conf, logger, dbPool, modules,
	// 	middleware.RecoveryMiddleware(logger),
	// 	middleware.LoggingMiddleware(logger, conf.Environment.IsDev()),
	// 	middleware.CORSMiddleware(conf),
	// )

	platformRuner := runner.New(logger, modules)

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

// setupLogger automatically configures the logger based on the environment
func setupLogger() (*slog.Logger, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		return nil, eris.New("environment variable APP_ENV not set.")
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

	var llogger *slog.Logger
	if isProd {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelDebug, // TODO: from plateform config
			ReplaceAttr: replaceErr,
		})
		llogger = slog.New(handler)
	} else {
		llogger = slog.New(oglslog.StdoutTxtHandler(slog.LevelDebug, replaceErr))
	}

	return llogger, nil
}

func getDatabasePoolConnexion(ctx context.Context, logger *slog.Logger, conf config.Config) (*pgxpool.Pool, error) {
	dbUrl := conf.GetDatabaseURL()
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
