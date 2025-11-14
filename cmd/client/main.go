//go:build js && wasm

package main

import (
	"math"
	"math/rand/v2"
	"template/internal/client/engine"
)

const (
	tilemapCols = 33
	tilemapRows = 21
	tilesetCols = 3
	tilesetRows = 5
	tileW       = 32
	tileH       = 32
	worldW      = float64(tilemapCols) * tileW
	worldH      = float64(tilemapRows) * tileH
)

const (
	indexImageSpritesheet = iota
	indexImageTileset
	indexImageUi
)

const (
	indexRowIdleRight = iota
	indexRowIdleLeft
	indexRowMoveRight
	indexRowMoveLeft
	indexRowAction1Right
	indexRowAction1Left
	indexRowMonsterMove
	indexRowDeath
)

var (
	indexUiButtonQ       int
	indexUiButtonE       int
	indexUiButtonR       int
	indexUiPlayer        int
	indexUiPlayerAttack1 int
	indexUiPlayerAttack2 int
	indexUiPlayerAttack3 int
	indexUiPlayerLive1   int
	indexUiPlayerLive2   int
	indexUiPlayerLive3   int
	indexUiPlayerLive4   int
)

const (
	StateAction1 = uint64(1 << (iota + 16))
	StateAction2
	StateAction3
	StateAggressive
	StateDead
)

func main() {
	// Load the assets.
	engine.LoadImages("/assets/spritesheet.png", "/assets/tileset.png", "/assets/ui.png")
	engine.LoadSounds("/assets/attack.wav", "/assets/hit.wav", "/assets/music.ogg")

	// Load the state mapping.
	engine.RowIndexMask = StateAction1 | StateAction2 | StateAction3 |
		StateAggressive | StateDead |
		engine.StateEntityFaceLeft | engine.StateEntityFaceRight |
		engine.StateEntityIdle | engine.StateEntityMove

	engine.RowIndexForState = map[uint64]int{
		engine.StateEntityFaceRight | StateAction1:           indexRowAction1Right,
		engine.StateEntityFaceLeft | StateAction1:            indexRowAction1Left,
		engine.StateEntityFaceRight | engine.StateEntityIdle: indexRowIdleRight,
		engine.StateEntityFaceLeft | engine.StateEntityIdle:  indexRowIdleLeft,
		engine.StateEntityFaceRight | engine.StateEntityMove: indexRowMoveRight,
		engine.StateEntityFaceLeft | engine.StateEntityMove:  indexRowMoveLeft,
		StateDead: indexRowDeath,
	}

	// Set the tilemap.
	tiles := []int{
		0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 11, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		9, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 10,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		6, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 12, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 8,
	}

	// Add the entities.
	addPlayer() // entity 0
	addUi()     // entity 1
	addMobs()
	engine.AddTilemap(indexImageTileset, tiles, tilemapCols, tilemapRows, tilesetCols, tilesetRows, tileW, tileH)

	// Start the game loop.
	engine.Run(func(dt float64) {
		// Skip game updates if there is no player input.
		if !engine.HasPlayerInput {
			return
		}

		// Play the title sound if it's not already playing.
		engine.PlaySound(2, 0.25, true)

		// Move aggressive monsters towards the player position.
		moveMonsters(dt)

		// Handle player-specific states.
		s := engine.States[0]
		s = handleAction1(s)
		s = handleMovement(s)
		engine.States[0] = s
	})

	// Ensure that the camera is centered on the player.
	engine.CamTarget = 0

	// Ensure that the camera position is within the world bounds.
	engine.SetWorldSize(worldW, worldH)

	// Prevent the Go runtime from exiting.
	select {}
}

// addMobs adds monsters to the game world.
func addMobs() {
	const n = 16
	const r = 64
	for i := 0; i < n; i++ {
		// Randomize space between monsters.
		space := rand.Float64() * float64(r)

		// Select x and y based on the map boundary at the center
		// of the left, top, right, and bottom sides.
		var x, y float64
		switch i % 4 {
		case 0: // Left side
			x = -float64(i*r) + space
			y = worldH / 2
		case 1: // Top side
			x = worldW / 2
			y = -float64(i*r) + space
		case 2: // Right side
			x = worldW + float64(i*r) + space
			y = worldH / 2
		case 3: // Bottom side
			x = worldW / 2
			y = worldH + float64(i*r) + space
		}

		// Add the monster (as invisible) to the game world.
		// It will be visible when the monster moves into the arena boundary.
		engine.AddEntity(
			engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|StateAggressive,
			indexImageSpritesheet, 0, indexRowMonsterMove, 32, 32,
			x, y, 1, 1,
		)
	}
}

// addPlayer adds the player entity to the game world.
func addPlayer() {
	engine.AddEntity(engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|
		engine.StateEntityFaceRight|engine.StateEntityIdle|engine.StateEntityVisible,
		indexImageSpritesheet, 0, 0, 32, 32,
		worldW/2, worldH/2, 1, 1,
	)
}

// addUi adds the UI entity to the game world.
func addUi() {
	center := float64(engine.CanvasWidth / 2)
	baseY := float64(engine.CanvasHeight)

	// Define local inline helpers.
	ui := func(imgCol, imgRow int, w, h, x, y float64, z int) int {
		return engine.AddUI(engine.StateEntityVisible, indexImageUi, imgCol, imgRow, w, h, x, y, 1, z)
	}
	uiButton := func(bgCol, bgRow, iconCol, iconRow int, x, y float64) (index int) {
		ui(bgCol, bgRow, 32, 32, x, y, 990)
		index = ui(iconCol, iconRow, 32, 32, x, y, 999)
		return index
	}

	// Add the player UI bar + portrait.
	ui(0, 0, 96, 32, 64, 32, 990)
	indexUiPlayer = ui(0, 2, 16, 16, 32, 32, 999)

	// Add the player lives.
	indexUiPlayerLive1 = ui(5, 2, 16, 16, 48+0, 32, 999)
	indexUiPlayerLive2 = ui(5, 2, 16, 16, 48+16, 32, 999)
	indexUiPlayerLive3 = ui(5, 2, 16, 16, 48+32, 32, 999)
	indexUiPlayerLive4 = ui(5, 2, 16, 16, 48+48, 32, 999)

	// Add the player attack buttons.
	indexUiPlayerAttack1 = uiButton(3, 0, 0, 2, center-32, baseY-56)
	indexUiPlayerAttack2 = uiButton(3, 0, 1, 2, center, baseY-56)
	indexUiPlayerAttack3 = ui(3, 0, 32, 32, center+32, baseY-56, 990)
}

// handleAction1 handles the player's action1 state and returns the next state.
func handleAction1(s uint64) (next uint64) {
	// Check if an attack is in progress.
	if s&StateAction1 == StateAction1 {
		if engine.Fos[0] == 3 {
			engine.CamShakeMagnitude = 2.5
			engine.CamShakeTime = 100
			engine.Ss[0] = 3.0
		}
		if engine.Fos[0] == 7 {
			s &= ^StateAction1
			engine.Ss[0] = 1.0
		}
	}

	// Apply a hit window during attack frames 4..6.
	if s&StateAction1 == StateAction1 && engine.Fos[0] >= 4 && engine.Fos[0] <= 6 {
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

	// Check if the player is pressing the attack button.
	if engine.KeyQ && s&StateAction1 != StateAction1 {
		s &= ^engine.StateEntityIdle
		s |= StateAction1
		engine.Fos[0] = 0
		engine.Fts[0] = 0
		engine.PlaySound(0, 1.0, false)
	}
	return s
}

// handleMovement checks if the player is within the bounds of the arena and updates their position.
func handleMovement(s uint64) (next uint64) {
	// Handle player movement.
	x, y := engine.Xs[0], engine.Ys[0]
	if x < 32 || x > worldW-32 {
		engine.Xs[0] = math.Max(32, math.Min(x, worldW-32))
	}
	if y < 32 || y > worldH-32 {
		engine.Ys[0] = math.Max(32, math.Min(y, worldH-32))
	}

	// Update monster movement.
	// Make the monster visible if its within the bounds of the arena.
	for i := 1; i < len(engine.States); i++ {
		s := engine.States[i]
		if s&engine.StateEntityVisible != engine.StateEntityVisible &&
			s&StateAggressive == StateAggressive {
			x, y := engine.Xs[i], engine.Ys[i]
			s := engine.States[i]

			// Make it visible if within bounds.
			if x >= 32 && x <= worldW-32 && y >= 32 && y <= worldH-32 {
				s |= engine.StateEntityVisible
				engine.States[i] = s
			}
		}
	}

	return s
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

	// Create a hit-stop at the 6th attack frame of the player's attack animation.
	engine.Fos[0] = 5
	engine.HitStopRemaining = 70
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
