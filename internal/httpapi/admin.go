package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/shinkeonkim/url-shortener/internal/store"
)

func (s *Server) createURL(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var body struct {
		Slug      string `json:"slug"`
		TargetURL string `json:"target_url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	body.Slug = strings.ToLower(strings.TrimSpace(body.Slug))
	if !validSlug(body.Slug) {
		writeError(w, 422, "slug must be 1-63 lowercase letters, numbers, or hyphens")
		return
	}
	if !validTarget(body.TargetURL) {
		writeError(w, 422, "target_url must be an absolute http(s) URL")
		return
	}
	u, err := s.repository.CreateURL(r.Context(), body.Slug, body.TargetURL)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, 409, "slug already exists")
		return
	}
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"url": u, "short_url": shortURL(u.Slug, s.baseDomain)})
}
func (s *Server) deleteURL(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if err := s.repository.DeleteURL(r.Context(), r.PathValue("slug")); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	stats, err := s.repository.Stats(r.Context(), r.PathValue("slug"), 100)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if s.authorized(r) {
		return true
	}
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeError(w, http.StatusUnauthorized, "authentication required")
	return false
}
func validTarget(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}
