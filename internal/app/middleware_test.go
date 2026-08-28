package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecureHeaders(t *testing.T) {
	t.Parallel()
	want := map[string]string{
		"Content-Security-Policy":   csp,
		"Strict-Transport-Security": "max-age=31536000",
		"X-Content-Type-Options":    "nosniff",
		"Referrer-Policy":           "same-origin",
	}

	rec := httptest.NewRecorder()
	h := (&App{}).secureHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	for name, value := range want {
		if got := rec.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}

	// Not a tautology, unlike the loop above: these fail if somebody
	// "tidies" the policy and takes the game down with it.
	if !strings.Contains(csp, "img-src 'self' data:") {
		t.Error("CSP lost img-src 'self' data:")
	}
	if !strings.Contains(csp, "script-src 'self' 'wasm-unsafe-eval'") {
		t.Error("CSP lost 'wasm-unsafe-eval' — browsers now refuse to compile the game")
	}
	for _, banned := range []string{"'unsafe-inline'", "'unsafe-eval'"} {
		if strings.Contains(csp, banned) {
			t.Errorf("CSP contains %s", banned)
		}
	}
}

func TestRecoverPanic(t *testing.T) {
	t.Parallel()
	a := newTestApp(t)
	h := a.recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("the panic text reached the response")
	}
}
