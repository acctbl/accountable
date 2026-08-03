package foundation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSecretResolverReturnsRedactedValue(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	secret := []byte("postgres://accountable:super-secret@localhost/accountable\n")
	if err := os.WriteFile(filepath.Join(directory, "database.api_dsn"), secret, 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	resolver, err := NewFileSecretResolver(directory)
	if err != nil {
		t.Fatalf("NewFileSecretResolver: %v", err)
	}
	value, err := resolver.Resolve(context.Background(), "database.api_dsn")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := string(value.Bytes()); got != strings.TrimSpace(string(secret)) {
		t.Fatalf("resolved value = %q", got)
	}
	if got := value.String(); got != "[REDACTED]" {
		t.Fatalf("String = %q", got)
	}
}

func TestFileSecretResolverRefusesMissingUnsafeOrEscapingFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "crypto.key")
	if err := os.WriteFile(path, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	resolver, err := NewFileSecretResolver(directory)
	if err != nil {
		t.Fatalf("NewFileSecretResolver: %v", err)
	}
	if _, err := resolver.Resolve(context.Background(), "crypto.key"); err == nil {
		t.Fatal("Resolve succeeded with group/world-readable secret")
	}
	if _, err := resolver.Resolve(context.Background(), "missing.key"); err == nil {
		t.Fatal("Resolve succeeded with missing secret")
	}

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "escaped.key"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside secret: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := resolver.Resolve(context.Background(), "escape/escaped.key"); err == nil {
		t.Fatal("Resolve followed an intermediate symlink outside the secret directory")
	}
}
