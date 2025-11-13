//go:build js && wasm

package main

import (
	"math"
	"template/internal/client/engine"
)

const (
	tilemapCols = 32
	tilemapRows = 20
	tilesetCols = 3
	tilesetRows = 3
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
	indexRowAttackRight
	indexRowAttackLeft
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
	StateAttack = uint64(1 << (iota + 16))
	StateAggressive
	StateDead
)

func main() {
	// Load the assets.
	engine.LoadImages("/assets/spritesheet.png", "/assets/tileset.png", "/assets/ui.png")
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
	tiles := []int{
		0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5,
		6, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 8,
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
		s = handleAttack(s)
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
	const n = 10
	const r = 500.0
	px, py := worldW/2, worldH/2
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
			indexImageSpritesheet, monsterCol, monsterRow, 32, 32,
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

// handleAttack handles the player's attack state and returns the next state.
func handleAttack(s uint64) (next uint64) {
	// Check if an attack is in progress.
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

	// Check if the player is pressing the attack button.
	if engine.KeyQ && s&StateAttack != StateAttack {
		s &= ^engine.StateEntityIdle
		s |= StateAttack
		engine.Fos[0] = 0
		engine.Fts[0] = 0
		engine.PlaySound(0, 1.0, false)
	}
	return s
}

// handleMovement checks if the player is within the bounds of the arena and updates their position.
func handleMovement(s uint64) (next uint64) {
	x, y := engine.Xs[0], engine.Ys[0]
	if x < 32 || x > worldW-32 {
		engine.Xs[0] = math.Max(32, math.Min(x, worldW-32))
	}
	if y < 32 || y > worldH-32 {
		engine.Ys[0] = math.Max(32, math.Min(y, worldH-32))
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
