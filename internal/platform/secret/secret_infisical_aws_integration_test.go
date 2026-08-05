package secret

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/acctbl/accountable/internal/platform/clock"
)

func TestInfisicalAWSContract(t *testing.T) {
	region := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_AWS_REGION")
	projectID := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_INFISICAL_PROJECT_ID")
	identityID := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_INFISICAL_MACHINE_IDENTITY_ID")
	secretPath := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_INFISICAL_SECRET_PATH")
	secretName := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_INFISICAL_SECRET_NAME")
	wantDigest := managedContractValue(t, "ACCOUNTABLE_MANAGED_CONTRACT_INFISICAL_SECRET_SHA256")
	if decoded, err := hex.DecodeString(wantDigest); err != nil || len(decoded) != sha256.Size {
		t.Fatal("ACCOUNTABLE_MANAGED_CONTRACT_INFISICAL_SECRET_SHA256 must be a SHA-256 hex digest")
	}
	ref, err := ParseRef(secretName)
	if err != nil {
		t.Fatalf("parse proof secret reference: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	source, err := NewSource(ctx, Config{
		Provider: ProviderInfisical, SiteURL: InfisicalCloudEUEndpoint, AWSRegion: region,
		ProjectID: projectID, Environment: "development", SecretPath: secretPath,
		AuthMethod: AuthAWSIAM, MachineIdentityID: identityID,
	}, clock.System{})
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	values, err := source.ResolveBatch(ctx, []Ref{ref})
	if err != nil {
		t.Fatalf("ResolveBatch: %v", err)
	}
	value := values[ref].Bytes()
	digest := sha256.Sum256(value)
	clear(value)
	if hex.EncodeToString(digest[:]) != wantDigest {
		t.Fatal("resolved proof secret digest does not match")
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
