package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	Redirects       atomic.Uint64
	mu              sync.Mutex
	requests        map[string]uint64
	durationSeconds float64
}

func NewMetrics() *Metrics { return &Metrics{requests: make(map[string]uint64)} }
func (m *Metrics) ObserveRequest(method string, status int, elapsed time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[method+"|"+strconv.Itoa(status)]++
	m.durationSeconds += elapsed.Seconds()
}
func (s *Server) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	m := s.metrics
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Fprintln(w, "# HELP url_shortener_http_requests_total HTTP requests.")
	fmt.Fprintln(w, "# TYPE url_shortener_http_requests_total counter")
	for key, count := range m.requests {
		method, status, _ := strings.Cut(key, "|")
		fmt.Fprintf(w, "url_shortener_http_requests_total{method=%q,status=%q} %d\n", method, status, count)
	}
	fmt.Fprintln(w, "# HELP url_shortener_redirects_total Successful redirects.")
	fmt.Fprintln(w, "# TYPE url_shortener_redirects_total counter")
	fmt.Fprintf(w, "url_shortener_redirects_total %d\n", m.Redirects.Load())
	fmt.Fprintln(w, "# HELP url_shortener_http_request_duration_seconds_sum Total request time.")
	fmt.Fprintln(w, "# TYPE url_shortener_http_request_duration_seconds_sum counter")
	fmt.Fprintf(w, "url_shortener_http_request_duration_seconds_sum %f\n", m.durationSeconds)
}

func logRequest(r *http.Request, status int, elapsed time.Duration) {
	slog.Info("http request", "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", elapsed.Milliseconds())
}
