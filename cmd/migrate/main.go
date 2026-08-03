package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/acctbl/accountable/db"
	"github.com/acctbl/accountable/internal/configfile"
	"github.com/acctbl/accountable/internal/foundation"
	"github.com/acctbl/accountable/internal/migration"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type fileConfig struct {
	foundation.FileConfig
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
	config, err := foundation.Parse(raw.Environment, path, raw.FileConfig)
	if err != nil {
		return err
	}
	resolver, err := foundation.NewFileSecretResolver(config.Secrets.Directory)
	if err != nil {
		return err
	}
	secret, err := resolver.Resolve(ctx, config.Database.DSNRef)
	if err != nil {
		return errors.New("database secret is unavailable")
	}
	dsn := secret.Bytes()
	database, err := sql.Open("pgx", string(dsn))
	clear(dsn)
	if err != nil {
		return errors.New("database configuration is invalid")
	}
	defer func() { _ = database.Close() }()
	return migration.Run(ctx, database, db.Migrations())
}
