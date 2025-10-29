//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

func main() {
	engine.LoadImages("wisp-engine.png")
	engine.AddEntity(engine.StateEntityAlive|engine.StateEntityVisible,
		0, 0, 0,
		320, 320,
		engine.CanvasWidth/2, engine.CanvasHeight/2, 1,
	)
	engine.Run(func(dt float64) {

	})
	select {}
}
