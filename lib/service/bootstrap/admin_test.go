package bootstrap

import (
	"encoding/json"
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
	info := InfoSnapshot{
		Service:   "mainsvr",
		Alive:     true,
		Ready:     true,
		Phase:     phaseReady,
		StartedAt: "2026-04-05T00:00:00Z",
		Uptime:    "10s",
		Build: BuildInfo{
			Path:      "github.com/Iori372552686/GoOne/src/mainsvr",
			GoVersion: "go1.25.4",
		},
	}

	mux := newAdminMux(AdminConfig{
		Enabled:     true,
		ServiceName: "mainsvr",
		EnablePprof: true,
	}, AdminState{
		IsAlive: alive.Load,
		IsReady: ready.Load,
		Info: func() InfoSnapshot {
			info.Alive = alive.Load()
			info.Ready = ready.Load()
			return info
		},
		Components: func() []ComponentStatus {
			return []ComponentStatus{
				{Name: componentAdminServer, State: componentStateReady, Ready: true, Message: "listening on :8102"},
				{Name: componentRuntime, State: componentStateReady, Ready: true, Message: "runtime active"},
			}
		},
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

	req = httptest.NewRequest(http.MethodGet, "/info", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /info 200, got %d", resp.Code)
	}
	var infoPayload InfoSnapshot
	if err := json.Unmarshal(resp.Body.Bytes(), &infoPayload); err != nil {
		t.Fatalf("expected valid /info json, got error %v", err)
	}
	if infoPayload.Service != "mainsvr" || infoPayload.Phase != phaseReady {
		t.Fatalf("unexpected /info payload: %+v", infoPayload)
	}

	req = httptest.NewRequest(http.MethodGet, "/components", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /components 200, got %d", resp.Code)
	}
	var componentsPayload struct {
		Service    string            `json:"service"`
		Components []ComponentStatus `json:"components"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &componentsPayload); err != nil {
		t.Fatalf("expected valid /components json, got error %v", err)
	}
	if componentsPayload.Service != "mainsvr" {
		t.Fatalf("expected /components service mainsvr, got %q", componentsPayload.Service)
	}
	if len(componentsPayload.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(componentsPayload.Components))
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected /debug/pprof/ 200, got %d", resp.Code)
	}
}

func TestAdminMuxFallbackInfoAndComponents(t *testing.T) {
	mux := newAdminMux(AdminConfig{
		Enabled:     true,
		ServiceName: "infosvr",
	}, AdminState{})

	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected fallback /info 200, got %d", resp.Code)
	}
	var infoPayload InfoSnapshot
	if err := json.Unmarshal(resp.Body.Bytes(), &infoPayload); err != nil {
		t.Fatalf("expected valid /info json, got error %v", err)
	}
	if infoPayload.Service != "infosvr" {
		t.Fatalf("expected fallback service infosvr, got %q", infoPayload.Service)
	}

	req = httptest.NewRequest(http.MethodGet, "/components", nil)
	resp = httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected fallback /components 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"components":[]`) {
		t.Fatalf("expected empty component list, got %s", resp.Body.String())
	}
}
