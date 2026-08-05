package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAcceptsCompleteManagedBootstrapEnvironment(t *testing.T) {
	t.Parallel()

	path := writeBootstrapConfig(t, `
aws_region = "eu-west-2"
database_host = "database.example.com"
environment = "development"
infisical_machine_identity_id = "identity-bootstrap"
infisical_project_id = "project-development"
infisical_secret_root = "/accountable/development-01"
`)
	config, err := loadConfig([]string{"--config", path})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if config.region != "eu-west-2" || config.databaseHost != "database.example.com" ||
		config.apiSecrets.SecretPath != "/accountable/development-01/api" ||
		config.migrateSecrets.SecretPath != "/accountable/development-01/migrate" {
		t.Fatalf("config = %#v", config)
	}
}

func TestLoadConfigRejectsMissingOrUnsafeManagedBootstrapEnvironment(t *testing.T) {
	t.Parallel()

	base := fileConfig{
		AWSRegion: "eu-west-2", DatabaseHost: "database.example.com", Environment: "development",
		MachineIdentityID: "identity-bootstrap", ProjectID: "project-development",
		SecretRoot: "/accountable/development-01",
	}
	for name, mutate := range map[string]func(*fileConfig){
		"local environment":      func(value *fileConfig) { value.Environment = "local" },
		"relative secret root":   func(value *fileConfig) { value.SecretRoot = "accountable/cell" },
		"traversing secret root": func(value *fileConfig) { value.SecretRoot = "/accountable/../cell" },
	} {
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			value := base
			mutate(&value)
			path := writeBootstrapConfig(t, `
aws_region = "`+value.AWSRegion+`"
database_host = "`+value.DatabaseHost+`"
environment = "`+value.Environment+`"
infisical_machine_identity_id = "`+value.MachineIdentityID+`"
infisical_project_id = "`+value.ProjectID+`"
infisical_secret_root = "`+value.SecretRoot+`"
`)
			if _, err := loadConfig([]string{"--config", path}); err == nil {
				t.Fatal("loadConfig accepted an unsafe environment")
			}
		})
	}
}

func writeBootstrapConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bootstrap.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
