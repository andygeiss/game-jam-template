package app

import (
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"testing"

	wisp "github.com/andygeiss/game-jam-template"
)

// newTestApp builds the app on the real embedded templates and static files,
// with a fixed version and a silent logger.
func newTestApp(t *testing.T) *App {
	t.Helper()
	staticFS, err := fs.Sub(wisp.StaticFS, "web")
	if err != nil {
		t.Fatalf("locating static files: %v", err)
	}
	funcs := template.FuncMap{"version": func() string { return "v-test" }}
	tmpl, err := template.New("").Funcs(funcs).ParseFS(wisp.TemplatesFS, "web/templates/*.html")
	if err != nil {
		t.Fatalf("parsing templates: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(logger, tmpl, staticFS, "v-test")
}
