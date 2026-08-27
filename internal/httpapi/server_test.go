package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	New().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}
