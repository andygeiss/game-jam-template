package ui

import (
	"net/http"

	"template/internal/app/config"

	"github.com/andygeiss/cloud-native-utils/security"
	"github.com/andygeiss/cloud-native-utils/templating"
)

// View defines an HTTP handler function for rendering a template with data.
func View(engine *templating.Engine, name string, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		engine.Render(w, name, data)
	}
}

// ViewIndexData specifies the view data.
type ViewIndexData struct {
	Email     string
	Issuer    string
	Name      string
	SessionID string
	Subject   string
	Verified  bool
}

// ViewIndex defines an HTTP handler function for rendering the index template.
func ViewIndex(cfg *config.Config, engine *templating.Engine, sessions *security.ServerSessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Make a shortcut for the current context.
		ctx := r.Context()

		// Add session-specific data.
		data := ViewIndexData{
			Email:     ctx.Value(security.ContextEmail).(string),
			Issuer:    ctx.Value(security.ContextIssuer).(string),
			Name:      ctx.Value(security.ContextName).(string),
			SessionID: ctx.Value(security.ContextSessionID).(string),
			Subject:   ctx.Value(security.ContextSubject).(string),
			Verified:  ctx.Value(security.ContextVerified).(bool),
		}

		// Render the template using the provided engine and data.
		View(engine, "index", data)(w, r)
	}
}

