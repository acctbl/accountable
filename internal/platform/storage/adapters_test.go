package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStorageRefuseMissingRoot(t *testing.T) {
	t.Parallel()

	if _, err := NewFileStorage(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("NewFileStorage error = %v", err)
	}
}

func TestFileStoragePreflightLeavesNoArtifact(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf("NewFileStorage: %v", err)
	}
	if err := store.Check(context.Background()); err != nil {
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
