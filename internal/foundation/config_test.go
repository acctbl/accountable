package foundation

import "testing"

func TestParseSecretRefRejectsSecretMaterialAndTraversal(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "../database", "/database", "postgres://user:password@host/db", "has space"} {
		if _, err := ParseSecretRef(value); err == nil {
			t.Errorf("ParseSecretRef(%q) succeeded", value)
		}
	}
}

func TestParseSecretRefAcceptsOpaqueReference(t *testing.T) {
	t.Parallel()

	ref, err := ParseSecretRef("database.api_dsn")
	if err != nil {
		t.Fatalf("ParseSecretRef: %v", err)
	}
	if ref != "database.api_dsn" {
		t.Fatalf("ref = %q", ref)
	}
}

func TestParseRejectsNonUTCDatabaseTimezone(t *testing.T) {
	t.Parallel()

	raw := FileConfig{
		Secrets:  SecretsFileConfig{Provider: "file", Directory: "secrets"},
		Database: DatabaseFileConfig{DSNRef: "database.dsn", ConnectTimeout: "1s", HealthCheckInterval: "1s", MaxConnections: 1, Timezone: "Europe/London"},
		Storage:  StorageFileConfig{Provider: "filesystem", Root: "storage"},
		Crypto:   CryptoFileConfig{Provider: "local", KeyRef: "crypto.key"},
	}
	if _, err := Parse("development", "/tmp/api.toml", raw); err == nil {
		t.Fatal("Parse accepted non-UTC database timezone")
	}
}
