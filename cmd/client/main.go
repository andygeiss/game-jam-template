//go:build js && wasm

package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"template/internal/client/engine"
)

const (
	projectileSpeed = 3.0
	tilemapCols     = 33
	tilemapRows     = 21
	tilesetCols     = 3
	tilesetRows     = 5
	tileW           = 32
	tileH           = 32
	worldW          = float64(tilemapCols) * tileW
	worldH          = float64(tilemapRows) * tileH
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
	indexRowAction2
)

var (
	indexUiButtonQ       int
	indexUiButtonE       int
	indexUiButtonR       int
	indexUiPlayer        int
	indexUiPlayerAction1 int
	indexUiPlayerAction2 int
	indexUiPlayerAction3 int
	indexUiPlayerLive1   int
	indexUiPlayerLive2   int
	indexUiPlayerLive3   int
	indexUiPlayerLive4   int
)

var (
	action1Cooldown           = 1000.0
	action2Cooldown           = 3000.0
	action1CooldownDt float64 = 0
	action2CooldownDt float64 = 0
	gameOver          bool
	playerLives       int
	monstersKilled    int
)

const (
	StateAction1 = uint64(1 << (iota + 16))
	StateAction2
	StateAction3
	StateAggressive
	StateDead
	StateInvincible
	StateProjectile
)

func main() {
	// Initialize game state.
	playerLives = 4
	monstersKilled = 0

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

	// Wire up the UI.
	engine.SetRenderUi(renderUI)

	// Start the game loop.
	engine.Run(func(dt float64) {
		// Skip game updates if there is no player input.
		if !engine.HasPlayerInput {
			return
		}

		// Stop the game loop if the game is over by clearing the player states.
		if gameOver {
			engine.States[0] = 0
			return
		}

		// Play the title sound if it's not already playing.
		engine.PlaySound(2, 0.25, true)

		reduceCooldowns(dt)
		moveMonsters(dt)
		moveProjectiles()
		checkCollision()
		updateButtons()

		// Handle player-specific states.
		s := engine.States[0]
		s = handleAction1(s)
		s = handleAction2(s)
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
	const n = 100
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
	ui(0, 0, 96, 32, center, 32, 990)
	indexUiPlayer = ui(0, 2, 16, 16, center-32, 32, 999)

	// Add the player lives.
	indexUiPlayerLive1 = ui(5, 2, 16, 16, center-16, 32, 999)
	indexUiPlayerLive2 = ui(5, 2, 16, 16, center+0, 32, 999)
	indexUiPlayerLive3 = ui(5, 2, 16, 16, center+16, 32, 999)
	indexUiPlayerLive4 = ui(5, 2, 16, 16, center+32, 32, 999)

	// Add the player attack buttons.
	indexUiPlayerAction1 = uiButton(3, 0, 0, 2, center-32, baseY-32)
	indexUiPlayerAction2 = uiButton(3, 0, 1, 2, center, baseY-32)
	indexUiPlayerAction3 = ui(3, 0, 32, 32, center+32, baseY-32, 990)
}

// checkCollision checks for collisions between the player and monsters.
func checkCollision() {
	for i := 1; i < len(engine.States); i++ {
		s := engine.States[i]

		// Skip non-aggressive, invisible or dead monsters.
		if s&StateAggressive == 0 || s&StateDead == StateDead {
			continue
		}

		// Check collision between player and monster.
		// Player is invincible during attack1.
		if engine.HasCollision(0, i) && engine.States[0]&StateInvincible == 0 {
			playerLives--
			engine.CamShakeMagnitude = 4.0
			engine.CamShakeTime = 150
			engine.HitStopRemaining = 150
			killMonster(i)
		}
	}
}

// fireAttack2Projectiles spawns 4 projectiles from the player position
// flying north, east, south and west.
func fireAttack2Projectiles() {
	px, py := engine.Xs[0], engine.Ys[0]

	spawn := func(extraState uint64) {
		fullState := engine.StateEntityAnimated |
			engine.StateEntityAnimatedLoop |
			engine.StateEntityVisible |
			extraState | StateProjectile

		// Reuse existing projectile if available.
		idx := spawnOrReuseProjectile(fullState, indexImageSpritesheet,
			0, indexRowAction2, px, py)

		engine.Ss[idx] = projectileSpeed
	}

	// North, East, South, West
	spawn(engine.StateEntityMoveUp)
	spawn(engine.StateEntityMoveRight)
	spawn(engine.StateEntityMoveDown)
	spawn(engine.StateEntityMoveLeft)
}

// handleAction1 handles the player's action1 state and returns the next state.
func handleAction1(s uint64) (next uint64) {
	// Check if an attack is in progress.
	if s&StateAction1 == StateAction1 {
		if engine.Fos[0] == 3 {
			engine.CamShakeMagnitude = 2.5
			engine.CamShakeTime = 100
			engine.Ss[0] = 2.0
		}
		if engine.Fos[0] == 7 {
			s &= ^(StateAction1 | StateInvincible)
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

	// Check for cooldown.
	if action1CooldownDt > 0 {
		return s
	}

	// Check if the player is pressing the attack button.
	if engine.KeyQ && s&StateAction1 != StateAction1 {
		s &= ^engine.StateEntityIdle
		s |= StateAction1 | StateInvincible
		engine.Fos[0] = 0
		engine.Fts[0] = 0
		engine.PlaySound(0, 1.0, false)
		action1CooldownDt = action1Cooldown
	}
	return s
}

// handleAction2 handles the second action of the player.
func handleAction2(s uint64) (next uint64) {
	// Check for cooldown and trigger only once per key press.
	if action2CooldownDt > 0 {
		return s &^ StateAction2
	}

	// Trigger only once per key press.
	if engine.KeyE && s&StateAction2 == 0 {
		s |= StateAction2
		fireAttack2Projectiles()
		engine.PlaySound(0, 1.0, false)

		// Start cooldown now!
		action2CooldownDt = action2Cooldown
	}

	// Clear action state as soon as possible.
	if !engine.KeyE {
		s &^= StateAction2
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

	// Increment the monsters killed counter.
	monstersKilled++
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

// moveProjectiles moves projectiles and handles collisions and despawn.
func moveProjectiles() {
	for i := 1; i < len(engine.States); i++ {
		s := engine.States[i]

		// Only care about visible projectiles.
		if s&StateProjectile == 0 || s&engine.StateEntityVisible == 0 {
			continue
		}

		x, y := engine.Xs[i], engine.Ys[i]

		// Despawn when leaving the world bounds.
		if x < 0 || x > worldW || y < 0 || y > worldH {
			s &^= (engine.StateEntityVisible |
				engine.StateEntityMove |
				engine.StateEntityMoveUp |
				engine.StateEntityMoveDown |
				engine.StateEntityMoveLeft |
				engine.StateEntityMoveRight)
			engine.States[i] = s
			continue
		}

		// Check collision with aggressive monsters.
		for j := 1; j < len(engine.States); j++ {
			// Skip self.
			if i == j {
				continue
			}

			// Skip non-aggressive or non-visible monsters.
			ms := engine.States[j]
			if ms&StateAggressive == 0 ||
				ms&engine.StateEntityVisible == 0 {
				continue
			}

			// Kill the monster if it's hit by the projectile.
			if engine.HasCollision(i, j) {
				engine.PlaySound(1, 1.0, false)
				killMonster(j)
				s &^= engine.StateEntityVisible
				engine.States[i] = s
				break
			}
		}
	}
}

// reduceCooldowns reduces the cooldowns of the player actions.
func reduceCooldowns(dt float64) {
	if action1CooldownDt > 0 {
		action1CooldownDt -= dt
		if action1CooldownDt < 0 {
			action1CooldownDt = 0
		}
	}

	if action2CooldownDt > 0 {
		action2CooldownDt -= dt
		if action2CooldownDt < 0 {
			action2CooldownDt = 0
		}
	}

}

// renderUI renders the UI elements.
func renderUI() {
	alive := 0
	for i := 1; i < len(engine.States); i++ {
		s := engine.States[i]
		if s&StateAggressive == StateAggressive &&
			s&StateDead != StateDead &&
			s&engine.StateEntityVisible == engine.StateEntityVisible {
			alive++
		}
	}

	aliveText := fmt.Sprintf("Monsters alive: %d", alive)
	engine.RenderText(8, 16, aliveText, "white", "16px Arial", "left")

	killedText := fmt.Sprintf("Monsters killed: %d", monstersKilled)
	engine.RenderText(8, 32, killedText, "yellow", "16px Arial", "left")

	if gameOver {
		engine.RenderText(engine.CanvasWidth/2, engine.CanvasHeight/2, "Game Over", "red", "24px Arial", "center")
	}
}

// spawnOrReuseProjectile spawns or reuses a projectile.
func spawnOrReuseProjectile(state uint64, imgIdx, col, row int, x, y float64) int {
	// Try to reuse invisible projectile.
	for i := 1; i < len(engine.States); i++ {
		if engine.States[i]&StateProjectile != 0 &&
			engine.States[i]&engine.StateEntityVisible == 0 {

			// Reuse this one.
			engine.States[i] = state
			engine.Xs[i] = x
			engine.Ys[i] = y
			engine.Fos[i] = 0
			engine.Fts[i] = 0
			return i
		}
	}

	// Create new projectile.
	return engine.AddEntity(
		state, imgIdx, col, row, 32, 32, x, y, 1.0, 2,
	)
}

// updateButtons updates the UI buttons.
func updateButtons() {
	// Handle action1 button.
	if action1CooldownDt > 0 {
		engine.As[indexUiPlayerAction1] = 0.25
	} else {
		engine.As[indexUiPlayerAction1] = 1.0
	}

	// Handle action2 button.
	if action2CooldownDt > 0 {
		engine.As[indexUiPlayerAction2] = 0.25
	} else {
		engine.As[indexUiPlayerAction2] = 1.0
	}

	// Update the player's lives.
	switch playerLives {
	case 1:
		engine.As[indexUiPlayerLive1] = 1.0
		engine.As[indexUiPlayerLive2] = 0.0
		engine.As[indexUiPlayerLive3] = 0.0
		engine.As[indexUiPlayerLive4] = 0.0
	case 2:
		engine.As[indexUiPlayerLive1] = 1.0
		engine.As[indexUiPlayerLive2] = 1.0
		engine.As[indexUiPlayerLive3] = 0.0
		engine.As[indexUiPlayerLive4] = 0.0
	case 3:
		engine.As[indexUiPlayerLive1] = 1.0
		engine.As[indexUiPlayerLive2] = 1.0
		engine.As[indexUiPlayerLive3] = 1.0
		engine.As[indexUiPlayerLive4] = 0.0
	case 4:
		engine.As[indexUiPlayerLive1] = 1.0
		engine.As[indexUiPlayerLive2] = 1.0
		engine.As[indexUiPlayerLive3] = 1.0
		engine.As[indexUiPlayerLive4] = 1.0
	default:
		engine.As[indexUiPlayerLive1] = 0.0
		engine.As[indexUiPlayerLive2] = 0.0
		engine.As[indexUiPlayerLive3] = 0.0
		engine.As[indexUiPlayerLive4] = 0.0
		gameOver = true
	}
}
