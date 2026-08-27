package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsAndHealthLogExclusion(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	s, _ := testServer(t)
	s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	if logs.Len() != 0 {
		t.Fatalf("health was logged: %s", logs.String())
	}
	s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if logs.Len() != 0 {
		t.Fatalf("metrics was logged: %s", logs.String())
	}
	s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing", nil))
	if !strings.Contains(logs.String(), `"path":"/missing"`) {
		t.Fatalf("request not logged: %s", logs.String())
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(w.Body.String(), "url_shortener_http_requests_total") {
		t.Fatal("missing metrics")
	}
}
