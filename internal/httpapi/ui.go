package httpapi

import (
	"embed"
	"net/http"
)

//go:embed web/*
var web embed.FS

func securityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
}
func (s *Server) adminUI(w http.ResponseWriter, _ *http.Request) {
	securityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, _ := web.ReadFile("web/index.html")
	w.Write(data)
}
func (s *Server) asset(w http.ResponseWriter, r *http.Request) {
	securityHeaders(w)
	name := r.PathValue("name")
	if name != "app.js" && name != "style.css" {
		http.NotFound(w, r)
		return
	}
	data, err := web.ReadFile("web/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if name == "app.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	w.Write(data)
}
