package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLog(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/geocode", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := accessLog(&buf)(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?suburb=Melbourne&state=VIC", nil)
	req.Header.Set("User-Agent", "test-client")
	req.RemoteAddr = "203.0.113.10:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	line := strings.TrimSpace(buf.String())
	if !strings.Contains(line, "GET /v1/geocode?suburb=Melbourne&state=VIC 200") {
		t.Fatalf("log line: %q", line)
	}
	if !strings.Contains(line, "203.0.113.10") {
		t.Fatalf("missing client IP: %q", line)
	}
	if !strings.Contains(line, "test-client") {
		t.Fatalf("missing user agent: %q", line)
	}
}

func TestAccessLogSkipsHealth(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := accessLog(&buf)(mux)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if buf.Len() != 0 {
		t.Fatalf("expected no log for /health, got %q", buf.String())
	}
}

func TestAccessLogForwardedFor(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/test", func(w http.ResponseWriter, r *http.Request) {})

	handler := accessLog(&buf)(mux)
	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.1")
	req.RemoteAddr = "10.0.0.5:8080"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "198.51.100.1") {
		t.Fatalf("expected forwarded IP: %q", buf.String())
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:443"
	if clientIP(req) != "192.0.2.1" {
		t.Fatalf("got %q", clientIP(req))
	}
}
