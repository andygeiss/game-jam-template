// Command server serves the game: one page, the static files, and the ops
// listener. It is wiring only; the handlers live in internal/app.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	wisp "github.com/andygeiss/game-jam-template"
	"github.com/andygeiss/game-jam-template/internal/app"
)

func main() {
	cfg, err := parseConfig(os.Args[1:], os.Stderr)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return
	case errors.Is(err, errUsage):
		os.Exit(2)
	case err != nil:
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(2)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

// run builds the app and serves it until SIGTERM or Ctrl-C.
func run(cfg Config) error {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if cfg.Env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)

	v := version()

	// Embedded paths keep their web/ prefix; strip it once so /static/x
	// resolves to static/x inside the sub tree.
	staticFS, err := fs.Sub(wisp.StaticFS, "web")
	if err != nil {
		return fmt.Errorf("locating static files: %w", err)
	}
	funcs := template.FuncMap{"version": func() string { return v }}
	tmpl, err := template.New("").Funcs(funcs).ParseFS(wisp.TemplatesFS, "web/templates/*.html")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	a := app.New(logger, tmpl, staticFS, v)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errorLog := slog.NewLogLogger(logger.Handler(), slog.LevelError)
	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:           a.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          errorLog,
	}
	ops := &http.Server{
		Addr:              "127.0.0.1:6060", // fixed, not a flag: never public, never proxied
		Handler:           app.OpsHandler(v),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          errorLog,
	}

	logger.Info("server starting", "addr", srv.Addr, "ops_addr", ops.Addr, "version", v, "config", cfg)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return serve(ctx, srv) })
	g.Go(func() error { return serve(ctx, ops) })
	return g.Wait()
}

// serve runs one server until ctx ends, then gives requests in flight ten
// seconds to finish. The shutdown deadline is a fresh context on purpose:
// ctx is already canceled at that moment and would cut every request off.
func serve(ctx context.Context, srv *http.Server) error {
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()

	select {
	case err := <-errc:
		return fmt.Errorf("listening on %s: %w", srv.Addr, err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down %s: %w", srv.Addr, err)
	}
	return nil
}
