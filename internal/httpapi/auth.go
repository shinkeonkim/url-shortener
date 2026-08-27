package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthConfig struct {
	Username, PasswordHash, Token, SessionKey string
	CookieSecure                              bool
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	userOK := subtle.ConstantTimeCompare([]byte(body.Username), []byte(s.auth.Username)) == 1
	passOK := bcrypt.CompareHashAndPassword([]byte(s.auth.PasswordHash), []byte(body.Password)) == nil
	if !userOK || !passOK || s.auth.SessionKey == "" {
		writeError(w, 401, "invalid credentials")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: s.session(), Path: "/", HttpOnly: true, Secure: s.auth.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: 8 * 3600})
	writeJSON(w, http.StatusOK, map[string]string{"status": "authenticated"})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeError(w, 401, "authentication required")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Path: "/", HttpOnly: true, Secure: s.auth.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authorized(r *http.Request) bool {
	if token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); token != "" && s.auth.Token != "" {
		return subtle.ConstantTimeCompare([]byte(token), []byte(s.auth.Token)) == 1
	}
	cookie, err := r.Cookie("admin_session")
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	expires, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return false
	}
	return hmac.Equal([]byte(parts[1]), []byte(s.sign(parts[0])))
}

func (s *Server) session() string {
	expires := strconv.FormatInt(time.Now().Add(8*time.Hour).Unix(), 10)
	return expires + "." + s.sign(expires)
}
func (s *Server) sign(value string) string {
	mac := hmac.New(sha256.New, []byte(s.auth.SessionKey))
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
