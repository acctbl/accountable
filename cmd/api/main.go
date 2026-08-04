package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	config, err := loadConfig(args)
	if err != nil {
		return err
	}
	log.Printf(
		"configuration loaded environment=%s cell=%s role=%s revision=%s fingerprint=%s",
		config.Foundation.Environment, config.Foundation.CellID, config.Foundation.RuntimeRole,
		config.Foundation.Revision, config.Foundation.Fingerprint,
	)
	return serve(ctx, config)
}
