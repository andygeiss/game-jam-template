//go:build js && wasm

package main

import (
	"template/internal/client/engine"
)

const (
	cols        = 22
	rows        = 12
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

func main() {
	// Load the assets.
	engine.LoadImages("/assets/spritesheet.png", "/assets/tileset.png")
	engine.LoadSounds("/assets/title.ogg")
	// Set the tilemap.
	tilemap := []int{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	// Add the entities.
	addPlayer()
	addTiles(tilemap)
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
		if engine.Key4 {

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
func addTiles(tiles []int) {
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			// Calculate the image column and row based on the tile index.
			imgCol := tiles[i*rows+j] / cols
			imgRow := tiles[i*rows+j] % cols
			// Add the tile entity to the game world.
			engine.AddEntity(engine.StateEntityVisible,
				indexImageTileset, imgCol, imgRow,
				32, 32,
				float64(i)*32, float64(j)*32,
				1,
				0,
			)
		}
	}
}
