package main

import (
	"context"

	oglos "github.com/ovya/ogl/os"
	"github.com/pivaldi/mmw/auth/internal/infra/config"
	"github.com/pivaldi/mmw/auth/internal/infra/persistence/migrations"

	// Needed to embed all the go migrations
	_ "github.com/pivaldi/mmw/auth/internal/infra/persistence/migrations/scripts"
)

var migrationsFS = migrations.FS

func loadConfig() (*config.Config, error) {
	ctx := context.Background()
	//nolint:wrapcheck // Generic loader
	return config.Load(ctx, "", oglos.EnvMap())
}
