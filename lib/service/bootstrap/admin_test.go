package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNewAdminConfigUsesLegacyDefaultPort(t *testing.T) {
	cfg := NewAdminConfig("mainsvr", 2, true, false, "", 0)
	if !cfg.Enabled {
		t.Fatalf("expected admin server to be enabled")
	}
	if cfg.Port != 8102 {
		t.Fatalf("expected legacy default port 8102, got %d", cfg.Port)
	}
}

func TestNewAdminConfigCanStayDisabled(t *testing.T) {
	cfg := NewAdminConfig("infosvr", 3, false, false, "", 0)
	if cfg.Enabled {
		t.Fatalf("expected admin server to stay disabled")
	}
}

func TestAdminMuxEndpoints(t *testing.T) {
	var alive atomic.Bool
	var ready atomic.Bool
	alive.Store(true)
	ready.Store(true)

	mux := newAdminMux(AdminConfig{
		Enabled:     true,
		ServiceName: "mainsvr",
		EnablePprof: true,
	}, AdminState{
		IsAlive: alive.Load,
		IsReady: ready.Load,
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /healthz 200, got %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /readyz 200, got %d", resp.Code)
	}

	ready.Store(false)
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /readyz 503, got %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /metrics 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "go_goroutines") {
		t.Fatalf("expected go runtime metrics to be exposed")
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /debug/pprof/ 200, got %d", resp.Code)
	}
}
