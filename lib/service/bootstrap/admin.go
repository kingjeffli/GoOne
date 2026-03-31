package bootstrap

import (
	"context"
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
	IsAlive func() bool
	IsReady func() bool
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
		_, _ = fmt.Fprintf(w, "%s admin endpoints: /healthz /readyz /metrics\n", cfg.ServiceName)
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
