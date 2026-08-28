package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpsHandler(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(OpsHandler("v-test"))
	t.Cleanup(srv.Close)

	t.Run("healthz reports ok and the version", func(t *testing.T) {
		t.Parallel()
		resp, err := http.Get(srv.URL + "/healthz")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var body struct{ Status, Version string }
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if body.Status != "ok" || body.Version != "v-test" {
			t.Errorf("body = %+v, want status ok and version v-test", body)
		}
	})

	t.Run("pprof index answers", func(t *testing.T) {
		t.Parallel()
		resp, err := http.Get(srv.URL + "/debug/pprof/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})
}
