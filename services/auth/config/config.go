// services/auth/internal/infra/config/config.go
package config

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	oglconfig "github.com/ovya/ogl/config"
	oglpfconfig "github.com/ovya/ogl/platform/config"
	"github.com/rotisserie/eris"
)

//go:embed configs/*.toml
var embeddedFS embed.FS

// getConfigFS returns the filesystem to use for reading config files.
// This is a variable so it can be mocked in tests.
var getConfigFS = func() fs.FS {
	return embeddedFS
}

type LogLevel string

// SlogLevel returns the slog.SlogLevel value corresponding to the string level
func (l LogLevel) SlogLevel() slog.Level {
	switch string(l) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo // default to info
	}
}

// String implements the Stringer interface
func (l LogLevel) String() string {
	return string(l)
}

// IsValid checks if the LogLevel value is valid
func (l LogLevel) IsValid() bool {
	switch string(l) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

type JWT struct {
	Secret string `env:"JWT_SECRET" json:"-"`
}

type Config struct {
	Database    *oglpfconfig.Database `mapstructure:"database"`
	Environment Environment           `env:"APP_ENV, required" mapstructure:"environment"`
	AppName     string                `env:"APP_NAME"`
	Server      *oglpfconfig.Server   `mapstructure:"server"`
	LogLevel    LogLevel              `mapstructure:"log-level"`
	JWT         JWT                   `json:"-"`
}

func (c *Config) GetAppEnv() fmt.Stringer {
	return c.Environment
}

var conf *Config

// Load reads the TOML files and automatically overrides them with Env Vars
// Load loads the configurations from embedded files:
// - configs/default.toml
// - configs/<APP_ENV>.toml if exist
// If envs is not nil, automatically overrides them with Env Vars.
func Load(ctx context.Context, envprefix string) (*Config, error) {
	if conf != nil {
		return conf, nil
	}

	conf := new(Config)

	configFS := getConfigFS()
	err := oglconfig.NewContext(ctx, configFS, envprefix).Fill(conf)
	if err != nil {
		return nil, eris.Wrap(err, "error filling config")
	}

	if conf.Database.Password == "" {
		return nil, eris.New("database password is empty")
	}

	if conf.JWT.Secret == "" {
		return nil, eris.New("jwt secret is empty")
	}

	return conf, nil
}
