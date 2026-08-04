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
	if environment == "staging" || environment == "production" {
		return fmt.Sprintf(`environment = %q
listen_address = "0.0.0.0:8080"
architecture_probe = %t
allowed_origins = ["https://shell.example"]
trusted_proxy_cidrs = ["10.0.0.0/8"]
unary_rpc_timeout = "10s"
stream_rpc_timeout = "25s"
foundation_check_timeout = "30s"

[features]
provider = "noop"

[secrets]
provider = "infisical"
site_url = "https://eu.infisical.com"
aws_region = "eu-west-2"
project_id = "accountable"
environment = %q
secret_path = "/accountable/api"
auth_method = "aws_iam"
machine_identity_id = "identity-api"

[database]
host = "postgres.internal"
port = 5432
name = "accountable"
user = "accountable_login"
role = "accountable_api"
password_ref = "database/password"
tls_mode = "verify-full"
connect_timeout = "5s"
statement_timeout = "10s"
health_check_interval = "2s"
max_connections = 16
timezone = "UTC"

[storage]
provider = "s3"
region = "eu-west-2"
bucket = "accountable-private"
prefix = "application/"
expected_owner = "123456789012"
kms_key_arn = "arn:aws:kms:eu-west-2:123456789012:key/11111111-1111-1111-1111-111111111111"

[crypto]
provider = "aws_kms"
region = "eu-west-2"
kms_key_arn = "arn:aws:kms:eu-west-2:123456789012:key/22222222-2222-2222-2222-222222222222"

[time]
provider = "linux"
max_clock_error = "1s"
max_database_skew = "1s"
`, environment, architectureProbe, environment)
	}
	return fmt.Sprintf(`environment = %q
listen_address = "127.0.0.1:8080"
architecture_probe = %t
allowed_origins = ["https://shell.example"]
trusted_proxy_cidrs = []
unary_rpc_timeout = "10s"
stream_rpc_timeout = "25s"
foundation_check_timeout = "30s"

[features]
provider = "noop"

[secrets]
provider = "file"
directory = "secrets"

[database]
host = "127.0.0.1"
port = 5432
name = "accountable"
user = "postgres"
role = "postgres"
password_ref = "database.password"
tls_mode = "disable"
connect_timeout = "5s"
statement_timeout = "10s"
health_check_interval = "2s"
max_connections = 16
timezone = "UTC"

[storage]
provider = "filesystem"
root = "storage"

[crypto]
provider = "local"
key_ref = "crypto.primary_key"

[time]
provider = "system"
max_clock_error = "1s"
max_database_skew = "1s"
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
	if err == nil || !strings.Contains(err.Error(), "features") {
		t.Fatalf("loadConfig error = %v, want incomplete foundation refusal", err)
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
	if loaded.Environment != "staging" || loaded.Addr != "0.0.0.0:8080" ||
		len(loaded.AllowedOrigins) != 1 || loaded.UnaryRPCDeadline.String() != "10s" ||
		loaded.StreamRPCDeadline.String() != "25s" {
		t.Fatalf("loaded config = %+v", loaded)
	}
}

func TestProductionAcceptsOnlyCompleteManagedFoundationProviders(t *testing.T) {
	t.Parallel()

	path := writeAPIConfig(t, completeAPIConfig("production", false))
	loaded, err := loadConfig([]string{"--config", path})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if loaded.Foundation.Secrets.Provider != "infisical" || loaded.Foundation.Storage.Provider != "s3" {
		t.Fatalf("foundation config = %+v", loaded.Foundation)
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

func TestConfigRejectsUnavailableTLSIdentityBeforeBootstrap(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(
		completeAPIConfig("development", false),
		`stream_rpc_timeout = "25s"`,
		`stream_rpc_timeout = "25s"
tls_certificate_file = "/definitely-missing/accountable-cert.pem"
tls_private_key_file = "/definitely-missing/accountable-key.pem"`,
		1,
	)
	path := writeAPIConfig(t, contents)
	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "TLS identity") {
		t.Fatalf("loadConfig error = %v, want unavailable TLS identity refusal", err)
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
		`password_ref = "database.password"`,
		`password_ref = "database.password"
password = "not-allowed-inline"`,
		1,
	)
	path := writeAPIConfig(t, contents)
	_, err := loadConfig([]string{"--config", path})
	if err == nil || !strings.Contains(err.Error(), "strict mode") {
		t.Fatalf("loadConfig error = %v, want inline secret field rejection", err)
	}
}
