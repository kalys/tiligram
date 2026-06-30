package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestCORS_allowedOrigin(t *testing.T) {
	h := CORS([]string{"https://osmonov.com"}, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.Header.Set("Origin", "https://osmonov.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://osmonov.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://osmonov.com")
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected handler to be called")
	}
}

func TestCORS_disallowedOrigin(t *testing.T) {
	h := CORS([]string{"https://osmonov.com"}, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected handler still to run for non-preflight, got %d", w.Code)
	}
}

func TestCORS_noOriginHeader(t *testing.T) {
	h := CORS([]string{"https://osmonov.com"}, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCORS_preflightAllowed(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	})
	h := CORS([]string{"https://osmonov.com"}, next)

	req := httptest.NewRequest(http.MethodOptions, "/search", nil)
	req.Header.Set("Origin", "https://osmonov.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called on preflight")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://osmonov.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Errorf("Access-Control-Allow-Headers = %q", got)
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("Access-Control-Max-Age = %q", got)
	}
}

func TestCORS_preflightDisallowedOrigin(t *testing.T) {
	h := CORS([]string{"https://osmonov.com"}, okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/search", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}
