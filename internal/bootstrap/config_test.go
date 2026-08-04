package bootstrap

import (
	"strings"
	"testing"

	"github.com/acctbl/accountable/internal/platform/crypto"
	"github.com/acctbl/accountable/internal/platform/database"
	"github.com/acctbl/accountable/internal/platform/features"
	"github.com/acctbl/accountable/internal/platform/secret"
	"github.com/acctbl/accountable/internal/platform/storage"
)

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
	if config.Database.PasswordRef != "database/password" || config.Database.TLSMode != database.TLSVerifyFull ||
		config.Secrets.Provider != secret.ProviderInfisical || config.Storage.Provider != storage.ProviderS3 ||
		config.Crypto.Provider != crypto.ProviderAWSKMS || config.Time.Provider != TimeProviderLinux {
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
		Features:     FeaturesFileConfig{Provider: features.ProviderNoop},
		Secrets: SecretsFileConfig{
			Provider: secret.ProviderInfisical, SiteURL: "https://eu.infisical.com", AWSRegion: "eu-west-2",
			ProjectID: "project-production", Environment: "production", SecretPath: "/accountable/api",
			AuthMethod: secret.AuthAWSIAM, MachineIdentityID: "identity-production",
		},
		Database: DatabaseFileConfig{
			Host: "postgres.internal", Port: 5432, Name: "accountable", User: "accountable_login", Role: "accountable_api",
			PasswordRef: "database/password", TLSMode: database.TLSVerifyFull, ConnectTimeout: "5s",
			StatementTimeout: "10s", HealthCheckInterval: "30s", MaxConnections: 16, Timezone: "UTC",
		},
		Storage: StorageFileConfig{
			Provider: storage.ProviderS3, Region: "eu-west-2", Bucket: "accountable-production-private",
			Prefix: "application/", ExpectedOwner: "123456789012",
			KMSKeyARN: "arn:aws:kms:eu-west-2:123456789012:key/11111111-1111-1111-1111-111111111111",
		},
		Crypto: CryptoFileConfig{
			Provider: crypto.ProviderAWSKMS, Region: "eu-west-2",
			KMSKeyARN: "arn:aws:kms:eu-west-2:123456789012:key/22222222-2222-2222-2222-222222222222",
		},
		Time: TimeFileConfig{Provider: TimeProviderLinux, MaxClockError: "1s", MaxDatabaseSkew: "1s"},
	}
}

func developmentFileConfig() FileConfig {
	return FileConfig{
		CheckTimeout: "10s",
		Features:     FeaturesFileConfig{Provider: features.ProviderNoop},
		Secrets:      SecretsFileConfig{Provider: secret.ProviderFile, Directory: "secrets"},
		Database: DatabaseFileConfig{
			Host:                "127.0.0.1",
			Port:                5432,
			Name:                "accountable",
			User:                "postgres",
			Role:                "postgres",
			PasswordRef:         "database.password",
			TLSMode:             database.TLSDisable,
			ConnectTimeout:      "1s",
			StatementTimeout:    "1s",
			HealthCheckInterval: "1s",
			MaxConnections:      1,
			Timezone:            "UTC",
		},
		Storage: StorageFileConfig{Provider: storage.ProviderFile, Root: "storage"},
		Crypto:  CryptoFileConfig{Provider: crypto.ProviderLocal, KeyRef: "crypto.key"},
		Time: TimeFileConfig{
			Provider:        TimeProviderSystem,
			MaxClockError:   "1s",
			MaxDatabaseSkew: "1s",
		},
	}
}
