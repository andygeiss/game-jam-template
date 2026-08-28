package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(newTestApp(t).Routes())
	t.Cleanup(srv.Close)

	// Never follow redirects: the 301 itself is what the test checks.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	tests := []struct {
		name       string
		method     string
		path       string
		origin     string
		wantStatus int
		wantHeader map[string]string
		wantBody   []string
	}{
		{
			name: "home renders the game page", method: "GET", path: "/",
			wantStatus: http.StatusOK,
			wantHeader: map[string]string{
				"Content-Type":            "text/html; charset=utf-8",
				"Content-Security-Policy": csp,
			},
			wantBody: []string{`<html lang="en" data-version="v-test">`, `<main>`, `/static/js/wasm_app.js?v=v-test`},
		},
		{
			name: "the old game address redirects home", method: "GET", path: "/game",
			wantStatus: http.StatusMovedPermanently,
			wantHeader: map[string]string{"Location": "/"},
		},
		{
			name: "static assets are immutable", method: "GET", path: "/static/css/app.css",
			wantStatus: http.StatusOK,
			wantHeader: map[string]string{"Cache-Control": "public, max-age=31536000, immutable"},
		},
		{
			name: "the wasm module has its content type", method: "GET", path: "/static/game.wasm",
			wantStatus: http.StatusOK,
			wantHeader: map[string]string{"Content-Type": "application/wasm"},
		},
		{
			name: "the static directory is not listed", method: "GET", path: "/static/",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "an unknown path is 404 with the security headers", method: "GET", path: "/nope",
			wantStatus: http.StatusNotFound,
			wantHeader: map[string]string{"X-Content-Type-Options": "nosniff"},
		},
		{
			name: "a cross-origin POST is refused", method: "POST", path: "/", origin: "https://evil.example",
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			for name, want := range tt.wantHeader {
				if got := resp.Header.Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			for _, want := range tt.wantBody {
				if !strings.Contains(string(body), want) {
					t.Errorf("body lacks %q", want)
				}
			}
		})
	}
}
