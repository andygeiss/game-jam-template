package app

import "net/http"

// handleHome serves the game page.
func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, http.StatusOK, "index.html", nil)
}

// handleGameRedirect keeps the old /game address working.
func (a *App) handleGameRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}
