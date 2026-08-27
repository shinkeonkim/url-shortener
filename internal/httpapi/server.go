package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/shinkeonkim/url-shortener/internal/store"
)

type Repository interface {
	CreateURL(context.Context, string, string) (store.URL, error)
	GetURL(context.Context, string) (store.URL, error)
	DeleteURL(context.Context, string) error
	RecordClick(context.Context, string, string, string) error
	Stats(context.Context, string, int) (store.Stats, error)
}

type Server struct {
	handler    http.Handler
	repository Repository
	baseDomain string
	auth       AuthConfig
	metrics    *Metrics
}

func New(repository ...Repository) *Server {
	s := &Server{baseDomain: "url.shinkeonkim.com", metrics: NewMetrics()}
	if len(repository) > 0 {
		s.repository = repository[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /r/{slug}", s.redirect)
	mux.HandleFunc("GET /api/v1/urls/{slug}", s.getURL)
	mux.HandleFunc("GET /api/v1/urls/{slug}/qr", s.qrCode)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("POST /api/v1/urls", s.createURL)
	mux.HandleFunc("DELETE /api/v1/urls/{slug}", s.deleteURL)
	mux.HandleFunc("GET /api/v1/urls/{slug}/stats", s.getStats)
	mux.HandleFunc("GET /admin", s.adminUI)
	mux.HandleFunc("GET /assets/{name}", s.asset)
	mux.HandleFunc("GET /metrics", s.metricsHandler)
	mux.HandleFunc("GET /", s.subdomainRedirect)
	s.handler = s.observe(mux)
	return s
}

func (s *Server) WithBaseDomain(domain string) *Server             { s.baseDomain = domain; return s }
func (s *Server) WithAuth(auth AuthConfig) *Server                 { s.auth = auth; return s }
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		if rw.status == 0 {
			rw.status = http.StatusOK
		}
		s.metrics.ObserveRequest(r.Method, rw.status, time.Since(started))
		if r.URL.Path != "/health" {
			logRequest(r, rw.status, time.Since(started))
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "short URL not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
func slugFromHost(host, domain string) string {
	host = strings.Split(host, ":")[0]
	suffix := "." + domain
	if !strings.HasSuffix(host, suffix) {
		return ""
	}
	return strings.TrimSuffix(host, suffix)
}
