package foundation

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalAdaptersRefuseMissingOrInvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := NewFileStorage(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("NewFileStorage error = %v", err)
	}
	if _, err := NewLocalCrypto(NewSecretValue([]byte("not-a-32-byte-base64-key"))); !errors.Is(err, ErrCryptoUnavailable) {
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
	cryptor, err := NewLocalCrypto(NewSecretValue(encoded))
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

func TestFileStoragePreflightLeavesNoArtifact(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf("NewFileStorage: %v", err)
	}
	if err := storage.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	entries, err := os.ReadDir(filepath.Clean(root))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("preflight artifacts = %v", entries)
	}
}
