package app

import (
	"fmt"
	"net/http"
	"net/http/pprof"
)

// OpsHandler serves /healthz and /debug/pprof on the localhost-only ops
// listener. There is no database to ping: healthy means the process answers.
func OpsHandler(version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":%q}`, version)
	})
	// Registered by hand: the blank net/http/pprof import registers on
	// http.DefaultServeMux, which nothing here serves.
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}
