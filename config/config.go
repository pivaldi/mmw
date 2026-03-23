package config

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	oglconfig "github.com/ovya/ogl/config"
	oglpfconfig "github.com/ovya/ogl/platform/config"
	oglslog "github.com/ovya/ogl/slog"
	authConfig "github.com/pivaldi/mmw-auth/config"
	todoConfig "github.com/pivaldi/mmw-todo/config"
	"github.com/rotisserie/eris"
)

//go:embed configs/*.toml
var embeddedFS embed.FS

// getConfigFS returns the filesystem to use for reading config files.
// This is a variable so it can be mocked in tests.
var getConfigFS = func() fs.FS {
	return embeddedFS
}

type Config struct {
	oglpfconfig.Base
	TodoConfig *todoConfig.Config
	AuthConfig *authConfig.Config
	LogLevel   oglslog.LogLevel `mapstructure:"log-level"`
}

func (c *Config) GetAppEnv() fmt.Stringer {
	return c.Environment
}

func Load(ctx context.Context) (*Config, error) {
	conf := new(Config)
	var err error

	conf.TodoConfig, err = todoConfig.Load(ctx, "TODO_")
	if err != nil {
		return nil, eris.Wrap(err, "failed to load todo config")
	}

	conf.AuthConfig, err = authConfig.Load(ctx, "AUTH_")
	if err != nil {
		return nil, eris.Wrap(err, "failed to load auth config")
	}

	configFS := getConfigFS()
	err = oglconfig.NewContext(ctx, configFS, "").Fill(conf)

	return conf, eris.Wrap(err, "error filling config")
}
