package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/shinkeonkim/url-shortener/internal/store"
)

func testServer(t *testing.T) (*Server, *store.SQLite) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db).WithBaseDomain("url.test"), db
}
func TestHealth(t *testing.T) {
	s, _ := testServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}
func TestRedirectStatsAndQR(t *testing.T) {
	s, db := testServer(t)
	if _, err := db.CreateURL(context.Background(), "go-docs", "https://go.dev/doc/"); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/r/go-docs", nil)
	r.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "https://go.dev/doc/" {
		t.Fatalf("redirect: %d %s", w.Code, w.Header().Get("Location"))
	}
	stats, err := db.Stats(context.Background(), "go-docs", 10)
	if err != nil || stats.URL.Clicks != 1 || stats.Recent[0].UserAgent != "test-agent" {
		t.Fatalf("stats: %#v %v", stats, err)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/urls/go-docs/qr", nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" || !bytes.HasPrefix(w.Body.Bytes(), []byte("\x89PNG")) {
		t.Fatal("invalid QR response")
	}
}
func TestSubdomainAndMissing(t *testing.T) {
	s, db := testServer(t)
	db.CreateURL(context.Background(), "demo", "https://example.com")
	r := httptest.NewRequest(http.MethodGet, "http://demo.url.test/", nil)
	r.Host = "demo.url.test"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/r/missing", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d", w.Code)
	}
}
