// cmd/migrate.go
package main

import (
	"fmt"
	"log/slog"
	"os"

	dbpgcli "github.com/ovya/ogl/db/cli"
	oglslog "github.com/ovya/ogl/slog"

	"github.com/rotisserie/eris"
)

func main() {
	conf, err := loadConfig()
	if err != nil {
		logError("loading config failed", err)

		return
	}

	if err := dbpgcli.Migrate(conf.Database.URL(), "auth", migrationsFS); err != nil {
		logError("command failed", err)

		return
	}

	os.Exit(0)
}

func logError(msg string, err error) {
	logger := slog.New(oglslog.StderrTxtHandler(slog.LevelDebug, nil))
	logger.Error(msg)
	// Print the formatted stack trace directly to stderr
	fmt.Fprintf(os.Stderr, "%s\n", eris.ToString(err, true))

	os.Exit(1)
}
