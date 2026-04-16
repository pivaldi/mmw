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
	pfpg "github.com/piprim/mmw/pkg/platform/pg"
	pfslog "github.com/piprim/mmw/pkg/platform/slog"
	auth "github.com/pivaldi/mmw-auth"
	authdef "github.com/pivaldi/mmw-contracts/go/application/auth"
	tododef "github.com/pivaldi/mmw-contracts/go/application/todo"
	notifications "github.com/pivaldi/mmw-notifications"
	todo "github.com/pivaldi/mmw-todo"
	mmwconfig "github.com/pivaldi/mmw/config"
	"github.com/rotisserie/eris"
)

const (
	outputChannelBufferSize = 1024
)

var exitCode = 0

func main() {
	// signal.NotifyContext cancels ctx on SIGINT / SIGTERM, which propagates a
	// graceful-shutdown signal to every running module via platform.Run.
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

	logger, err := initObservability(config)
	if err != nil {
		return
	}

	dbPool, err = pfpg.GetPgxPool(ctx, logger, config.MainDatabase.URL())
	if err != nil {
		logError(logger, "failed to create database pool", err)
		return
	}

	// Creates the in-process Watermill GoChannel and wraps it in the
	// platform SystemEventBus interface.
	// rawBus is the concrete GoChannel used directly by modules that need a
	// message.Subscriber (e.g. the todo module's event router, the notifications
	// module). eventBus is the publishing interface passed to every module so they
	// can emit domain events without depending on the Watermill type.
	rawBus := getRawbus(logger)
	eventBus := pfevents.NewWatermillBus(rawBus)
	defer rawBus.Close()

	modules, err := initModules(logger, dbPool, rawBus, eventBus)
	if err != nil {
		return
	}

	// platform.Run launches every module in its own goroutine via errgroup and
	// blocks until the context is cancelled or one module fails.
	logger.Info("Platform startup…")
	if err = platform.New(logger, modules).Run(ctx); err != nil {
		logError(logger, "platform error", err)
	}
}

// initObservability loads the application config and creates the structured logger.
// If config.ServerDebugEnabled is true, it also starts a pprof server on localhost:6060 in the background.
// Both resources are derived from config, so they belong together.
func initObservability(config *mmwconfig.Config) (*slog.Logger, error) {
	if config.ServerDebugEnabled {
		// pprof is only useful in development; binding to localhost keeps it off the network.
		go startPprofServer()
	}

	logger, err := pfslog.New(pfslog.HandlerText, config.LogLevel.SlogLevel())
	if err != nil {
		exitCode = 1
		_, _ = fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		//nolint:wrapcheck // Will be wrapped later.
		return nil, err
	}

	return logger, nil
}

// initModules wires and returns all application modules in dependency order.
//
// Ordering matters: auth must be initialised before todo because todo's Connect
// handler requires an AuthPrivateService to validate JWT tokens. Notifications
// subscribes to topics from both auth and todo, so it is initialised last.
func initModules(
	logger *slog.Logger,
	dbPool *pgxpool.Pool,
	rawBus *gochannel.GoChannel,
	eventBus pfevents.SystemEventBus,
) ([]pfcore.Module, error) {
	// 1. Auth — no inter-module dependencies.
	authModule, err := auth.New(auth.Infrastructure{
		DBPool:   dbPool,
		EventBus: eventBus,
		Logger:   logger.With("module", auth.ModuleName),
	})
	if err != nil {
		logError(logger, "failed to initialize auth module", err)

		//nolint:wrapcheck // Will be wrapped later.
		return nil, err
	}

	// 2. Todo — depends on auth's private service to validate bearer tokens.
	todoModule, err := todo.New(todo.Infrastructure{
		DBPool:     dbPool,
		EventBus:   eventBus,
		Subscriber: rawBus,
		Logger:     logger.With("module", todo.ModuleName),
		AuthSvc:    authModule.PrivateService(),
	})
	if err != nil {
		logError(logger, "failed to initialize todo module", err)

		//nolint:wrapcheck // Will be wrapped later.
		return nil, err
	}

	// 3. Notifications — subscribes to domain events from both auth and todo.
	//    The topic list is built by merging the two modules' exported topic slices.
	notifModule, err := notifications.New(notifications.Infrastructure{
		Subscriber:  rawBus,
		Logger:      logger.With("module", notifications.ModuleName),
		Topics:      append(tododef.Topics, authdef.Topics...),
		WithNotifer: false,
	})
	if err != nil {
		logError(logger, "failed to initialize notifications module", err)

		//nolint:wrapcheck // Will be wrapped later.
		return nil, err
	}

	return []pfcore.Module{todoModule, authModule, notifModule}, nil
}

func logError(logger *slog.Logger, msg string, err error) {
	exitCode = 1
	logger.Error(msg, "err", err)
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
