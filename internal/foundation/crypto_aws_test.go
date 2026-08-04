package foundation

import (
	"context"
	"errors"
	"testing"
)

type fakeEncryptionEngine struct {
	fail   bool
	tamper bool
	prefix []byte
}

func (f fakeEncryptionEngine) Encrypt(_ context.Context, plaintext []byte) ([]byte, error) {
	if f.fail {
		return nil, errors.New("KMS unavailable")
	}
	return append(append([]byte(nil), f.prefix...), plaintext...), nil
}

func (f fakeEncryptionEngine) Decrypt(_ context.Context, ciphertext []byte) ([]byte, error) {
	if f.fail || len(ciphertext) < len(f.prefix) {
		return nil, errors.New("decrypt failed")
	}
	plaintext := append([]byte(nil), ciphertext[len(f.prefix):]...)
	if f.tamper && len(plaintext) > 0 {
		plaintext[0] ^= 0xff
	}
	return plaintext, nil
}

func TestAWSCryptoPreflightRoundTripsThrowawayData(t *testing.T) {
	t.Parallel()

	cryptor := newAWSCrypto(fakeEncryptionEngine{prefix: []byte("ciphertext:")})
	if err := cryptor.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestAWSCryptoPreflightRefusesProviderFailureAndMismatch(t *testing.T) {
	t.Parallel()

	for _, engine := range []fakeEncryptionEngine{
		{fail: true},
		{prefix: []byte("ciphertext:"), tamper: true},
	} {
		if err := newAWSCrypto(engine).Check(context.Background()); !errors.Is(err, ErrCryptoUnavailable) {
			t.Fatalf("Check = %v, want crypto unavailable", err)
		}
	}
}
