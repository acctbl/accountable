package storage

const (
	ProviderFile = "filesystem"
	ProviderS3   = "s3"
)

type Config struct {
	Provider      string
	Root          string
	Region        string
	Bucket        string
	Prefix        string
	ExpectedOwner string
	KMSKeyARN     string
}
