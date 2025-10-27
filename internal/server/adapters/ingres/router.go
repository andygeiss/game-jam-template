package ingres

import (
	"context"
	"net/http"

	"template/internal/server/adapters/ingres/ui"
	"template/internal/server/config"

	"github.com/andygeiss/cloud-native-utils/logging"
	"github.com/andygeiss/cloud-native-utils/security"
	"github.com/andygeiss/cloud-native-utils/templating"
)

// Route creates a new mux with the liveness and readiness probe (/liveness, /readiness),
// the static assets endpoint (/) and the ui endpoints (/ui).
func Route(ctx context.Context, cfg *config.Config) *http.ServeMux {
	// Create a new mux with liveness and readyness endpoint.
	// Embed the assets into the mux.
	mux, serverSessions := security.NewServeMux(ctx, cfg.Efs)

	// Create a new templating engine and parse the templates.
	engine := templating.NewEngine(cfg.Efs)
	engine.Parse(cfg.Templates)

	// Add the UI endpoints for HTMX.

	// Add the UI endpoints for HTMX.
	mux.HandleFunc("GET /ui", logging.WithLogging(cfg.Logging,
		security.WithAuth(serverSessions, ui.ViewIndex(cfg, engine, serverSessions))))

	return mux
}
