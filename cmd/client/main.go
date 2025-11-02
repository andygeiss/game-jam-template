//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

const (
	cols        = 200
	rows        = 200
	worldWidth  = float64(cols) * 16
	worldHeight = float64(rows) * 16
)

const (
	indexLogo = iota
	indexPlayer
)

func main() {
	// Load the assets.
	engine.LoadImages("/assets/wisp-engine.png", "/assets/player.png", "/assets/tileset.png")
	engine.LoadSounds("/assets/title.ogg")
	// Add the engine logo and player entity.
	engine.AddEntity(engine.StateEntityVisible,
		0, 0, 0,
		160, 160,
		worldWidth/2, worldHeight/2,
		1,
		2,
	)
	engine.AddEntity(engine.StateEntityVisible,
		1, 0, 0,
		24, 24,
		worldWidth/2, worldHeight/2,
		1,
		1,
	)
	// Add the tiles.
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			engine.AddEntity(engine.StateEntityVisible,
				2, i%3, j%3,
				16, 16,
				float64(i)*16, float64(j)*16,
				1,
				0,
			)
		}
	}
	//
	startedTs := 0.0
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
		// Hide the logo after 2 seconds.
		if startedTs >= 2000 &&
			engine.States[0]&engine.StateEntityVisible == engine.StateEntityVisible {
			engine.States[0] ^= engine.StateEntityVisible
		}
		startedTs += dt
	})
	// Make an animation by using 8 frames with a duration of 125 ms each frame.
	// This is handled by the engine under the hood.
	// Thus we only need to set the state.
	engine.States[indexPlayer] |= engine.StateEntityAnimated | engine.StateEntityAnimatedLoop
	// Ensure that the camera is centered on the player.
	engine.CamTarget = indexPlayer
	// Ensure that the camera position is within the world bounds.
	engine.SetWorldSize(worldWidth, worldHeight)
	// Prevent the Go runtime from exiting.
	select {}
}
