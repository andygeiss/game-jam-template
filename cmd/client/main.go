//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

func main() {
	// Load the assets.
	engine.LoadImages("/assets/wisp-engine.png", "/assets/player.png")
	engine.LoadSounds("/assets/title.ogg")
	// Add the engine logo and player entity.
	engine.AddEntity(engine.StateEntityVisible,
		0, 0, 0,
		160, 160,
		640/2, 360/2,
		1,
		0,
	)
	engine.AddEntity(engine.StateEntityVisible,
		1, 0, 0,
		16, 16,
		640/2, 360/2+140,
		1,
		0,
	)
	// Make an animation by using 8 frames with a duration of 125 ms each frame.
	// This is handled by the engine under the hood.
	// Thus we only need to set the state.
	engine.States[1] |= engine.StateEntityAnimated | engine.StateEntityAnimatedLoop
	// Start the game loop.
	engine.Run(func(dt float64) {
		// Apply camera shake when key 1 is pressed.
		if engine.Key1 {
			engine.CamShakeMagnitude = 2.5
			engine.CamShakeTime = 150 // big finisher: 150-200, light hits: 40-70
		}
		// Play the title sound if it's not already playing.
		if engine.Key2 {
			engine.PlaySound(0, 0.25, true)
		}
		if engine.Key3 {
			engine.HitStopRemaining = 100
		}
	})
	// Prevent the Go runtime from exiting.
	select {}
}
