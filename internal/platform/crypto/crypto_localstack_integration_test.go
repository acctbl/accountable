package crypto

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

func TestKMSLocalStackContract(t *testing.T) {
	endpoint := os.Getenv("ACCOUNTABLE_TEST_AWS_ENDPOINT")
	if endpoint == "" {
		if os.Getenv("ACCOUNTABLE_REQUIRE_LOCALSTACK") == "1" {
			t.Fatal("ACCOUNTABLE_TEST_AWS_ENDPOINT is required")
		}
		t.Skip("run through task test:aws-contract")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	awsConfig := aws.Config{
		Region: "us-east-1", BaseEndpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentialsProvider("accountable-test", "accountable-test", ""),
		HTTPClient:  &http.Client{Timeout: 5 * time.Second}, RetryMaxAttempts: 1,
	}
	kmsClient := kms.NewFromConfig(awsConfig)
	key, err := kmsClient.CreateKey(ctx, &kms.CreateKeyInput{})
	if err != nil || key.KeyMetadata == nil || key.KeyMetadata.Arn == nil {
		t.Fatalf("create KMS key: %v", err)
	}
	keyARN := aws.ToString(key.KeyMetadata.Arn)
	cryptor, err := NewAWSCrypto(ctx, Config{
		Provider: ProviderAWSKMS, Region: "us-east-1", KMSKeyARN: keyARN,
		EncryptionContextPrefix: "accountable.foundation",
	}, awsConfig)
	if err != nil {
		t.Fatalf("NewAWSCrypto: %v", err)
	}
	want := []byte("accountable-localstack-kms-contract")
	sealed, err := cryptor.Seal(want)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	opened, err := cryptor.Open(sealed)
	if err != nil || !bytes.Equal(opened, want) {
		t.Fatalf("Open = (%q, %v), want round trip", opened, err)
	}
	if err := cryptor.Probe(ctx); err != nil {
		t.Fatalf("enabled key Probe: %v", err)
	}
	if _, err := kmsClient.DisableKey(ctx, &kms.DisableKeyInput{KeyId: aws.String(keyARN)}); err != nil {
		t.Fatalf("disable key: %v", err)
	}
	if err := cryptor.Probe(ctx); !errors.Is(err, ErrCryptoUnavailable) {
		t.Fatalf("disabled key Probe = %v, want crypto unavailable", err)
	}
}
