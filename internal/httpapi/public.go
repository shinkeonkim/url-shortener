package httpapi

import (
	"net/http"
	"net/url"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	s.redirectSlug(w, r, r.PathValue("slug"))
}
func (s *Server) subdomainRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	slug := slugFromHost(r.Host, s.baseDomain)
	if slug == "" || strings.Contains(slug, ".") {
		http.NotFound(w, r)
		return
	}
	s.redirectSlug(w, r, slug)
}
func (s *Server) redirectSlug(w http.ResponseWriter, r *http.Request, slug string) {
	if !validSlug(slug) {
		writeError(w, http.StatusBadRequest, "invalid slug")
		return
	}
	u, err := s.repository.GetURL(r.Context(), slug)
	if err != nil {
		storeError(w, err)
		return
	}
	if err := s.repository.RecordClick(r.Context(), slug, limited(r.Referer(), 500), limited(r.UserAgent(), 500)); err != nil {
		storeError(w, err)
		return
	}
	http.Redirect(w, r, u.TargetURL, http.StatusFound)
}
func (s *Server) getURL(w http.ResponseWriter, r *http.Request) {
	u, err := s.repository.GetURL(r.Context(), r.PathValue("slug"))
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": u, "short_url": shortURL(r.PathValue("slug"), s.baseDomain)})
}
func (s *Server) qrCode(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if _, err := s.repository.GetURL(r.Context(), slug); err != nil {
		storeError(w, err)
		return
	}
	png, err := qrcode.Encode(shortURL(slug, s.baseDomain), qrcode.Medium, 256)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create QR code")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(png)
}
func shortURL(slug, domain string) string {
	return (&url.URL{Scheme: "https", Host: slug + "." + domain}).String()
}
func limited(value string, max int) string {
	if len(value) > max {
		return value[:max]
	}
	return value
}
func validSlug(slug string) bool {
	if len(slug) < 1 || len(slug) > 63 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	for _, r := range slug {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}
