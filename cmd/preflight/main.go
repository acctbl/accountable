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
	config, err := appconfig.LoadFoundation(args, "")
	if err != nil {
		return err
	}
	fmt.Printf(
		"configuration: environment=%s cell=%s role=%s revision=%s fingerprint=%s\n",
		config.Environment, config.CellID, config.RuntimeRole, config.Revision, config.Fingerprint,
	)
	report, err := bootstrap.Preflight(ctx, config)
	if err != nil {
		return err
	}
	flagState := "safe_default"
	if !report.BootstrapFlag.Defaulted {
		flagState = "evaluated"
	}
	fmt.Printf(
		"preflight: safe; revision: %s; fingerprint: %s; feature_flag: %s\n",
		config.Revision, config.Fingerprint, flagState,
	)
	return nil
}
