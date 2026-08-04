package foundation

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

type s3API interface {
	GetPublicAccessBlock(context.Context, *s3.GetPublicAccessBlockInput, ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error)
	GetBucketEncryption(context.Context, *s3.GetBucketEncryptionInput, ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type S3Storage struct {
	config StorageConfig
	client s3API
}

func NewS3Storage(config StorageConfig, awsConfig aws.Config) *S3Storage {
	return newS3Storage(config, s3.NewFromConfig(awsConfig))
}

func newS3Storage(config StorageConfig, client s3API) *S3Storage {
	return &S3Storage{config: config, client: client}
}

func (s *S3Storage) Check(ctx context.Context) error {
	publicAccess, err := s.client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket:              aws.String(s.config.Bucket),
		ExpectedBucketOwner: aws.String(s.config.ExpectedOwner),
	})
	if err != nil || !blocksAllPublicAccess(publicAccess) {
		return ErrStorageUnavailable
	}
	encryption, err := s.client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket:              aws.String(s.config.Bucket),
		ExpectedBucketOwner: aws.String(s.config.ExpectedOwner),
	})
	if err != nil || !usesConfiguredKMSKey(encryption, s.config.KMSKeyARN) {
		return ErrStorageUnavailable
	}

	want := []byte("accountable-storage-preflight")
	key := strings.TrimSuffix(s.config.Prefix, "/") + "/.accountable-preflight/" + uuid.NewString()
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(s.config.Bucket),
		Key:                  aws.String(key),
		Body:                 bytes.NewReader(want),
		ContentLength:        aws.Int64(int64(len(want))),
		ExpectedBucketOwner:  aws.String(s.config.ExpectedOwner),
		ServerSideEncryption: types.ServerSideEncryptionAwsKms,
		SSEKMSKeyId:          aws.String(s.config.KMSKeyARN),
	})
	if err != nil {
		return ErrStorageUnavailable
	}
	clean := func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), managedAWSRequestTimeout)
		defer cancel()
		_, deleteErr := s.client.DeleteObject(cleanupCtx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.config.Bucket), Key: aws.String(key),
			ExpectedBucketOwner: aws.String(s.config.ExpectedOwner),
		})
		return deleteErr
	}
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket), Key: aws.String(key),
		ExpectedBucketOwner: aws.String(s.config.ExpectedOwner),
	})
	if err != nil {
		_ = clean()
		return ErrStorageUnavailable
	}
	got, readErr := io.ReadAll(io.LimitReader(object.Body, int64(len(want)+1)))
	closeErr := object.Body.Close()
	deleteErr := clean()
	if readErr != nil || closeErr != nil || deleteErr != nil || !bytes.Equal(got, want) {
		return ErrStorageUnavailable
	}
	return nil
}

func blocksAllPublicAccess(output *s3.GetPublicAccessBlockOutput) bool {
	if output == nil || output.PublicAccessBlockConfiguration == nil {
		return false
	}
	config := output.PublicAccessBlockConfiguration
	return aws.ToBool(config.BlockPublicAcls) && aws.ToBool(config.IgnorePublicAcls) &&
		aws.ToBool(config.BlockPublicPolicy) && aws.ToBool(config.RestrictPublicBuckets)
}

func usesConfiguredKMSKey(output *s3.GetBucketEncryptionOutput, keyARN string) bool {
	if output == nil || output.ServerSideEncryptionConfiguration == nil {
		return false
	}
	for _, rule := range output.ServerSideEncryptionConfiguration.Rules {
		defaults := rule.ApplyServerSideEncryptionByDefault
		if defaults != nil && defaults.SSEAlgorithm == types.ServerSideEncryptionAwsKms &&
			aws.ToString(defaults.KMSMasterKeyID) == keyARN {
			return true
		}
	}
	return false
}
