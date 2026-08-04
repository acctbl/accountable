package crypto

import (
	"bytes"
	"context"
	"time"

	mpl "github.com/aws/aws-cryptographic-material-providers-library/releases/go/mpl/awscryptographymaterialproviderssmithygenerated"
	mpltypes "github.com/aws/aws-cryptographic-material-providers-library/releases/go/mpl/awscryptographymaterialproviderssmithygeneratedtypes"
	esdk "github.com/aws/aws-encryption-sdk/releases/go/encryption-sdk/awscryptographyencryptionsdksmithygenerated"
	esdktypes "github.com/aws/aws-encryption-sdk/releases/go/encryption-sdk/awscryptographyencryptionsdksmithygeneratedtypes"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

const managedCryptoOperationTimeout = 10 * time.Second

type encryptionEngine interface {
	Encrypt(context.Context, []byte) ([]byte, error)
	Decrypt(context.Context, []byte) ([]byte, error)
}

type AWSCrypto struct{ engine encryptionEngine }

func NewAWSCrypto(ctx context.Context, config Config, awsConfig aws.Config) (*AWSCrypto, error) {
	materials, err := mpl.NewClient(mpltypes.MaterialProvidersConfig{})
	if err != nil {
		return nil, ErrCryptoUnavailable
	}
	keyring, err := materials.CreateAwsKmsKeyring(ctx, mpltypes.CreateAwsKmsKeyringInput{
		KmsClient: kms.NewFromConfig(awsConfig),
		KmsKeyId:  config.KMSKeyARN,
	})
	if err != nil {
		return nil, ErrCryptoUnavailable
	}
	policy := mpltypes.ESDKCommitmentPolicyRequireEncryptRequireDecrypt
	maximumEncryptedDataKeys := int64(1)
	client, err := esdk.NewClient(esdktypes.AwsEncryptionSdkConfig{
		CommitmentPolicy:     &policy,
		MaxEncryptedDataKeys: &maximumEncryptedDataKeys,
	})
	if err != nil {
		return nil, ErrCryptoUnavailable
	}
	return newAWSCrypto(&awsEncryptionEngine{client: client, keyring: keyring}), nil
}

func newAWSCrypto(engine encryptionEngine) *AWSCrypto { return &AWSCrypto{engine: engine} }

func (c *AWSCrypto) Seal(plaintext []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), managedCryptoOperationTimeout)
	defer cancel()
	return c.seal(ctx, plaintext)
}

func (c *AWSCrypto) Open(ciphertext []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), managedCryptoOperationTimeout)
	defer cancel()
	return c.open(ctx, ciphertext)
}

func (c *AWSCrypto) Check(ctx context.Context) error {
	want := []byte("accountable-crypto-preflight")
	sealed, err := c.seal(ctx, want)
	if err != nil {
		return ErrCryptoUnavailable
	}
	opened, err := c.open(ctx, sealed)
	if err != nil || !bytes.Equal(opened, want) {
		return ErrCryptoUnavailable
	}
	return nil
}

func (c *AWSCrypto) seal(ctx context.Context, plaintext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, ErrCryptoUnavailable
	}
	sealed, err := c.engine.Encrypt(ctx, plaintext)
	if err != nil || len(sealed) == 0 {
		return nil, ErrCryptoUnavailable
	}
	return sealed, nil
}

func (c *AWSCrypto) open(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, ErrCryptoUnavailable
	}
	opened, err := c.engine.Decrypt(ctx, ciphertext)
	if err != nil {
		return nil, ErrCryptoUnavailable
	}
	return opened, nil
}

type encryptionSDKAPI interface {
	Encrypt(context.Context, esdktypes.EncryptInput) (*esdktypes.EncryptOutput, error)
	Decrypt(context.Context, esdktypes.DecryptInput) (*esdktypes.DecryptOutput, error)
}

type awsEncryptionEngine struct {
	client  encryptionSDKAPI
	keyring mpltypes.IKeyring
}

func (e *awsEncryptionEngine) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	output, err := e.client.Encrypt(ctx, esdktypes.EncryptInput{
		Plaintext: plaintext, EncryptionContext: encryptionContext(), Keyring: e.keyring,
	})
	if err != nil || output == nil {
		return nil, ErrCryptoUnavailable
	}
	return output.Ciphertext, nil
}

func (e *awsEncryptionEngine) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	wantContext := encryptionContext()
	output, err := e.client.Decrypt(ctx, esdktypes.DecryptInput{
		Ciphertext: ciphertext, EncryptionContext: wantContext, Keyring: e.keyring,
	})
	if err != nil || output == nil || !sameEncryptionContext(output.EncryptionContext, wantContext) {
		return nil, ErrCryptoUnavailable
	}
	return output.Plaintext, nil
}

func encryptionContext() map[string]string {
	return map[string]string{"application": "accountable", "purpose": "foundation"}
}

func sameEncryptionContext(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for key, value := range want {
		if got[key] != value {
			return false
		}
	}
	return true
}
