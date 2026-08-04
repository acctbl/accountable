package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, contents string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "web.toml")
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath, filepath.Join(dir, "config.json")
}

func TestWebConfigWritesTheValidatedArtifact(t *testing.T) {
	t.Parallel()

	configPath, outputPath := writeConfig(t, `environment = "staging"
api_base_url = "https://api.staging.accountable.example"
architecture_probe = false
`)
	if err := run([]string{"--config", configPath, "--revision", "deadbeef", "--output", outputPath}); err != nil {
		t.Fatalf("run: %v", err)
	}
	document, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	want := `{"schema_version":1,"api_base_url":"https://api.staging.accountable.example",` +
		`"architecture_probe":false,"configuration_revision":"deadbeef"}` + "\n"
	if string(document) != want {
		t.Fatalf("artifact = %s, want %s", document, want)
	}
}

func TestWebConfigRefusalLeavesNoArtifact(t *testing.T) {
	t.Parallel()

	configPath, outputPath := writeConfig(t, `environment = "production"
api_base_url = "http://api.accountable.example"
architecture_probe = false
`)
	if err := run([]string{"--config", configPath, "--revision", "deadbeef", "--output", outputPath}); err == nil {
		t.Fatal("run accepted a production plain-HTTP api_base_url")
	}
	entries, err := os.ReadDir(filepath.Dir(outputPath))
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "web.toml" {
			t.Fatalf("refusal left an artifact: %s", entry.Name())
		}
	}
}

func TestWebConfigRefusesMalformedArguments(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"missing flags":   {"--config", "/tmp/web.toml"},
		"swapped flags":   {"--revision", "rev", "--config", "/tmp/web.toml", "--output", "/tmp/config.json"},
		"relative config": {"--config", "web.toml", "--revision", "rev", "--output", "/tmp/config.json"},
		"relative output": {"--config", "/tmp/web.toml", "--revision", "rev", "--output", "config.json"},
		"empty revision":  {"--config", "/tmp/web.toml", "--revision", "", "--output", "/tmp/config.json"},
	} {
		if _, _, _, err := parseArgs(args); err == nil {
			t.Errorf("%s: parseArgs accepted malformed arguments", name)
		}
	}
}
