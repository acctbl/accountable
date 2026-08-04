package foundation

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/acctbl/accountable/internal/platform/clock"
)

var ErrFoundationUnavailable = errors.New("foundation dependencies are unavailable")

type PreflightReport struct {
	BootstrapFlag FlagEvaluation
}

type Dependencies struct {
	Database      *Database
	Storage       Storage
	Crypto        Crypto
	FeatureFlags  *FeatureFlags
	BootstrapFlag FlagEvaluation
	clock         clock.Clock
	timeHealth    clock.Health
	maximumDBSkew time.Duration
}

func Build(ctx context.Context, config Config) (*Dependencies, error) {
	timeSource := clock.Clock(clock.System{})
	timeHealth := clock.Health(clock.SystemHealth{})
	if config.Time.Provider == TimeProviderLinux {
		timeHealth = clock.NewLinuxHealth(config.Time.MaxClockError)
	}
	if err := timeHealth.Check(ctx); err != nil {
		return nil, err
	}

	resolver, err := buildSecretSource(ctx, config, timeSource)
	if err != nil {
		return nil, ErrFoundationUnavailable
	}
	database, err := OpenDatabase(ctx, config.Database, resolver)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Dependencies, error) {
		database.Close()
		return nil, err
	}
	storage, err := buildStorage(ctx, config)
	if err != nil {
		return fail(err)
	}
	cryptor, err := buildCrypto(ctx, config, resolver)
	if err != nil {
		return fail(err)
	}
	flags := NewFeatureFlags()
	dependencies := &Dependencies{
		Database: database, Storage: storage, Crypto: cryptor,
		FeatureFlags: flags, BootstrapFlag: flags.BootstrapProbe(ctx),
		clock: timeSource, timeHealth: timeHealth, maximumDBSkew: config.Time.MaxDatabaseSkew,
	}
	checkCtx, cancel := context.WithTimeout(ctx, config.CheckTimeout)
	defer cancel()
	if err := dependencies.Check(checkCtx); err != nil {
		return fail(err)
	}
	return dependencies, nil
}

func (d *Dependencies) Check(ctx context.Context) error {
	if err := d.timeHealth.Check(ctx); err != nil {
		return err
	}
	if err := d.Database.Check(ctx); err != nil {
		return err
	}
	if err := d.Database.CheckClock(ctx, d.clock, d.maximumDBSkew); err != nil {
		return err
	}
	if err := d.Storage.Check(ctx); err != nil {
		return err
	}
	if err := d.Crypto.Check(ctx); err != nil {
		return err
	}
	return nil
}

func (d *Dependencies) Close() { d.Database.Close() }

func Preflight(ctx context.Context, config Config) (PreflightReport, error) {
	dependencies, err := Build(ctx, config)
	if err != nil {
		return PreflightReport{}, err
	}
	defer dependencies.Close()
	return PreflightReport{BootstrapFlag: dependencies.BootstrapFlag}, nil
}

func buildSecretSource(ctx context.Context, config Config, timeSource clock.Clock) (SecretSource, error) {
	if config.Secrets.Provider == SecretProviderFile {
		return NewFileSecretSource(config.Secrets.Directory)
	}
	awsConfig, err := loadAWSConfig(ctx, config.Secrets.AWSRegion)
	if err != nil {
		return nil, err
	}
	return NewInfisicalSecretSource(
		config.Secrets,
		&http.Client{Timeout: 10 * time.Second},
		awsConfig.Credentials,
		timeSource,
	), nil
}

func buildStorage(ctx context.Context, config Config) (Storage, error) {
	if config.Storage.Provider == StorageProviderFile {
		return NewFileStorage(config.Storage.Root)
	}
	awsConfig, err := loadAWSConfig(ctx, config.Storage.Region)
	if err != nil {
		return nil, err
	}
	return NewS3Storage(config.Storage, awsConfig), nil
}

func buildCrypto(ctx context.Context, config Config, secrets SecretSource) (Crypto, error) {
	if config.Crypto.Provider == CryptoProviderLocal {
		values, err := secrets.ResolveBatch(ctx, []SecretRef{config.Crypto.KeyRef})
		if err != nil {
			return nil, ErrCryptoUnavailable
		}
		key, ok := values[config.Crypto.KeyRef]
		if !ok {
			return nil, ErrCryptoUnavailable
		}
		return NewLocalCrypto(key)
	}
	awsConfig, err := loadAWSConfig(ctx, config.Crypto.Region)
	if err != nil {
		return nil, err
	}
	return NewAWSCrypto(ctx, config.Crypto, awsConfig)
}
