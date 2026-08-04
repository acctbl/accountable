package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/acctbl/accountable/internal/bootstrap"
	"github.com/acctbl/accountable/internal/configfile"
	"github.com/acctbl/accountable/internal/migration"
	"github.com/acctbl/accountable/internal/platform/clock"
	"github.com/acctbl/accountable/internal/platform/database"
	"github.com/acctbl/accountable/internal/platform/secret"
)

type fileConfig struct {
	bootstrap.FileConfig
	Environment string `toml:"environment"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	path, err := configfile.AbsolutePath(args)
	if err != nil {
		return err
	}
	var raw fileConfig
	if err := configfile.Decode(path, &raw); err != nil {
		return err
	}
	if raw.Environment != "development" && raw.Environment != "staging" && raw.Environment != "production" {
		return errors.New("environment must be development, staging, or production")
	}
	config, err := bootstrap.Parse(raw.Environment, path, raw.FileConfig)
	if err != nil {
		return err
	}
	source, err := secret.NewSource(ctx, config.Secrets, clock.System{})
	if err != nil {
		return err
	}
	sqlDB, err := database.OpenMigrationDatabase(ctx, config.Database, source)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	return migration.Run(ctx, sqlDB)
}
