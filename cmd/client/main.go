//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

func main() {
	engine.LoadImages("wisp-engine.png")
	engine.LoadImages("player.png")
	engine.AddEntity(engine.StateEntityAlive|engine.StateEntityVisible,
		0, 0, 0,
		320, 320,
		640/2, 360/2,
		1,
		0,
	)
	engine.AddEntity(engine.StateEntityAlive|engine.StateEntityVisible,
		1, 0, 0,
		32, 32,
		640/2, 360/2+140,
		1,
		0,
	)
	fadeOut := false
	engine.Run(func(dt float64) {
		if fadeOut {
			engine.As[0] -= dt / 1000
			if engine.As[0] < 0 {
				engine.As[0] = 0
				fadeOut = false
			}
		} else {
			engine.As[0] += dt / 1000
			if engine.As[0] > 1 {
				engine.As[0] = 1
				fadeOut = true
			}
		}
	})
	select {}
}
