package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/acctbl/accountable/internal/appconfig"
	"github.com/acctbl/accountable/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	config, err := appconfig.LoadAPI(args)
	if err != nil {
		return err
	}
	report, err := bootstrap.Preflight(ctx, config.Foundation)
	if err != nil {
		return err
	}
	flagState := "safe_default"
	if !report.BootstrapFlag.Defaulted {
		flagState = "evaluated"
	}
	fmt.Printf("preflight: safe; feature_flag: %s\n", flagState)
	return nil
}
