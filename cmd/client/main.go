//go:build js && wasm

package main

import (
	"math"
	"template/internal/client/engine"
)

const (
	cols        = 22
	rows        = 12
	worldWidth  = float64(cols) * 32
	worldHeight = float64(rows) * 32
)

const (
	indexImageSpritesheet = iota
	indexImageTileset
)

const (
	indexRowIdleRight = iota
	indexRowIdleLeft
	indexRowMoveRight
	indexRowMoveLeft
	indexRowAttackRight
	indexRowAttackLeft
	indexRowMonsterMove
	indexRowDeath
)

const (
	StateAttack = uint64(1 << (iota + 16))
	StateAggressive
	StateDead
)

func main() {
	// Load the assets.
	engine.LoadImages("/assets/spritesheet.png", "/assets/tileset.png")
	engine.LoadSounds("/assets/attack.wav", "/assets/hit.wav", "/assets/music.ogg")
	// Load the state mapping.
	engine.RowIndexMask = StateAttack | StateDead |
		engine.StateEntityFaceLeft | engine.StateEntityFaceRight |
		engine.StateEntityIdle | engine.StateEntityMove
	engine.RowIndexForState = map[uint64]int{
		engine.StateEntityFaceRight | StateAttack:            indexRowAttackRight,
		engine.StateEntityFaceLeft | StateAttack:             indexRowAttackLeft,
		engine.StateEntityFaceRight | engine.StateEntityIdle: indexRowIdleRight,
		engine.StateEntityFaceLeft | engine.StateEntityIdle:  indexRowIdleLeft,
		engine.StateEntityFaceRight | engine.StateEntityMove: indexRowMoveRight,
		engine.StateEntityFaceLeft | engine.StateEntityMove:  indexRowMoveLeft,
		StateDead: indexRowDeath,
	}
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
	addMobs()
	addTiles(tilemap)
	// Start the game loop.
	engine.Run(func(dt float64) {
		// Play the title sound if it's not already playing.
		engine.PlaySound(2, 0.25, true)
		// Move aggressive monsters towards the player position.
		moveMonsters(dt)
		// Handle player-specific states.
		s := engine.States[0]
		if s&StateAttack == StateAttack {
			if engine.Fos[0] == 3 {
				engine.CamShakeMagnitude = 2.5
				engine.CamShakeTime = 100
				engine.Ss[0] = 3.0
			}
			if engine.Fos[0] == 7 {
				s &= ^StateAttack
				engine.Ss[0] = 1.0
			}
		}
		if engine.KeyE && s&StateAttack != StateAttack {
			s &= ^engine.StateEntityIdle
			s |= StateAttack
			engine.Fos[0] = 0
			engine.Fts[0] = 0
			engine.PlaySound(0, 1.0, false)
		}
		// Apply a hit window during attack frames 4..6.
		if s&StateAttack == StateAttack && engine.Fos[0] >= 4 && engine.Fos[0] <= 6 {
			for i := 1; i < len(engine.States); i++ {
				if engine.States[i]&StateAggressive == StateAggressive &&
					engine.States[i]&engine.StateEntityVisible == engine.StateEntityVisible {
					if engine.HasCollision(0, i) {
						engine.PlaySound(1, 1.0, false)
						killMonster(i)
					}
				}
			}
		}
		engine.States[0] = s
	})
	// Ensure that the camera is centered on the player.
	engine.CamTarget = 0
	// Ensure that the camera position is within the world bounds.
	engine.SetWorldSize(worldWidth, worldHeight)
	// Prevent the Go runtime from exiting.
	select {}
}

// addMobs adds monsters to the game world.
func addMobs() {
	const n = 10
	const r = 500.0 // at least 300 px from the player
	px, py := worldWidth/2, worldHeight/2
	for i := 0; i < n; i++ {
		ang := 2 * math.Pi * float64(i) / float64(n)
		x := px + math.Cos(ang)*r
		y := py + math.Sin(ang)*r
		// If your monster frames are not at col 0 / row 6, adjust these two:
		const monsterCol = 0
		const monsterRow = 6
		engine.AddEntity(
			engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|
				StateAggressive|engine.StateEntityVisible,
			indexImageSpritesheet, monsterCol, monsterRow,
			32, 32,
			x, y,
			1,
			1,
		)
	}
}

// addPlayer adds the player entity to the game world.
func addPlayer() {
	engine.AddEntity(engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|
		engine.StateEntityFaceRight|engine.StateEntityIdle|engine.StateEntityVisible,
		indexImageSpritesheet, 0, 0,
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

// killMonster stops rendering and updating this monster.
func killMonster(i int) {
	s := engine.States[i]
	// Remove behavior & movement, faces and loops.
	s &^= (StateAggressive |
		engine.StateEntityMove | engine.StateEntityIdle |
		engine.StateEntityMoveDown | engine.StateEntityMoveLeft |
		engine.StateEntityMoveRight | engine.StateEntityMoveUp |
		engine.StateEntityAnimatedLoop |
		engine.StateEntityFaceLeft | engine.StateEntityFaceRight)
	// Mark as dead and start one-shot animation (marked as auto-hide).
	s |= StateDead | engine.StateEntityAnimated | engine.StateEntityAutoHide | engine.StateEntityVisible
	engine.States[i] = s
	// Create a hit-stop at the 6th attack frame.
	engine.Fos[0] = 5
	engine.HitStopRemaining = 200
}

// moveMonsters moves the monsters towards the player position.
func moveMonsters(dt float64) {
	px, py := engine.Xs[0], engine.Ys[0]
	const pxPerMs = 0.05
	for i := 1; i < len(engine.States); i++ {
		if engine.States[i]&StateAggressive == StateAggressive {
			dx := px - engine.Xs[i]
			dy := py - engine.Ys[i]
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 0 {
				step := pxPerMs * dt
				engine.Xs[i] += (dx / dist) * step
				engine.Ys[i] += (dy / dist) * step
			}
		}
	}
}
