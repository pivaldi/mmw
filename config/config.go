package config

import (
	"context"
	"embed"
	"io/fs"

	oglconfig "github.com/ovya/ogl/config"
	oglpfconfig "github.com/ovya/ogl/platform/config"
	oglslog "github.com/ovya/ogl/slog"
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
	MainDatabase oglpfconfig.Database `mapstructure:"database"`
	LogLevel     oglslog.LogLevel     `mapstructure:"log-level"`
}

func Load(ctx context.Context) (*Config, error) {
	conf := new(Config)
	var err error
	configFS := getConfigFS()
	err = oglconfig.NewContext(ctx, configFS, "").Fill(conf)

	if conf.MainDatabase.Password == "" {
		return nil, eris.New("database password is empty")
	}

	return conf, eris.Wrap(err, "error filling config")
}
