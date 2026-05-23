package stoplist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreLoadMissingFileReturnsEmptyList(t *testing.T) {
	t.Parallel()

	store := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))

	terms, err := store.Load()
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}

	if len(terms) != 0 {
		t.Fatalf("expected empty terms, got %#v", terms)
	}
}

func TestFileStoreSaveAndLoad(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "data", "stoplist.json")
	store := NewFileStore(path)

	err := store.Save([]string{
		"  Casino   Online ",
		"casino online",
		"КАЗИНО",
		"",
	})
	if err != nil {
		t.Fatalf("failed to save stop-list: %v", err)
	}

	terms, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load stop-list: %v", err)
	}

	want := []string{"casino online", "казино"}

	if len(terms) != len(want) {
		t.Fatalf("terms length mismatch: got %d, want %d; terms=%#v", len(terms), len(want), terms)
	}

	for i := range want {
		if terms[i] != want[i] {
			t.Fatalf("term %d mismatch: got %q, want %q", i, terms[i], want[i])
		}
	}
}

func TestFileStoreLoadRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "stoplist.json")

	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid json: %v", err)
	}

	store := NewFileStore(path)

	_, err := store.Load()
	if err == nil {
		t.Fatalf("expected error for invalid json")
	}
}
