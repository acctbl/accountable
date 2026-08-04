package crypto

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/acctbl/accountable/internal/platform/secret"
)

func TestLocalCryptoRefuseInvalidKey(t *testing.T) {
	t.Parallel()

	if _, err := NewLocalCrypto(secret.NewSecretValue([]byte("not-a-32-byte-base64-key"))); !errors.Is(err, ErrCryptoUnavailable) {
		t.Fatalf("NewLocalCrypto error = %v", err)
	}
}

func TestLocalCryptoPreflightAndTamperRefusal(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index + 1)
	}
	encoded := []byte(base64.StdEncoding.EncodeToString(key))
	cryptor, err := NewLocalCrypto(secret.NewSecretValue(encoded))
	if err != nil {
		t.Fatalf("NewLocalCrypto: %v", err)
	}
	if err := cryptor.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	sealed, err := cryptor.Seal([]byte("synthetic"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	sealed[len(sealed)-1] ^= 0xff
	if _, err := cryptor.Open(sealed); err == nil {
		t.Fatal("Open accepted tampered ciphertext")
	}
}

func TestLocalCryptoProbeRefusesMissingOrUnsafeKeyFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	keyPath := filepath.Join(root, "crypto.key")
	key := secret.NewSecretValue([]byte(base64.StdEncoding.EncodeToString(make([]byte, 32))))
	cryptor, err := NewLocalCrypto(key, keyPath)
	if err != nil {
		t.Fatalf("NewLocalCrypto: %v", err)
	}
	if err := cryptor.Probe(context.Background()); !errors.Is(err, ErrCryptoUnavailable) {
		t.Fatalf("missing key Probe = %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o644); err != nil {
		t.Fatalf("write unsafe key: %v", err)
	}
	if err := cryptor.Probe(context.Background()); !errors.Is(err, ErrCryptoUnavailable) {
		t.Fatalf("unsafe key Probe = %v", err)
	}
	if err := os.Chmod(keyPath, 0o600); err != nil {
		t.Fatalf("secure key: %v", err)
	}
	if err := cryptor.Probe(context.Background()); err != nil {
		t.Fatalf("safe key Probe: %v", err)
	}
}
