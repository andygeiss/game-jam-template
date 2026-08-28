package app

import "net/http"

// Routes is the whole URL surface. cmd/server sets it as the app server's
// handler.
func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.handleHome)
	mux.HandleFunc("GET /game", a.handleGameRedirect)

	// /static/ sits outside the middleware chain: nothing there is user
	// supplied, and the immutable cache must not vary. The exact directory
	// URL would list the files, so it answers 404 instead.
	root := http.NewServeMux()
	root.HandleFunc("GET /static/{$}", http.NotFound)
	root.Handle("GET /static/", cacheImmutable(http.FileServerFS(a.staticFS)))
	root.Handle("/", a.middleware(mux))
	return root
}
