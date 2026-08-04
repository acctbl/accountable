package awsconfig_test

import (
	"testing"

	"github.com/acctbl/accountable/internal/platform/awsconfig"
)

func TestSafeCredentialSourceRejectsStaticCredentials(t *testing.T) {
	t.Parallel()

	for _, source := range []string{"EnvConfigCredentials", "SharedConfigCredentials: /tmp/credentials", "StaticCredentials"} {
		if awsconfig.SafeCredentialSource(source) {
			t.Errorf("SafeCredentialSource(%q) = true", source)
		}
	}
	if !awsconfig.SafeCredentialSource("WebIdentityCredentials") {
		t.Fatal("workload identity credentials were rejected")
	}
}
