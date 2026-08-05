package storage

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/acctbl/accountable/internal/platform/awsconfig"
)

func TestS3AWSContract(t *testing.T) {
	region := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_AWS_REGION")
	accountID := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_AWS_ACCOUNT_ID")
	secureBucket := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_S3_BUCKET")
	insecureBucket := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_S3_INSECURE_BUCKET")
	storageKeyARN := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_STORAGE_KMS_KEY_ARN")
	cryptoKeyARN := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_CRYPTO_KMS_KEY_ARN")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	awsConfig, err := awsconfig.LoadConfig(ctx, region)
	if err != nil {
		t.Fatalf("load AWS workload identity: %v", err)
	}
	config := Config{
		Provider: ProviderS3, Region: region, Bucket: secureBucket, Prefix: "contract",
		ExpectedOwner: accountID, KMSKeyARN: storageKeyARN, AccessPurpose: "foundation-proof",
	}
	store := NewS3Storage(config, awsConfig)
	if err := store.Check(ctx); err != nil {
		t.Fatalf("correctly configured S3 Check: %v", err)
	}
	if err := store.Probe(ctx); err != nil {
		t.Fatalf("correctly configured S3 Probe: %v", err)
	}

	insecure := config
	insecure.Bucket = insecureBucket
	if err := NewS3Storage(insecure, awsConfig).Probe(ctx); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("public-access-block refusal Probe = %v, want storage unavailable", err)
	}

	wrongKey := config
	wrongKey.KMSKeyARN = cryptoKeyARN
	if err := NewS3Storage(wrongKey, awsConfig).Check(ctx); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("wrong-key Check = %v, want storage unavailable", err)
	}

	wrongOwner := config
	wrongOwner.ExpectedOwner = differentAWSAccountID(accountID)
	if err := NewS3Storage(wrongOwner, awsConfig).Probe(ctx); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("wrong-owner Probe = %v, want storage unavailable", err)
	}
}

func managedContractValue(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value != "" {
		return value
	}
	if os.Getenv("ACCOUNTABLE_REQUIRE_MANAGED_CONTRACT") == "1" {
		t.Fatalf("%s is required", name)
	}
	t.Skip("run through task test:managed-contract")
	return ""
}

func differentAWSAccountID(accountID string) string {
	if accountID == "000000000000" {
		return "111111111111"
	}
	return "000000000000"
}
