package main

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"
)

// startPprofServer starts an opt-in diagnostics endpoint. It deliberately
// accepts loopback addresses only: pprof exposes CPU, heap, goroutine and
// execution data and must not be exposed to an untrusted network.
func startPprofServer(log *slog.Logger, address string) {
	addr := strings.TrimSpace(address)
	if addr == "" {
		addr = "127.0.0.1:6060"
	}
	if err := validatePprofAddress(addr); err != nil {
		log.Error("pprof disabled", "error", err)
		return
	}

	mux := http.NewServeMux()
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

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Info("pprof diagnostics listening (loopback only)", "address", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("pprof diagnostics server stopped", "error", err)
		}
	}()
}

// validatePprofAddress prevents accidentally exposing profiling endpoints
// on public or LAN interfaces.
func validatePprofAddress(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if port == "" {
		return errors.New("missing TCP port")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("address must use localhost, 127.0.0.1 or [::1]")
	}
	return nil
}
