package config

import (
	"context"
	"embed"
	"io/fs"

	pfconfig "github.com/piprim/mmw/pkg/platform/config"
	pfslog "github.com/piprim/mmw/pkg/platform/slog"
	"github.com/rotisserie/eris"
)

//go:embed configs/*.toml
var embeddedFS embed.FS

var getConfigFS = func() fs.FS {
	return embeddedFS
}

type Config struct {
	pfconfig.Base
	MainDatabase pfconfig.Database `mapstructure:"database"`
	LogLevel     pfslog.LogLevel   `mapstructure:"log-level"`
}

func Load(ctx context.Context) (*Config, error) {
	conf := new(Config)
	var err error
	configFS := getConfigFS()
	err = pfconfig.NewContext(ctx, configFS, "").Fill(conf)

	if conf.MainDatabase.Password == "" {
		return nil, eris.New("database password is empty")
	}

	return conf, eris.Wrap(err, "error filling config")
}
