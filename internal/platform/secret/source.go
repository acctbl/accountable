package secret

import (
	"context"
	"net/http"
	"time"

	"github.com/acctbl/accountable/internal/platform/awsconfig"
	"github.com/acctbl/accountable/internal/platform/clock"
)

func NewSource(ctx context.Context, config Config, timeSource clock.Clock) (SecretSource, error) {
	if config.Provider == ProviderFile {
		return NewFileSecretSource(config.Directory)
	}
	awsCfg, err := awsconfig.LoadConfig(ctx, config.AWSRegion)
	if err != nil {
		return nil, err
	}
	return NewInfisicalSecretSource(
		config,
		&http.Client{Timeout: 10 * time.Second},
		awsCfg.Credentials,
		timeSource,
	), nil
}
