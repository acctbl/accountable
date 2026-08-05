package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/acctbl/accountable/internal/tofupolicy"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("tofupolicy", flag.ContinueOnError)
	accountID := flags.String("account-id", "", "expected AWS account ID")
	planPath := flags.String("plan", "", "path to tofu show -json output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *accountID == "" || *planPath == "" {
		return errors.New("usage: tofupolicy --account-id <12-digit-id> --plan <plan.json>")
	}
	payload, err := os.ReadFile(*planPath)
	if err != nil {
		return fmt.Errorf("read saved plan JSON: %w", err)
	}
	violations, err := tofupolicy.Evaluate(payload, *accountID)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		fmt.Println("saved-plan-policy: pass")
		return nil
	}
	for _, violation := range violations {
		fmt.Fprintf(os.Stderr, "%s: %s\n", violation.Address, violation.Message)
	}
	return fmt.Errorf("saved-plan-policy: rejected %d violation(s)", len(violations))
}
