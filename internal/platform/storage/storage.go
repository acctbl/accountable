package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
)

var ErrStorageUnavailable = errors.New("storage is unavailable")

type Storage interface {
	Check(context.Context) error
}

type FileStorage struct{ root string }

func NewFileStorage(root string) (*FileStorage, error) {
	if !filepath.IsAbs(root) {
		return nil, ErrStorageUnavailable
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrStorageUnavailable
	}
	return &FileStorage{root: filepath.Clean(root)}, nil
}

func (s *FileStorage) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.CreateTemp(s.root, ".accountable-preflight-*")
	if err != nil {
		return ErrStorageUnavailable
	}
	path := file.Name()
	defer func() { _ = os.Remove(path) }()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return ErrStorageUnavailable
	}
	want := []byte("accountable-storage-preflight")
	if _, err := file.Write(want); err != nil {
		_ = file.Close()
		return ErrStorageUnavailable
	}
	if err := file.Close(); err != nil {
		return ErrStorageUnavailable
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		return ErrStorageUnavailable
	}
	if err := os.Remove(path); err != nil {
		return ErrStorageUnavailable
	}
	return nil
}
