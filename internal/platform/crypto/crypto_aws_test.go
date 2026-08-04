package crypto

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"testing"

	esdktypes "github.com/aws/aws-encryption-sdk/releases/go/encryption-sdk/awscryptographyencryptionsdksmithygeneratedtypes"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type fakeEncryptionEngine struct {
	fail   bool
	tamper bool
	prefix []byte
}

type fakeKMS struct {
	enabled bool
	state   types.KeyState
	err     error
}

func (f fakeKMS) DescribeKey(context.Context, *kms.DescribeKeyInput, ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &kms.DescribeKeyOutput{KeyMetadata: &types.KeyMetadata{Enabled: f.enabled, KeyState: f.state}}, nil
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

type fakeEncryptionSDK struct {
	decryptContext func(requested map[string]string) map[string]string
}

func (f fakeEncryptionSDK) Encrypt(_ context.Context, input esdktypes.EncryptInput) (*esdktypes.EncryptOutput, error) {
	return &esdktypes.EncryptOutput{
		Ciphertext:        append([]byte("sealed:"), input.Plaintext...),
		EncryptionContext: input.EncryptionContext,
	}, nil
}

func (f fakeEncryptionSDK) Decrypt(_ context.Context, input esdktypes.DecryptInput) (*esdktypes.DecryptOutput, error) {
	return &esdktypes.DecryptOutput{
		Plaintext:         bytes.TrimPrefix(input.Ciphertext, []byte("sealed:")),
		EncryptionContext: f.decryptContext(input.EncryptionContext),
	}, nil
}

func TestAWSCryptoAcceptsSignedSuiteReservedContextPair(t *testing.T) {
	t.Parallel()

	engine := &awsEncryptionEngine{client: fakeEncryptionSDK{
		decryptContext: func(requested map[string]string) map[string]string {
			stored := maps.Clone(requested)
			stored["aws-crypto-public-key"] = "AmFsZ29yaXRobS1zaWduaW5nLWtleQ=="
			return stored
		},
	}}
	if err := newAWSCrypto(engine).Check(context.Background()); err != nil {
		t.Fatalf("Check = %v, want signed-suite ciphertext accepted", err)
	}
}

func TestAWSCryptoRefusesMissingOrMismatchedContextPair(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(map[string]string) map[string]string{
		"missing pair": func(requested map[string]string) map[string]string {
			stored := maps.Clone(requested)
			delete(stored, "accountable.foundation.purpose")
			return stored
		},
		"mismatched value": func(requested map[string]string) map[string]string {
			stored := maps.Clone(requested)
			stored["accountable.foundation.application"] = "other"
			return stored
		},
	} {
		engine := &awsEncryptionEngine{client: fakeEncryptionSDK{decryptContext: mutate}}
		if err := newAWSCrypto(engine).Check(context.Background()); !errors.Is(err, ErrCryptoUnavailable) {
			t.Fatalf("%s: Check = %v, want crypto unavailable", name, err)
		}
	}
}

func TestAWSCryptoUsesConfiguredEncryptionContextPrefix(t *testing.T) {
	t.Parallel()

	engine := &awsEncryptionEngine{
		encryptionContextPrefix: "payroll.foundation",
		client: fakeEncryptionSDK{decryptContext: func(requested map[string]string) map[string]string {
			if requested["payroll.foundation.application"] != "accountable" ||
				requested["payroll.foundation.purpose"] != "foundation" || len(requested) != 2 {
				t.Fatalf("encryption context = %v", requested)
			}
			return maps.Clone(requested)
		}},
	}
	if err := newAWSCrypto(engine).Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestAWSCryptoProbeRequiresEnabledKey(t *testing.T) {
	t.Parallel()

	for name, client := range map[string]fakeKMS{
		"disabled":         {enabled: false, state: types.KeyStateDisabled},
		"pending deletion": {enabled: true, state: types.KeyStatePendingDeletion},
		"provider failure": {err: errors.New("KMS unavailable")},
	} {
		cryptor := &AWSCrypto{kms: client, keyID: "key-id"}
		if err := cryptor.Probe(context.Background()); !errors.Is(err, ErrCryptoUnavailable) {
			t.Errorf("%s: Probe = %v, want crypto unavailable", name, err)
		}
	}
	cryptor := &AWSCrypto{kms: fakeKMS{enabled: true, state: types.KeyStateEnabled}, keyID: "key-id"}
	if err := cryptor.Probe(context.Background()); err != nil {
		t.Fatalf("enabled key Probe: %v", err)
	}
}
