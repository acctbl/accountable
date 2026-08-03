package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAPIConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "api.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func completeAPIConfig(environment string, architectureProbe bool) string {
	return fmt.Sprintf(`environment = %q
listen_address = "127.0.0.1:8080"
architecture_probe = %t
allowed_origins = ["https://shell.example"]
trusted_proxy_cidrs = []
unary_rpc_timeout = "10s"
stream_rpc_timeout = "25s"

[secrets]
provider = "file"
directory = "secrets"

[database]
dsn_ref = "database.api_dsn"
connect_timeout = "5s"
health_check_interval = "2s"
max_connections = 16
timezone = "UTC"

[storage]
provider = "filesystem"
root = "storage"

[crypto]
provider = "local"
key_ref = "crypto.primary_key"
`, environment, architectureProbe)
}

func TestConfigRequiresFoundationDependencies(t *testing.T) {
	t.Parallel()

	path := writeAPIConfig(t, `environment = "development"
listen_address = "127.0.0.1:8080"
architecture_probe = false
allowed_origins = ["http://localhost:3000"]
trusted_proxy_cidrs = []
unary_rpc_timeout = "10s"
stream_rpc_timeout = "25s"
`)

	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "secrets") {
		t.Fatalf("loadConfig error = %v, want missing secrets refusal", err)
	}
}

func TestRunRequiresOneAbsoluteConfigPath(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		nil,
		{"--config", "api.toml"},
		{"--config", "/tmp/api.toml", "unexpected"},
	} {
		err := run(context.Background(), args)
		if err == nil || !strings.Contains(err.Error(), "--config") {
			t.Fatalf("run(%q) error = %v, want --config usage error", args, err)
		}
	}
}

func TestProductionConfigCannotInheritDevelopmentDefaults(t *testing.T) {
	t.Parallel()

	path := writeAPIConfig(t, `environment = "production"
`)

	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "listen_address") {
		t.Fatalf("loadConfig error = %v, want missing listen_address", err)
	}
}

func TestNonProductionConfigIsCompleteAndExplicit(t *testing.T) {
	t.Parallel()

	path := writeAPIConfig(t, completeAPIConfig("staging", false))
	loaded, err := loadConfig([]string{"--config", path})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if loaded.Environment != "staging" || loaded.Addr != "127.0.0.1:8080" ||
		len(loaded.AllowedOrigins) != 1 || loaded.UnaryRPCDeadline.String() != "10s" ||
		loaded.StreamRPCDeadline.String() != "25s" {
		t.Fatalf("loaded config = %+v", loaded)
	}
}

func TestProductionRefusesUndecidedFoundationProviders(t *testing.T) {
	t.Parallel()

	path := writeAPIConfig(t, completeAPIConfig("production", false))
	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "production foundation providers") {
		t.Fatalf("loadConfig error = %v, want production provider refusal", err)
	}
}

func TestProductionConfigRejectsArchitectureProbe(t *testing.T) {
	t.Parallel()

	path := writeAPIConfig(t, completeAPIConfig("production", true))
	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "architecture_probe") {
		t.Fatalf("loadConfig error = %v, want production probe rejection", err)
	}
}

func TestConfigRejectsNonOriginCORSValue(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(
		completeAPIConfig("development", false),
		`https://shell.example`,
		`https://shell.example/path`,
		1,
	)
	path := writeAPIConfig(t, contents)
	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "allowed_origins") {
		t.Fatalf("loadConfig error = %v, want invalid origin rejection", err)
	}
}

func TestConfigRejectsRelativeTLSPaths(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(
		completeAPIConfig("development", false),
		`stream_rpc_timeout = "25s"`,
		`stream_rpc_timeout = "25s"
tls_certificate_file = "cert.pem"
tls_private_key_file = "key.pem"`,
		1,
	)
	path := writeAPIConfig(t, contents)
	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("loadConfig error = %v, want absolute TLS path rejection", err)
	}
}

func TestConfigRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "api.toml")
	if err := os.WriteFile(path, []byte(`environment = "development"
listen_address = "127.0.0.1:8080"
architecture_probe = false
allowed_origins = ["http://localhost:3000"]
trusted_proxy_cidrs = []
surprise = "not allowed"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "strict mode") {
		t.Fatalf("loadConfig error = %v, want unknown field rejection", err)
	}
}

func TestConfigRejectsInlineDatabaseSecret(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(
		completeAPIConfig("development", false),
		`dsn_ref = "database.api_dsn"`,
		`dsn_ref = "database.api_dsn"
dsn = "postgres://accountable:secret@database/accountable"`,
		1,
	)
	path := writeAPIConfig(t, contents)
	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "strict mode") {
		t.Fatalf("loadConfig error = %v, want inline secret field rejection", err)
	}
}
