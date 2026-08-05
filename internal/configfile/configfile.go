package configfile

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

func AbsolutePath(args []string) (string, error) {
	if len(args) != 2 || args[0] != "--config" || args[1] == "" {
		return "", errors.New("usage: --config <absolute-path>")
	}
	if !filepath.IsAbs(args[1]) {
		return "", errors.New("--config path must be absolute")
	}
	return filepath.Clean(args[1]), nil
}

func Decode(path string, target any) error {
	_, err := DecodeWithFingerprint(path, target)
	return err
}

func DecodeWithFingerprint(path string, target any) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("open config: %w", err)
	}

	decoder := toml.NewDecoder(bytes.NewReader(contents)).DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return "", fmt.Errorf("decode config: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(contents)), nil
}
