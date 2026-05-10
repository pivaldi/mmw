package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	pfpg "github.com/piprim/mmw/pkg/platform/pg"
	pfslog "github.com/piprim/mmw/pkg/platform/slog"
	auth "github.com/pivaldi/mmw-auth"
	todo "github.com/pivaldi/mmw-todo"
	mmwconfig "github.com/pivaldi/mmw/config"
	"github.com/rotisserie/eris"
)

var exitCode = 0

type migrateFnc func(context.Context, *pgxpool.Pool) error

var migrateMapFncs = map[string]migrateFnc{
	"Migrating Auth module": auth.Migrate,
	"Migrating Todo module": todo.Migrate,
}

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

	logger, err := pfslog.New(pfslog.HandlerText, config.LogLevel.SlogLevel())
	if err != nil {
		exitCode = 1
		_, _ = fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	dbPool, err = pfpg.GetPgxPool(ctx, logger, config.MainDatabase.URL())
	if err != nil {
		exitCode = 1
		logError(logger, "failed to create database pool", err)

		return
	}

	for msg, mFnc := range migrateMapFncs {
		logger.Info(msg)
		if err := mFnc(ctx, dbPool); err == nil {
			continue
		}

		logError(logger, "failed to migrate", err)

		return
	}
}

func logError(logger *slog.Logger, msg string, err error) {
	exitCode = 1
	logger.Error(msg, "err", err)
}
