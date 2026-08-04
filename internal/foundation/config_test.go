package foundation

import (
	"strings"
	"testing"
)

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

	raw := developmentFileConfig()
	raw.Database.Timezone = "Europe/London"
	if _, err := Parse("development", "/tmp/api.toml", raw); err == nil {
		t.Fatal("Parse accepted non-UTC database timezone")
	}
}

func TestParseAcceptsExplicitManagedFoundation(t *testing.T) {
	t.Parallel()

	raw := managedFileConfig()
	config, err := Parse("production", "/etc/accountable/api.toml", raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if config.Database.PasswordRef != "database/password" || config.Database.TLSMode != DatabaseTLSVerifyFull ||
		config.Secrets.Provider != SecretProviderInfisical || config.Storage.Provider != StorageProviderS3 ||
		config.Crypto.Provider != CryptoProviderAWSKMS || config.Time.Provider != TimeProviderLinux {
		t.Fatalf("managed config = %+v", config)
	}
}

func TestParseRefusesLocalProvidersInStaging(t *testing.T) {
	t.Parallel()

	_, err := Parse("staging", "/tmp/api.toml", developmentFileConfig())
	if err == nil || !strings.Contains(err.Error(), "staging and production") {
		t.Fatalf("Parse error = %v, want managed-provider refusal", err)
	}
}

func TestParseRefusesCrossEnvironmentSecretProject(t *testing.T) {
	t.Parallel()

	raw := managedFileConfig()
	raw.Secrets.Environment = "staging"
	if _, err := Parse("production", "/etc/accountable/api.toml", raw); err == nil {
		t.Fatal("Parse accepted a staging secret environment for production")
	}
}

func managedFileConfig() FileConfig {
	return FileConfig{
		CheckTimeout: "30s",
		Features:     FeaturesFileConfig{Provider: FeatureProviderNoop},
		Secrets: SecretsFileConfig{
			Provider: SecretProviderInfisical, SiteURL: "https://eu.infisical.com", AWSRegion: "eu-west-2",
			ProjectID: "project-production", Environment: "production", SecretPath: "/accountable/api",
			AuthMethod: InfisicalAuthAWSIAM, MachineIdentityID: "identity-production",
		},
		Database: DatabaseFileConfig{
			Host: "postgres.internal", Port: 5432, Name: "accountable", User: "accountable_login", Role: "accountable_api",
			PasswordRef: "database/password", TLSMode: DatabaseTLSVerifyFull, ConnectTimeout: "5s",
			StatementTimeout: "10s", HealthCheckInterval: "30s", MaxConnections: 16, Timezone: "UTC",
		},
		Storage: StorageFileConfig{
			Provider: StorageProviderS3, Region: "eu-west-2", Bucket: "accountable-production-private",
			Prefix: "application/", ExpectedOwner: "123456789012",
			KMSKeyARN: "arn:aws:kms:eu-west-2:123456789012:key/11111111-1111-1111-1111-111111111111",
		},
		Crypto: CryptoFileConfig{
			Provider: CryptoProviderAWSKMS, Region: "eu-west-2",
			KMSKeyARN: "arn:aws:kms:eu-west-2:123456789012:key/22222222-2222-2222-2222-222222222222",
		},
		Time: TimeFileConfig{Provider: TimeProviderLinux, MaxClockError: "1s", MaxDatabaseSkew: "1s"},
	}
}

func developmentFileConfig() FileConfig {
	return FileConfig{
		CheckTimeout: "10s",
		Features:     FeaturesFileConfig{Provider: FeatureProviderNoop},
		Secrets:      SecretsFileConfig{Provider: SecretProviderFile, Directory: "secrets"},
		Database: DatabaseFileConfig{
			Host:                "127.0.0.1",
			Port:                5432,
			Name:                "accountable",
			User:                "postgres",
			Role:                "postgres",
			PasswordRef:         "database.password",
			TLSMode:             DatabaseTLSDisable,
			ConnectTimeout:      "1s",
			StatementTimeout:    "1s",
			HealthCheckInterval: "1s",
			MaxConnections:      1,
			Timezone:            "UTC",
		},
		Storage: StorageFileConfig{Provider: StorageProviderFile, Root: "storage"},
		Crypto:  CryptoFileConfig{Provider: CryptoProviderLocal, KeyRef: "crypto.key"},
		Time: TimeFileConfig{
			Provider:        TimeProviderSystem,
			MaxClockError:   "1s",
			MaxDatabaseSkew: "1s",
		},
	}
}
