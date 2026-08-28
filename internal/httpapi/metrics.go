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

var durationBounds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

type Metrics struct {
	Redirects       atomic.Uint64
	mu              sync.Mutex
	requests        map[string]uint64
	durationBuckets []uint64
	durationCount   uint64
	durationSum     float64
}

func NewMetrics() *Metrics {
	return &Metrics{requests: make(map[string]uint64), durationBuckets: make([]uint64, len(durationBounds))}
}

func (m *Metrics) ObserveRequest(method, route string, status int, elapsed time.Duration) {
	if route == "" {
		route = "unmatched"
	}
	seconds := elapsed.Seconds()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[strings.Join([]string{method, route, strconv.Itoa(status)}, "|")]++
	m.durationCount++
	m.durationSum += seconds
	for index, bound := range durationBounds {
		if seconds <= bound {
			m.durationBuckets[index]++
		}
	}
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	s.writeRequestMetrics(w)
	stats, err := s.repository.Overview(r.Context())
	if err != nil {
		slog.Error("metrics storage overview failed", "error", err)
		return
	}
	fmt.Fprintln(w, "# HELP url_shortener_storage_items Current SQLite item counts.")
	fmt.Fprintln(w, "# TYPE url_shortener_storage_items gauge")
	fmt.Fprintf(w, "url_shortener_storage_items{kind=\"urls\"} %d\n", stats.URLs)
	fmt.Fprintf(w, "url_shortener_storage_items{kind=\"total_clicks\"} %d\n", stats.TotalClicks)
	fmt.Fprintf(w, "url_shortener_storage_items{kind=\"raw_clicks\"} %d\n", stats.RawClicks)
	fmt.Fprintf(w, "url_shortener_storage_items{kind=\"daily_rollups\"} %d\n", stats.DailyRollups)
	fmt.Fprintf(w, "url_shortener_storage_items{kind=\"monthly_rollups\"} %d\n", stats.MonthRollups)
	clicks, err := s.repository.URLClickStats(r.Context())
	if err != nil {
		slog.Error("metrics URL clicks failed", "error", err)
		return
	}
	fmt.Fprintln(w, "# HELP url_shortener_url_clicks Stored clicks by short URL.")
	fmt.Fprintln(w, "# TYPE url_shortener_url_clicks gauge")
	for _, item := range clicks {
		fmt.Fprintf(w, "url_shortener_url_clicks{slug=%q} %d\n", item.Slug, item.Clicks)
	}
}

func (s *Server) writeRequestMetrics(w http.ResponseWriter) {
	m := s.metrics
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Fprintln(w, "# HELP url_shortener_http_requests_total HTTP requests by route and status.")
	fmt.Fprintln(w, "# TYPE url_shortener_http_requests_total counter")
	for key, count := range m.requests {
		parts := strings.SplitN(key, "|", 3)
		fmt.Fprintf(w, "url_shortener_http_requests_total{method=%q,route=%q,status=%q} %d\n", parts[0], parts[1], parts[2], count)
	}
	fmt.Fprintln(w, "# HELP url_shortener_http_request_duration_seconds HTTP request duration.")
	fmt.Fprintln(w, "# TYPE url_shortener_http_request_duration_seconds histogram")
	for index, bound := range durationBounds {
		fmt.Fprintf(w, "url_shortener_http_request_duration_seconds_bucket{le=%q} %d\n", strconv.FormatFloat(bound, 'f', -1, 64), m.durationBuckets[index])
	}
	fmt.Fprintf(w, "url_shortener_http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", m.durationCount)
	fmt.Fprintf(w, "url_shortener_http_request_duration_seconds_sum %f\n", m.durationSum)
	fmt.Fprintf(w, "url_shortener_http_request_duration_seconds_count %d\n", m.durationCount)
	fmt.Fprintln(w, "# HELP url_shortener_redirects_total Successful redirects.")
	fmt.Fprintln(w, "# TYPE url_shortener_redirects_total counter")
	fmt.Fprintf(w, "url_shortener_redirects_total %d\n", m.Redirects.Load())
}

func logRequest(r *http.Request, status int, elapsed time.Duration) {
	slog.Info("http request", "method", r.Method, "path", r.URL.Path, "route", r.Pattern, "status", status, "duration_ms", elapsed.Milliseconds())
}
