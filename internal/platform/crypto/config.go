package crypto

import "github.com/acctbl/accountable/internal/platform/secret"

const (
	ProviderLocal  = "local"
	ProviderAWSKMS = "aws_kms"
)

type Config struct {
	Provider  string
	KeyRef    secret.Ref
	Region    string
	KMSKeyARN string
}
