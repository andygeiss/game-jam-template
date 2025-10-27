//go:build js && wasm

package engine

import "syscall/js"

const (
	canvasWidth  = 640
	canvasHeight = 360
)

var (
	canvas js.Value
)

func init() {
	canvas = js.Global().Get("document").Call("createElement", "canvas")
	canvas.Set("width", canvasWidth)
	canvas.Set("height", canvasHeight)
	js.Global().Get("document").Get("body").Call("appendChild", canvas)
}

func Version() string {
	return "0.1.0"
}
