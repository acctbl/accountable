package storage

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

func TestS3LocalStackContract(t *testing.T) {
	endpoint := os.Getenv("ACCOUNTABLE_TEST_AWS_ENDPOINT")
	if endpoint == "" {
		if os.Getenv("ACCOUNTABLE_REQUIRE_LOCALSTACK") == "1" {
			t.Fatal("ACCOUNTABLE_TEST_AWS_ENDPOINT is required")
		}
		t.Skip("run through task test:aws-contract")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	awsConfig := localStackAWSConfig(endpoint)
	kmsClient := kms.NewFromConfig(awsConfig)
	key, err := kmsClient.CreateKey(ctx, &kms.CreateKeyInput{})
	if err != nil || key.KeyMetadata == nil || key.KeyMetadata.Arn == nil {
		t.Fatalf("create KMS key: %v", err)
	}
	keyARN := aws.ToString(key.KeyMetadata.Arn)
	bucket := "accountable-contract-" + uuid.NewString()
	s3Client := s3.NewFromConfig(awsConfig, func(options *s3.Options) { options.UsePathStyle = true })
	if _, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = s3Client.DeleteBucket(cleanupCtx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	})
	setPublicAccessBlock := func(enabled bool) {
		t.Helper()
		_, err := s3Client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
			Bucket: aws.String(bucket),
			PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls: aws.Bool(enabled), IgnorePublicAcls: aws.Bool(enabled),
				BlockPublicPolicy: aws.Bool(enabled), RestrictPublicBuckets: aws.Bool(enabled),
			},
		})
		if err != nil {
			t.Fatalf("set public access block: %v", err)
		}
	}
	setPublicAccessBlock(true)
	if _, err := s3Client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucket),
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{{
				ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
					SSEAlgorithm: types.ServerSideEncryptionAwsKms, KMSMasterKeyID: aws.String(keyARN),
				},
			}},
		},
	}); err != nil {
		t.Fatalf("set bucket encryption: %v", err)
	}

	config := Config{
		Provider: ProviderS3, Region: "us-east-1", Bucket: bucket, Prefix: "application",
		ExpectedOwner: "000000000000", KMSKeyARN: keyARN, AccessPurpose: "foundation-proof",
	}
	store := NewS3Storage(config, awsConfig)
	if err := store.Check(ctx); err != nil {
		t.Fatalf("correctly configured S3 Check: %v", err)
	}
	if err := store.Probe(ctx); err != nil {
		t.Fatalf("correctly configured S3 Probe: %v", err)
	}
	setPublicAccessBlock(false)
	if err := store.Probe(ctx); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("public bucket Probe = %v, want storage unavailable", err)
	}
	setPublicAccessBlock(true)
	wrongKey := config
	wrongKey.KMSKeyARN += "-wrong"
	if err := NewS3Storage(wrongKey, awsConfig).Check(ctx); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("wrong-key Check = %v, want storage unavailable", err)
	}
	wrongOwner := config
	wrongOwner.ExpectedOwner = "111111111111"
	if err := NewS3Storage(wrongOwner, awsConfig).Probe(ctx); err == nil {
		t.Log("LocalStack does not enforce ExpectedBucketOwner; the unit fake retains that residual case")
	}
}

func localStackAWSConfig(endpoint string) aws.Config {
	return aws.Config{
		Region: "us-east-1", BaseEndpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentialsProvider("accountable-test", "accountable-test", ""),
		HTTPClient:  &http.Client{Timeout: 5 * time.Second}, RetryMaxAttempts: 1,
	}
}
