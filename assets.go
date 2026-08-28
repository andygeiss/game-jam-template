// Package wisp holds the embedded web tree: the page templates and the static
// files, the compiled game included. cmd/server hands both to the app.
package wisp

import "embed"

//go:embed web/templates
var TemplatesFS embed.FS

//go:embed web/static
var StaticFS embed.FS
