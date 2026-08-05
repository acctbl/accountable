package crypto

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/acctbl/accountable/internal/platform/awsconfig"
)

func TestKMSAWSContract(t *testing.T) {
	region := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_AWS_REGION")
	keyARN := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_CRYPTO_KMS_KEY_ARN")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	awsConfig, err := awsconfig.LoadConfig(ctx, region)
	if err != nil {
		t.Fatalf("load AWS workload identity: %v", err)
	}
	cryptor, err := NewAWSCrypto(ctx, Config{
		Provider: ProviderAWSKMS, Region: region, KMSKeyARN: keyARN,
		EncryptionContextPrefix: "accountable.foundation",
	}, awsConfig)
	if err != nil {
		t.Fatalf("NewAWSCrypto: %v", err)
	}
	want := []byte("accountable-aws-kms-contract")
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
