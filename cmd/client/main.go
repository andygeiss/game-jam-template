//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

func main() {
	// Load the assets.
	engine.LoadImages("wisp-engine.png")
	engine.LoadImages("player.png")
	// Add the engine logo and player entity.
	engine.AddEntity(engine.StateEntityAlive|engine.StateEntityVisible,
		0, 0, 0,
		160, 160,
		640/2, 360/2,
		1,
		0,
	)
	engine.AddEntity(engine.StateEntityAlive|engine.StateEntityVisible,
		1, 0, 0,
		16, 16,
		640/2, 360/2+140,
		1,
		0,
	)
	// Make an animation by using 4 frames with a duration of 150 ms each frame
	// and make it loop (does not end - will be animated indefinitely).
	engine.States[1] |= engine.StateEntityAnimated | engine.StateEntityAnimatedLoop
	engine.Fcs[1] = 4
	// Run the game loop by fading in and out the background image.
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
	// Prevent the Go runtime from exiting.
	select {}
}
