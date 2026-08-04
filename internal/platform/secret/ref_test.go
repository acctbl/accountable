package secret_test

import (
	"testing"

	"github.com/acctbl/accountable/internal/platform/secret"
)

func TestParseRefRejectsSecretMaterialAndTraversal(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "../database", "/database", "postgres://user:password@host/db", "has space"} {
		if _, err := secret.ParseRef(value); err == nil {
			t.Errorf("ParseRef(%q) succeeded", value)
		}
	}
}

func TestParseRefAcceptsOpaqueReference(t *testing.T) {
	t.Parallel()

	ref, err := secret.ParseRef("database.api_dsn")
	if err != nil {
		t.Fatalf("ParseRef: %v", err)
	}
	if ref != "database.api_dsn" {
		t.Fatalf("ref = %q", ref)
	}
}
