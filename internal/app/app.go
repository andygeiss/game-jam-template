// Package app is the HTTP edge: routes, middleware, and the one page handler.
package app

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

// App holds what the handlers need. Dependencies come in through New, never
// through package variables.
type App struct {
	logger   *slog.Logger
	tmpl     *template.Template
	staticFS fs.FS
	version  string
}

// New creates the app. staticFS is the tree below web/, so /static/x resolves
// to static/x in it. version names this build; it busts the asset cache.
func New(logger *slog.Logger, tmpl *template.Template, staticFS fs.FS, version string) *App {
	return &App{logger: logger, tmpl: tmpl, staticFS: staticFS, version: version}
}

// render writes a template as a full page. It renders into a buffer first, so
// a template error becomes a clean 500 instead of half a page.
func (a *App) render(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	var buf bytes.Buffer
	if err := a.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		a.serverError(w, r, fmt.Errorf("rendering %s: %w", name, err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w) // the client went away; nothing left to do about it
}

// serverError logs the full error and answers a generic 500. The error text
// never reaches the browser.
func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
