package config

import (
	"context"
	"fmt"
	"os"

	authConfig "github.com/pivaldi/mmw-auth/config"
	todoConfig "github.com/pivaldi/mmw-todo/config"
	"github.com/rotisserie/eris"
)

type Config struct {
	TodoConfig *todoConfig.Config
	AuthConfig *authConfig.Config
}

func Load(ctx context.Context) (*Config, error) {
	conf := new(Config)
	var err error

	conf.TodoConfig, err = todoConfig.Load(ctx, "TODO_")
	if err != nil {
		fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")
	}

	return conf, nil
}
