//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

func main() {
	w, h := float64(engine.CanvasWidth), float64(engine.CanvasHeight)
	engine.LoadImages("wisp-engine.png")
	engine.AddEntity(0, 0, 128, 128, 0, 0, w/2, h/2, 1)
	engine.Run(func(dt float64) {

	})
	select {}
}
