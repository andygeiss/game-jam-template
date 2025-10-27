package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"

	"template/internal/app/adapters/ingres"
	"template/internal/app/config"

	"github.com/andygeiss/cloud-native-utils/logging"
	"github.com/andygeiss/cloud-native-utils/messaging"
	"github.com/andygeiss/cloud-native-utils/security"
	"github.com/andygeiss/cloud-native-utils/service"
)

//go:embed assets
var efs embed.FS

func main() {
	// Create a new context with a cancel function.
	ctx, cancel := service.Context()
	defer cancel()

	// Create a new configuration with the embedded file system.
	cfg := &config.Config{
		Efs:       efs,
		Logging:   logging.NewJsonLogger(),
		Messaging: messaging.NewExternalDispatcher(),
		Templates: "assets/*.html",
	}

	// Create a new service with the configuration.
	mux := ingres.Route(ctx, cfg)
	srv := security.NewServer(mux)
	defer srv.Close()

    // Register the server shutdown function on the context done function.
	service.RegisterOnContextDone(ctx, func() {
		srv.Shutdown(context.Background())
	})

	// Start the HTTP server in the main goroutine.
	cfg.Logging.Info(
		"server initialized",
		"port", os.Getenv("PORT"),
	)
	if err := srv.ListenAndServe(); err != nil {
		// Check if the server was closed intentionally.
		if err == http.ErrServerClosed {
			cfg.Logging.Error(
				"server closed",
				"reason", "server closed intentionally",
			)
			return
		}

		// Log the error and terminate the program.
		cfg.Logging.Error(
			"server failed",
			"reason", fmt.Sprintf("listening failed: %v", err),
		)
	}
}
