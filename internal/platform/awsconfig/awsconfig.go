package awsconfig

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

var ErrAWSWorkloadIdentityUnavailable = errors.New("AWS workload identity is unavailable")

const ManagedRequestTimeout = 5 * time.Second

func LoadConfig(ctx context.Context, region string) (aws.Config, error) {
	config, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithHTTPClient(&http.Client{Timeout: ManagedRequestTimeout}),
		awsconfig.WithRetryMaxAttempts(2),
	)
	if err != nil {
		return aws.Config{}, ErrAWSWorkloadIdentityUnavailable
	}
	credentials, err := config.Credentials.Retrieve(ctx)
	if err != nil || !SafeCredentialSource(credentials.Source) {
		return aws.Config{}, ErrAWSWorkloadIdentityUnavailable
	}
	return config, nil
}

func SafeCredentialSource(source string) bool {
	return source == "WebIdentityCredentials" || source == "CredentialsEndpointProvider" ||
		strings.HasPrefix(source, "EC2RoleProvider")
}
