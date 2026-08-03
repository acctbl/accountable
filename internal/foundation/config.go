package foundation

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	SecretProviderFile  = "file"
	StorageProviderFile = "filesystem"
	CryptoProviderLocal = "local"
)

var secretRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)

// SecretRef is an opaque lookup key. It never contains secret material.
type SecretRef string

type Config struct {
	Secrets  SecretsConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Crypto   CryptoConfig
}

type SecretsConfig struct {
	Provider  string
	Directory string
}

type DatabaseConfig struct {
	DSNRef              SecretRef
	ConnectTimeout      time.Duration
	HealthCheckInterval time.Duration
	MaxConnections      int32
	Timezone            string
}

type StorageConfig struct {
	Provider string
	Root     string
}

type CryptoConfig struct {
	Provider string
	KeyRef   SecretRef
}

// FileConfig is embedded in each binary's strict TOML document.
type FileConfig struct {
	Secrets  SecretsFileConfig  `toml:"secrets"`
	Database DatabaseFileConfig `toml:"database"`
	Storage  StorageFileConfig  `toml:"storage"`
	Crypto   CryptoFileConfig   `toml:"crypto"`
}

type SecretsFileConfig struct {
	Provider  string `toml:"provider"`
	Directory string `toml:"directory"`
}

type DatabaseFileConfig struct {
	DSNRef              string `toml:"dsn_ref"`
	ConnectTimeout      string `toml:"connect_timeout"`
	HealthCheckInterval string `toml:"health_check_interval"`
	MaxConnections      int32  `toml:"max_connections"`
	Timezone            string `toml:"timezone"`
}

type StorageFileConfig struct {
	Provider string `toml:"provider"`
	Root     string `toml:"root"`
}

type CryptoFileConfig struct {
	Provider string `toml:"provider"`
	KeyRef   string `toml:"key_ref"`
}

func Parse(environment, configPath string, raw FileConfig) (Config, error) {
	if raw.Secrets.Provider == "" {
		return Config{}, errors.New("secrets.provider is required")
	}
	if raw.Secrets.Provider != SecretProviderFile {
		return Config{}, errors.New("secrets.provider is unsupported")
	}
	if raw.Secrets.Directory == "" {
		return Config{}, errors.New("secrets.directory is required")
	}
	if raw.Database.DSNRef == "" {
		return Config{}, errors.New("database.dsn_ref is required")
	}
	dsnRef, err := ParseSecretRef(raw.Database.DSNRef)
	if err != nil {
		return Config{}, fmt.Errorf("database.dsn_ref: %w", err)
	}
	connectTimeout, err := parsePositiveDuration("database.connect_timeout", raw.Database.ConnectTimeout)
	if err != nil {
		return Config{}, err
	}
	healthInterval, err := parsePositiveDuration("database.health_check_interval", raw.Database.HealthCheckInterval)
	if err != nil {
		return Config{}, err
	}
	if raw.Database.MaxConnections < 1 || raw.Database.MaxConnections > 128 {
		return Config{}, errors.New("database.max_connections must be between 1 and 128")
	}
	if raw.Database.Timezone != "UTC" {
		return Config{}, errors.New("database.timezone must be UTC")
	}
	if raw.Storage.Provider != StorageProviderFile {
		if raw.Storage.Provider == "" {
			return Config{}, errors.New("storage.provider is required")
		}
		return Config{}, errors.New("storage.provider is unsupported")
	}
	if raw.Storage.Root == "" {
		return Config{}, errors.New("storage.root is required")
	}
	if raw.Crypto.Provider != CryptoProviderLocal {
		if raw.Crypto.Provider == "" {
			return Config{}, errors.New("crypto.provider is required")
		}
		return Config{}, errors.New("crypto.provider is unsupported")
	}
	keyRef, err := ParseSecretRef(raw.Crypto.KeyRef)
	if err != nil {
		return Config{}, fmt.Errorf("crypto.key_ref: %w", err)
	}
	if environment == "production" {
		return Config{}, errors.New("production foundation providers are not configured")
	}
	base := filepath.Dir(configPath)
	return Config{
		Secrets: SecretsConfig{
			Provider:  raw.Secrets.Provider,
			Directory: absoluteFrom(base, raw.Secrets.Directory),
		},
		Database: DatabaseConfig{
			DSNRef:              dsnRef,
			ConnectTimeout:      connectTimeout,
			HealthCheckInterval: healthInterval,
			MaxConnections:      raw.Database.MaxConnections,
			Timezone:            raw.Database.Timezone,
		},
		Storage: StorageConfig{
			Provider: raw.Storage.Provider,
			Root:     absoluteFrom(base, raw.Storage.Root),
		},
		Crypto: CryptoConfig{
			Provider: raw.Crypto.Provider,
			KeyRef:   keyRef,
		},
	}, nil
}

func ParseSecretRef(value string) (SecretRef, error) {
	if !secretRefPattern.MatchString(value) || filepath.IsAbs(value) || value == "." || value == ".." {
		return "", errors.New("must be a 1 to 128 character reference using only letters, digits, '.', '_', ':', '/', or '-'")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", errors.New("must not traverse directories")
		}
	}
	if filepath.Clean(value) != value || value == ".." || len(value) >= 3 && value[:3] == "../" {
		return "", errors.New("must not traverse directories")
	}
	return SecretRef(value), nil
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
