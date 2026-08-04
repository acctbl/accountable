package bootstrap

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/acctbl/accountable/internal/platform/crypto"
	"github.com/acctbl/accountable/internal/platform/database"
	"github.com/acctbl/accountable/internal/platform/features"
	"github.com/acctbl/accountable/internal/platform/secret"
	"github.com/acctbl/accountable/internal/platform/storage"
)

const (
	TimeProviderSystem = "system"
	TimeProviderLinux  = "linux"

	infisicalCloudEUEndpoint = secret.InfisicalCloudEUEndpoint
)

var (
	postgresIdentifier  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)
	awsRegionPattern    = regexp.MustCompile(`^[a-z]{2}(?:-gov)?-[a-z]+-\d$`)
	awsAccountPattern   = regexp.MustCompile(`^[0-9]{12}$`)
	s3BucketPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	awsKMSKeyARNPattern = regexp.MustCompile(`^arn:aws(?:-us-gov)?:kms:([a-z0-9-]+):([0-9]{12}):key/[A-Za-z0-9-]+$`)
)

type Config struct {
	Environment  string
	CheckTimeout time.Duration
	Features     features.Config
	Secrets      secret.Config
	Database     database.Config
	Storage      storage.Config
	Crypto       crypto.Config
	Time         TimeConfig
}

type TimeConfig struct {
	Provider        string
	MaxClockError   time.Duration
	MaxDatabaseSkew time.Duration
}

// FileConfig is embedded in each binary's strict TOML document.
type FileConfig struct {
	CheckTimeout string             `toml:"foundation_check_timeout"`
	Features     FeaturesFileConfig `toml:"features"`
	Secrets      SecretsFileConfig  `toml:"secrets"`
	Database     DatabaseFileConfig `toml:"database"`
	Storage      StorageFileConfig  `toml:"storage"`
	Crypto       CryptoFileConfig   `toml:"crypto"`
	Time         TimeFileConfig     `toml:"time"`
}

type FeaturesFileConfig struct {
	Provider string `toml:"provider"`
}

type SecretsFileConfig struct {
	Provider          string `toml:"provider"`
	Directory         string `toml:"directory"`
	SiteURL           string `toml:"site_url"`
	AWSRegion         string `toml:"aws_region"`
	ProjectID         string `toml:"project_id"`
	Environment       string `toml:"environment"`
	SecretPath        string `toml:"secret_path"`
	AuthMethod        string `toml:"auth_method"`
	MachineIdentityID string `toml:"machine_identity_id"`
}

type DatabaseFileConfig struct {
	Host                string `toml:"host"`
	Port                uint16 `toml:"port"`
	Name                string `toml:"name"`
	User                string `toml:"user"`
	Role                string `toml:"role"`
	PasswordRef         string `toml:"password_ref"`
	TLSMode             string `toml:"tls_mode"`
	TLSRootCAFile       string `toml:"tls_root_ca_file"`
	ConnectTimeout      string `toml:"connect_timeout"`
	StatementTimeout    string `toml:"statement_timeout"`
	HealthCheckInterval string `toml:"health_check_interval"`
	MaxConnections      int32  `toml:"max_connections"`
	Timezone            string `toml:"timezone"`
}

type StorageFileConfig struct {
	Provider      string `toml:"provider"`
	Root          string `toml:"root"`
	Region        string `toml:"region"`
	Bucket        string `toml:"bucket"`
	Prefix        string `toml:"prefix"`
	ExpectedOwner string `toml:"expected_owner"`
	KMSKeyARN     string `toml:"kms_key_arn"`
}

type CryptoFileConfig struct {
	Provider  string `toml:"provider"`
	KeyRef    string `toml:"key_ref"`
	Region    string `toml:"region"`
	KMSKeyARN string `toml:"kms_key_arn"`
}

type TimeFileConfig struct {
	Provider        string `toml:"provider"`
	MaxClockError   string `toml:"max_clock_error"`
	MaxDatabaseSkew string `toml:"max_database_skew"`
}

func Parse(environment, configPath string, raw FileConfig) (Config, error) {
	if environment != "development" && environment != "staging" && environment != "production" {
		return Config{}, errors.New("environment must be development, staging, or production")
	}
	if raw.Features.Provider != features.ProviderNoop {
		return Config{}, errors.New("features.provider must be noop")
	}
	checkTimeout, err := parsePositiveDuration("foundation_check_timeout", raw.CheckTimeout)
	if err != nil {
		return Config{}, err
	}
	managed := environment == "staging" || environment == "production"
	secrets, err := parseSecrets(environment, managed, configPath, raw.Secrets)
	if err != nil {
		return Config{}, err
	}
	databaseConfig, err := parseDatabase(managed, configPath, raw.Database)
	if err != nil {
		return Config{}, err
	}
	storageConfig, err := parseStorage(managed, configPath, raw.Storage)
	if err != nil {
		return Config{}, err
	}
	cryptoConfig, err := parseCrypto(managed, raw.Crypto)
	if err != nil {
		return Config{}, err
	}
	timeConfig, err := parseTime(managed, raw.Time)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Environment:  environment,
		CheckTimeout: checkTimeout,
		Features:     features.Config{Provider: raw.Features.Provider},
		Secrets:      secrets,
		Database:     databaseConfig,
		Storage:      storageConfig,
		Crypto:       cryptoConfig,
		Time:         timeConfig,
	}, nil
}

func parseSecrets(environment string, managed bool, configPath string, raw SecretsFileConfig) (secret.Config, error) {
	if !managed {
		if raw.Provider != secret.ProviderFile || raw.Directory == "" {
			return secret.Config{}, errors.New("development secrets.provider must be file with a directory")
		}
		return secret.Config{Provider: raw.Provider, Directory: absoluteFrom(filepath.Dir(configPath), raw.Directory)}, nil
	}
	if raw.Provider != secret.ProviderInfisical {
		return secret.Config{}, errors.New("staging and production secrets.provider must be infisical")
	}
	parsedURL, err := url.Parse(raw.SiteURL)
	if err != nil || parsedURL.String() != infisicalCloudEUEndpoint {
		return secret.Config{}, errors.New("staging and production secrets.site_url must select managed Infisical Cloud EU")
	}
	if !awsRegionPattern.MatchString(raw.AWSRegion) {
		return secret.Config{}, errors.New("infisical aws_region is invalid")
	}
	if raw.ProjectID == "" || raw.Environment == "" || raw.MachineIdentityID == "" {
		return secret.Config{}, errors.New("infisical project_id, environment, and machine_identity_id are required")
	}
	if raw.Environment != environment {
		return secret.Config{}, errors.New("infisical environment must match the application environment")
	}
	if raw.AuthMethod != secret.AuthAWSIAM {
		return secret.Config{}, errors.New("infisical auth_method must be aws_iam")
	}
	if raw.SecretPath == "" || !strings.HasPrefix(raw.SecretPath, "/") || path.Clean(raw.SecretPath) != raw.SecretPath {
		return secret.Config{}, errors.New("infisical secret_path must be an absolute clean path")
	}
	return secret.Config{
		Provider:          raw.Provider,
		SiteURL:           raw.SiteURL,
		AWSRegion:         raw.AWSRegion,
		ProjectID:         raw.ProjectID,
		Environment:       raw.Environment,
		SecretPath:        raw.SecretPath,
		AuthMethod:        raw.AuthMethod,
		MachineIdentityID: raw.MachineIdentityID,
	}, nil
}

func parseDatabase(managed bool, configPath string, raw DatabaseFileConfig) (database.Config, error) {
	if raw.Host == "" || raw.Port == 0 || raw.Name == "" {
		return database.Config{}, errors.New("database host, port, and name are required")
	}
	if !postgresIdentifier.MatchString(raw.User) || !postgresIdentifier.MatchString(raw.Role) {
		return database.Config{}, errors.New("database user and role must be PostgreSQL identifiers")
	}
	passwordRef, err := secret.ParseRef(raw.PasswordRef)
	if err != nil {
		return database.Config{}, fmt.Errorf("database.password_ref: %w", err)
	}
	if managed && raw.TLSMode != database.TLSVerifyFull {
		return database.Config{}, errors.New("staging and production database.tls_mode must be verify-full")
	}
	if !managed && raw.TLSMode != database.TLSDisable {
		return database.Config{}, errors.New("development database.tls_mode must be disable")
	}
	rootCA := ""
	if raw.TLSRootCAFile != "" {
		rootCA = absoluteFrom(filepath.Dir(configPath), raw.TLSRootCAFile)
	}
	connectTimeout, err := parsePositiveDuration("database.connect_timeout", raw.ConnectTimeout)
	if err != nil {
		return database.Config{}, err
	}
	statementTimeout, err := parsePositiveDuration("database.statement_timeout", raw.StatementTimeout)
	if err != nil {
		return database.Config{}, err
	}
	healthInterval, err := parsePositiveDuration("database.health_check_interval", raw.HealthCheckInterval)
	if err != nil {
		return database.Config{}, err
	}
	if raw.MaxConnections < 1 || raw.MaxConnections > 128 {
		return database.Config{}, errors.New("database.max_connections must be between 1 and 128")
	}
	if raw.Timezone != "UTC" {
		return database.Config{}, errors.New("database.timezone must be UTC")
	}
	return database.Config{
		Host:                raw.Host,
		Port:                raw.Port,
		Name:                raw.Name,
		User:                raw.User,
		Role:                raw.Role,
		PasswordRef:         passwordRef,
		TLSMode:             raw.TLSMode,
		TLSRootCAFile:       rootCA,
		ConnectTimeout:      connectTimeout,
		StatementTimeout:    statementTimeout,
		HealthCheckInterval: healthInterval,
		MaxConnections:      raw.MaxConnections,
		Timezone:            raw.Timezone,
	}, nil
}

func parseStorage(managed bool, configPath string, raw StorageFileConfig) (storage.Config, error) {
	if !managed {
		if raw.Provider != storage.ProviderFile || raw.Root == "" {
			return storage.Config{}, errors.New("development storage.provider must be filesystem with a root")
		}
		return storage.Config{Provider: raw.Provider, Root: absoluteFrom(filepath.Dir(configPath), raw.Root)}, nil
	}
	if raw.Provider != storage.ProviderS3 {
		return storage.Config{}, errors.New("staging and production storage.provider must be s3")
	}
	if !awsRegionPattern.MatchString(raw.Region) || !s3BucketPattern.MatchString(raw.Bucket) ||
		!awsAccountPattern.MatchString(raw.ExpectedOwner) {
		return storage.Config{}, errors.New("S3 region, bucket, and expected_owner are invalid")
	}
	if raw.Prefix == "" || strings.HasPrefix(raw.Prefix, "/") || strings.Contains(raw.Prefix, "..") {
		return storage.Config{}, errors.New("S3 prefix must be a non-empty relative prefix")
	}
	if err := validateKMSKeyARN(raw.KMSKeyARN, raw.Region, raw.ExpectedOwner); err != nil {
		return storage.Config{}, fmt.Errorf("storage.kms_key_arn: %w", err)
	}
	return storage.Config{
		Provider: raw.Provider, Region: raw.Region, Bucket: raw.Bucket, Prefix: raw.Prefix,
		ExpectedOwner: raw.ExpectedOwner, KMSKeyARN: raw.KMSKeyARN,
	}, nil
}

func parseCrypto(managed bool, raw CryptoFileConfig) (crypto.Config, error) {
	if !managed {
		if raw.Provider != crypto.ProviderLocal {
			return crypto.Config{}, errors.New("development crypto.provider must be local")
		}
		keyRef, err := secret.ParseRef(raw.KeyRef)
		if err != nil {
			return crypto.Config{}, fmt.Errorf("crypto.key_ref: %w", err)
		}
		return crypto.Config{Provider: raw.Provider, KeyRef: keyRef}, nil
	}
	if raw.Provider != crypto.ProviderAWSKMS {
		return crypto.Config{}, errors.New("staging and production crypto.provider must be aws_kms")
	}
	if !awsRegionPattern.MatchString(raw.Region) {
		return crypto.Config{}, errors.New("crypto.region is invalid")
	}
	if err := validateKMSKeyARN(raw.KMSKeyARN, raw.Region, ""); err != nil {
		return crypto.Config{}, fmt.Errorf("crypto.kms_key_arn: %w", err)
	}
	return crypto.Config{Provider: raw.Provider, Region: raw.Region, KMSKeyARN: raw.KMSKeyARN}, nil
}

func parseTime(managed bool, raw TimeFileConfig) (TimeConfig, error) {
	wantProvider := TimeProviderSystem
	if managed {
		wantProvider = TimeProviderLinux
	}
	if raw.Provider != wantProvider {
		return TimeConfig{}, fmt.Errorf("time.provider must be %s for this environment", wantProvider)
	}
	maxClockError, err := parsePositiveDuration("time.max_clock_error", raw.MaxClockError)
	if err != nil {
		return TimeConfig{}, err
	}
	maxDatabaseSkew, err := parsePositiveDuration("time.max_database_skew", raw.MaxDatabaseSkew)
	if err != nil {
		return TimeConfig{}, err
	}
	return TimeConfig{Provider: raw.Provider, MaxClockError: maxClockError, MaxDatabaseSkew: maxDatabaseSkew}, nil
}

func validateKMSKeyARN(value, region, account string) error {
	matches := awsKMSKeyARNPattern.FindStringSubmatch(value)
	if len(matches) != 3 || matches[1] != region || account != "" && matches[2] != account {
		return errors.New("must be a key ARN in the configured region and account")
	}
	return nil
}

func absoluteFrom(base, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Join(base, value)
}

func parsePositiveDuration(field, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", field)
	}
	return duration, nil
}
