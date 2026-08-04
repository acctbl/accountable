package foundation

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

var ErrAWSWorkloadIdentityUnavailable = errors.New("AWS workload identity is unavailable")

const managedAWSRequestTimeout = 5 * time.Second

func loadAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	config, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithHTTPClient(&http.Client{Timeout: managedAWSRequestTimeout}),
		awsconfig.WithRetryMaxAttempts(2),
	)
	if err != nil {
		return aws.Config{}, ErrAWSWorkloadIdentityUnavailable
	}
	credentials, err := config.Credentials.Retrieve(ctx)
	if err != nil || !safeCredentialSource(credentials.Source) {
		return aws.Config{}, ErrAWSWorkloadIdentityUnavailable
	}
	return config, nil
}
