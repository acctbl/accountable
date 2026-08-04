package crypto

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"

	"github.com/acctbl/accountable/internal/platform/secret"
)

var ErrCryptoUnavailable = errors.New("crypto is unavailable")

type Crypto interface {
	Seal([]byte) ([]byte, error)
	Open([]byte) ([]byte, error)
	Check(context.Context) error
	Probe(context.Context) error
}

type LocalCrypto struct {
	aead    cipher.AEAD
	keyPath string
}

func NewLocalCrypto(value secret.SecretValue, keyPath ...string) (*LocalCrypto, error) {
	encoded := value.Bytes()
	key := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	n, err := base64.StdEncoding.Decode(key, encoded)
	clear(encoded)
	if err != nil || n != 32 {
		clear(key)
		return nil, ErrCryptoUnavailable
	}
	key = key[:n]
	block, err := aes.NewCipher(key)
	clear(key)
	if err != nil {
		return nil, ErrCryptoUnavailable
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrCryptoUnavailable
	}
	path := ""
	if len(keyPath) > 0 {
		path = filepath.Clean(keyPath[0])
	}
	return &LocalCrypto{aead: aead, keyPath: path}, nil
}

func (c *LocalCrypto) Probe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.keyPath == "" {
		return nil
	}
	info, err := os.Lstat(c.keyPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return ErrCryptoUnavailable
	}
	return nil
}

func (c *LocalCrypto) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, ErrCryptoUnavailable
	}
	return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *LocalCrypto) Open(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.aead.NonceSize() {
		return nil, ErrCryptoUnavailable
	}
	nonce, payload := ciphertext[:c.aead.NonceSize()], ciphertext[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, ErrCryptoUnavailable
	}
	return plaintext, nil
}

func (c *LocalCrypto) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	want := []byte("accountable-crypto-preflight")
	sealed, err := c.Seal(want)
	if err != nil {
		return ErrCryptoUnavailable
	}
	opened, err := c.Open(sealed)
	if err != nil || !bytes.Equal(opened, want) {
		return ErrCryptoUnavailable
	}
	return nil
}
