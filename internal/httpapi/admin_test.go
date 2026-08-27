package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func authenticatedServer(t *testing.T) *Server {
	s, _ := testServer(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret-password"), bcrypt.MinCost)
	return s.WithAuth(AuthConfig{Username: "admin", PasswordHash: string(hash), Token: "api-token", SessionKey: "a-test-key-long-enough", CookieSecure: false})
}
func TestAdminBearerCRUD(t *testing.T) {
	s := authenticatedServer(t)
	body := `{"slug":"project","target_url":"https://example.com/path"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/urls", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer api-token")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/urls/project/stats", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized = %d", w.Code)
	}
}
func TestLoginSession(t *testing.T) {
	s := authenticatedServer(t)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret-password"}`))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK || len(w.Result().Cookies()) != 1 {
		t.Fatalf("login = %d", w.Code)
	}
	r = httptest.NewRequest(http.MethodGet, "/api/v1/urls/missing/stats", nil)
	r.AddCookie(w.Result().Cookies()[0])
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("session auth = %d", w.Code)
	}
}
