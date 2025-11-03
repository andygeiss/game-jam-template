//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

const (
	cols        = 30
	rows        = 30
	worldWidth  = float64(cols) * 32
	worldHeight = float64(rows) * 32
)

const (
	indexEntityPlayer = iota
)

const (
	indexImagePlayer = iota
	indexImageTileset
)

// addPlayer adds the player entity to the game world.
func addPlayer() {
	engine.AddEntity(engine.StateEntityVisible,
		indexImagePlayer, 0, 0,
		32, 32,
		worldWidth/2, worldHeight/2,
		1,
		1,
	)
}

// addTiles adds the tiles to the game world.
func addTiles() {
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			col := i % 2
			row := j % 2
			engine.AddEntity(engine.StateEntityVisible,
				indexImageTileset, row, col,
				32, 32,
				float64(i)*32, float64(j)*32,
				1,
				0,
			)
		}
	}
}

func main() {
	// Load the assets.
	engine.LoadImages("/assets/player.png", "/assets/tileset.png")
	engine.LoadSounds("/assets/title.ogg")
	// Add the entities.
	addPlayer()
	addTiles()
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
	// Make an animation by using 8 frames with a duration of 100 ms each frame.
	// This is handled by the engine under the hood.
	// Thus we only need to set the state.
	engine.States[indexEntityPlayer] |= engine.StateEntityAnimated | engine.StateEntityAnimatedLoop
	// Ensure that the camera is centered on the player.
	engine.CamTarget = indexEntityPlayer
	// Ensure that the camera position is within the world bounds.
	engine.SetWorldSize(worldWidth, worldHeight)
	// Prevent the Go runtime from exiting.
	select {}
}
