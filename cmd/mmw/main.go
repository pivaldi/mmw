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
	"github.com/ovya/ogl/core"
	oglos "github.com/ovya/ogl/os"
	"github.com/ovya/ogl/platform/runner"
	"github.com/pivaldi/mmw/internal/adapters/eventbus"
	"github.com/pivaldi/mmw/notifications"
	"github.com/pivaldi/mmw/todo"
)

const (
	outputChannelBufferSize = 1024
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := setupLogger()
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
	systemBus := eventbus.NewWatermillBus(rawBus)

	app := todo.New()
	if err := app.Bootstrap(ctx, oglos.EnvMap()); err != nil {
		panic(err)
	}

	defer app.Close()

	dbPool, err := app.GetBdPool()
	if err != nil {
		panic(err)
	}

	modules := []core.Module{
		todo.Build(dbPool, systemBus, logger),
		notifications.Build(rawBus, logger),
	}

	// if err := app.SetModules(modules); err != nil {
	// 	panic(err)
	// }

	config, err := app.GetConfig(ctx, nil)
	if err != nil {
		panic(err)
	}

	platform := runner.New(config, logger, dbPool, modules)

	err = platform.Run(ctx)
	if err != nil {
		panic(err)
	}

	// modules := []core.Module{
	// 	todo.Build(dbPool, systemBus, logger),
	// 	notifications.Build(rawBus, logger),
	// }

	// if err := app.SetModules(modules); err != nil {
	// 	panic(err)
	// }
	// app.Run(ctx)
}

// setupLogger creates a structured logger based on environment
func setupLogger() *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return slog.New(handler)
}
