package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acctbl/accountable/internal/appconfig"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	configPath, revision, outputPath, err := parseArgs(args)
	if err != nil {
		return err
	}
	runtime, err := appconfig.LoadWebRuntime(configPath, revision)
	if err != nil {
		return err
	}
	document, err := runtime.RuntimeConfigJSON()
	if err != nil {
		return err
	}
	return writeArtifact(outputPath, document)
}

func parseArgs(args []string) (string, string, string, error) {
	usage := errors.New("usage: --config <absolute-path> --revision <revision> --output <absolute-path>")
	if len(args) != 6 || args[0] != "--config" || args[2] != "--revision" || args[4] != "--output" {
		return "", "", "", usage
	}
	configPath, revision, outputPath := args[1], args[3], args[5]
	if revision == "" || !filepath.IsAbs(configPath) || !filepath.IsAbs(outputPath) {
		return "", "", "", usage
	}
	return filepath.Clean(configPath), revision, filepath.Clean(outputPath), nil
}

func writeArtifact(path string, document []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".runtime-config-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	if _, err := temporary.Write(document); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}
