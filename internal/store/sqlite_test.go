package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestURLLifecycleAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateURL(context.Background(), "docs", "https://go.dev/doc/")
	if err != nil || created.Slug != "docs" {
		t.Fatalf("create: %#v %v", created, err)
	}
	if _, err := s.CreateURL(context.Background(), "docs", "https://example.com"); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.GetURL(context.Background(), "docs")
	if err != nil || got.TargetURL != "https://go.dev/doc/" {
		t.Fatalf("get: %#v %v", got, err)
	}
	if err := s.DeleteURL(context.Background(), "docs"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetURL(context.Background(), "docs"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}
