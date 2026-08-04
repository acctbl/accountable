package crypto

import (
	"context"
	"encoding/base64"
	"errors"
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
