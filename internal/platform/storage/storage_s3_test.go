package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type fakeS3 struct {
	public           bool
	keyARN           string
	object           []byte
	deleted          bool
	putErr           error
	deleteErr        error
	putKey           string
	encryptionChecks int
}

func (f *fakeS3) GetPublicAccessBlock(context.Context, *s3.GetPublicAccessBlockInput, ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error) {
	return &s3.GetPublicAccessBlockOutput{PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
		BlockPublicAcls: aws.Bool(f.public), IgnorePublicAcls: aws.Bool(f.public),
		BlockPublicPolicy: aws.Bool(f.public), RestrictPublicBuckets: aws.Bool(f.public),
	}}, nil
}

func (f *fakeS3) GetBucketEncryption(context.Context, *s3.GetBucketEncryptionInput, ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
	f.encryptionChecks++
	return &s3.GetBucketEncryptionOutput{ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
		Rules: []types.ServerSideEncryptionRule{{ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
			SSEAlgorithm: types.ServerSideEncryptionAwsKms, KMSMasterKeyID: aws.String(f.keyARN),
		}}},
	}}, nil
}

func (f *fakeS3) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if f.putErr != nil {
		return nil, f.putErr
	}
	if input.ServerSideEncryption != types.ServerSideEncryptionAwsKms || aws.ToString(input.SSEKMSKeyId) != f.keyARN {
		return nil, errors.New("unsafe encryption")
	}
	value, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	f.object = value
	f.putKey = aws.ToString(input.Key)
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(f.object))}, nil
}

func (f *fakeS3) DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	f.deleted = true
	return &s3.DeleteObjectOutput{}, f.deleteErr
}

func TestS3StoragePreflightProvesPrivateKMSRoundTripAndCleanup(t *testing.T) {
	t.Parallel()

	config := managedConfig()
	client := &fakeS3{public: true, keyARN: config.KMSKeyARN}
	storage := newS3Storage(config, client)
	if err := storage.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !client.deleted {
		t.Fatal("preflight object was not deleted")
	}
	if !strings.Contains(client.putKey, "/.accountable-preflight/foundation-proof/") {
		t.Fatalf("preflight key = %q, want access purpose", client.putKey)
	}
}

func TestS3StorageProbeChecksOnlyPublicAccessBlock(t *testing.T) {
	t.Parallel()

	config := managedConfig()
	client := &fakeS3{public: true, keyARN: config.KMSKeyARN}
	if err := newS3Storage(config, client).Probe(context.Background()); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if client.encryptionChecks != 0 || client.object != nil {
		t.Fatalf("Probe performed expensive work: encryption=%d object=%q", client.encryptionChecks, client.object)
	}
}

func TestS3StorageRefusesUnsafeBucketConfiguration(t *testing.T) {
	t.Parallel()

	config := managedConfig()
	for _, client := range []*fakeS3{
		{public: false, keyARN: config.KMSKeyARN},
		{public: true, keyARN: "arn:aws:kms:eu-west-2:123456789012:key/wrong"},
	} {
		if err := newS3Storage(config, client).Check(context.Background()); !errors.Is(err, ErrStorageUnavailable) {
			t.Fatalf("Check = %v, want storage unavailable", err)
		}
	}
}

func managedConfig() Config {
	return Config{
		Provider: ProviderS3, Region: "eu-west-2", Bucket: "accountable-private", Prefix: "application/",
		ExpectedOwner: "123456789012", KMSKeyARN: "arn:aws:kms:eu-west-2:123456789012:key/11111111-1111-1111-1111-111111111111",
		AccessPurpose: "foundation-proof",
	}
}
