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

func TestFileStorageProbeRefusesUnwritableRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewFileStorage(root)
	if err != nil {
		t.Fatalf("NewFileStorage: %v", err)
	}
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatalf("make root unwritable: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	if err := store.Probe(context.Background()); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("Probe = %v, want storage unavailable", err)
	}
}
