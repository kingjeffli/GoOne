package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"

	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type AdminConfig struct {
	Enabled     bool
	IP          string
	Port        int
	ServiceName string
	EnablePprof bool
}

type AdminState struct {
	IsAlive    func() bool
	IsReady    func() bool
	Info       func() InfoSnapshot
	Components func() []ComponentStatus
}

type BuildInfo struct {
	Path        string `json:"path,omitempty"`
	Version     string `json:"version,omitempty"`
	GoVersion   string `json:"go_version,omitempty"`
	GOOS        string `json:"go_os,omitempty"`
	GOARCH      string `json:"go_arch,omitempty"`
	VCSRevision string `json:"vcs_revision,omitempty"`
	VCSTime     string `json:"vcs_time,omitempty"`
	VCSDirty    string `json:"vcs_dirty,omitempty"`
}

type InfoSnapshot struct {
	Service      string    `json:"service"`
	Alive        bool      `json:"alive"`
	Ready        bool      `json:"ready"`
	Phase        string    `json:"phase,omitempty"`
	StartedAt    string    `json:"started_at,omitempty"`
	PhaseSince   string    `json:"phase_since,omitempty"`
	Uptime       string    `json:"uptime,omitempty"`
	LastReloadAt string    `json:"last_reload_at,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
	Build        BuildInfo `json:"build"`
}

type ComponentStatus struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	Ready     bool   `json:"ready"`
	Message   string `json:"message,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type AdminServer struct {
	cfg  AdminConfig
	srv  *http.Server
	addr string
}

func NewAdminConfig(serviceName string, serverType int, enabled, enablePprof bool, ip string, port int) AdminConfig {
	isEnabled := enabled || enablePprof || port > 0
	if !isEnabled {
		return AdminConfig{ServiceName: serviceName}
	}
	if port <= 0 {
		port = defaultAdminPort(serverType)
	}
	return AdminConfig{
		Enabled:     true,
		IP:          ip,
		Port:        port,
		ServiceName: serviceName,
		EnablePprof: enablePprof,
	}
}

func defaultAdminPort(serverType int) int {
	if serverType <= 0 {
		return 0
	}
	return 8100 + serverType
}

func StartAdminServer(cfg AdminConfig, state AdminState) (*AdminServer, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.Port <= 0 {
		return nil, fmt.Errorf("%s admin server port is invalid", cfg.ServiceName)
	}

	addr := net.JoinHostPort(cfg.IP, strconv.Itoa(cfg.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	mux := newAdminMux(cfg, state)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	admin := &AdminServer{
		cfg:  cfg,
		srv:  srv,
		addr: addr,
	}
	go func() {
		logger.Infof("%s admin server listening on %s", cfg.ServiceName, addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("%s admin server stopped with error | %v", cfg.ServiceName, err)
		}
	}()
	return admin, nil
}

func (s *AdminServer) Shutdown(ctx context.Context) error {
	if s == nil || s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

func newAdminMux(cfg AdminConfig, state AdminState) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintf(w, "%s admin endpoints: /healthz /readyz /info /components /metrics\n", cfg.ServiceName)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if state.IsAlive != nil && !state.IsAlive() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if state.IsReady == nil || !state.IsReady() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ready\n"))
	})

	mux.HandleFunc("/info", func(w http.ResponseWriter, _ *http.Request) {
		info := InfoSnapshot{
			Service: cfg.ServiceName,
		}
		if state.Info != nil {
			info = state.Info()
			if info.Service == "" {
				info.Service = cfg.ServiceName
			}
		} else {
			if state.IsAlive != nil {
				info.Alive = state.IsAlive()
			}
			if state.IsReady != nil {
				info.Ready = state.IsReady()
			}
		}
		writeJSON(w, http.StatusOK, info)
	})

	mux.HandleFunc("/components", func(w http.ResponseWriter, _ *http.Request) {
		components := make([]ComponentStatus, 0)
		if state.Components != nil {
			components = state.Components()
		}
		writeJSON(w, http.StatusOK, struct {
			Service    string            `json:"service"`
			Components []ComponentStatus `json:"components"`
		}{
			Service:    cfg.ServiceName,
			Components: components,
		})
	})

	mux.Handle("/metrics", promhttp.Handler())

	if cfg.EnablePprof {
		registerPprof(mux)
	}
	return mux
}

func registerPprof(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Errorf("write admin json response failed | %v", err)
	}
}
