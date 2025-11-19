//go:build js && wasm

package main

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/andygeiss/game-jam-template/internal/client/engine"
)

const (
	indexImageSpritesheet = iota
	indexImageTileset
	indexImageUi
	indexImageBoss
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
	indexRowAction3Right
	indexRowAction3Left
)

const (
	stateAction1 = uint64(1 << (iota + 16))
	stateAction2
	stateAction3
	stateAggressive
	stateDead
	stateInvincible
	stateProjectile
)

const (
	monstersMax     = 10
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

var (
	action1Cooldown              = 1000.0
	action2Cooldown              = 3000.0
	action3Cooldown              = 5000.0
	action1CooldownDt    float64 = 0
	action2CooldownDt    float64 = 0
	action3CooldownDt    float64 = 0
	bossSpawned          bool
	gameOver             bool
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
	playerLives          int
	monstersKilled       int
)

func main() {
	// Add the entities.
	initializeGame()

	// Wire up the UI.
	engine.SetRenderUi(renderUI)

	// Start the game loop.
	engine.Run(func(dt float64) {
		// Skip game updates if there is no player input.
		if !engine.HasPlayerInput {
			return
		}

		// Restart the game if the player presses 'N'.
		if engine.KeyN {
			initializeGame()
			engine.KeyN = false
			gameOver = false
		}

		// Stop the game loop if the game is over by clearing the player states.
		if gameOver {
			engine.EntityState[0] = 0
			return
		}

		// Add a boss (only once) if all monsters are killed.
		if monstersKilled >= monstersMax && !bossSpawned {
			addBoss()
			engine.CamShakeMagnitude = 10.0
			engine.CamShakeTime = 1000.0
			bossSpawned = true
		}

		// Play the title sound if it's not already playing.
		engine.PlaySound(2, 0.25, true)

		reduceCooldowns(dt)
		moveMonsters(dt)
		moveProjectiles()
		checkCollision()
		updateButtons()

		// Handle player-specific states.
		s := engine.EntityState[0]
		s = handleAction1(s)
		s = handleAction2(s)
		s = handleAction3(s)
		s = handleMovement(s)
		engine.EntityState[0] = s
	})

	// Ensure that the camera is centered on the player.
	engine.CamTarget = 0

	// Ensure that the camera position is within the world bounds.
	engine.SetWorldSize(worldW, worldH)

	// Prevent the Go runtime from exiting.
	select {}
}

// addBoss adds a boss at the center of the world.
func addBoss() {
	engine.AddEntity(
		engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|stateAggressive,
		indexImageBoss, 0, 0, 96, 96,
		worldW/2, worldH/2, 1, 1,
	)
}

// addMonsters adds monsters to the game world.
func addMonsters() {
	const n = monstersMax
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
			engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|stateAggressive,
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
	indexUiPlayerAction3 = uiButton(3, 0, 2, 2, center+32, baseY-32)
}

// checkCollision checks for collisions between the player and monsters.
func checkCollision() {
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]

		// Skip non-aggressive, invisible or dead monsters.
		if s&stateAggressive == 0 || s&stateDead == stateDead {
			continue
		}

		// Check collision between player and monster.
		// Player is invincible during attack1.
		if engine.HasCollision(0, i) && engine.EntityState[0]&stateInvincible == 0 {
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
	px, py := engine.EntityX[0], engine.EntityY[0]

	spawn := func(extraState uint64) {
		fullState := engine.StateEntityAnimated |
			engine.StateEntityAnimatedLoop |
			engine.StateEntityVisible |
			extraState | stateProjectile

		// Reuse existing projectile if available.
		idx := spawnOrReuseProjectile(fullState, indexImageSpritesheet,
			0, indexRowAction2, px, py)

		engine.EntitySpeedFactor[idx] = projectileSpeed
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
	if s&stateAction1 == stateAction1 {
		if engine.EntityFrameOffset[0] == 3 {
			engine.CamShakeMagnitude = 2.5
			engine.CamShakeTime = 100
			engine.EntitySpeedFactor[0] = 2.0
		}
		if engine.EntityFrameOffset[0] == 7 {
			s &= ^(stateAction1 | stateInvincible)
			engine.EntitySpeedFactor[0] = 1.0
		}
	}

	// Apply a hit window during attack frames 4..6.
	if s&stateAction1 == stateAction1 && engine.EntityFrameOffset[0] >= 4 && engine.EntityFrameOffset[0] <= 6 {
		for i := 1; i < len(engine.EntityState); i++ {
			if engine.EntityState[i]&stateAggressive == stateAggressive &&
				engine.EntityState[i]&engine.StateEntityVisible == engine.StateEntityVisible {
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
	if engine.KeyQ && s&stateAction1 != stateAction1 {
		s &= ^engine.StateEntityIdle
		s |= stateAction1 | stateInvincible
		engine.EntityFrameOffset[0] = 0
		engine.EntityFrameTime[0] = 0
		engine.PlaySound(0, 1.0, false)
		action1CooldownDt = action1Cooldown
	}
	return s
}

// handleAction2 handles the second action of the player.
func handleAction2(s uint64) (next uint64) {
	// Check for cooldown and trigger only once per key press.
	if action2CooldownDt > 0 {
		return s &^ stateAction2
	}

	// Trigger only once per key press.
	if engine.KeyE && s&stateAction2 == 0 {
		s |= stateAction2
		fireAttack2Projectiles()
		engine.PlaySound(0, 1.0, false)

		// Start cooldown now!
		action2CooldownDt = action2Cooldown
	}

	// Clear action state as soon as possible.
	if !engine.KeyE {
		s &^= stateAction2
	}
	return s
}

// handleAction3 handles the third action of the player.
func handleAction3(s uint64) (next uint64) {
	// If action3 is currently playing, let the animation run.
	if s&stateAction3 == stateAction3 {
		// Restore the speed multiplier to 1.0 after the animation ends.
		if engine.EntityFrameOffset[0] == engine.AnimationFrameCount-1 {
			s &^= (stateAction3 | stateInvincible)
			engine.EntitySpeedFactor[0] = 1.0
		}
		return s
	}

	// If on cooldown, don't start a new action3.
	if action3CooldownDt > 0 {
		return s
	}

	// Trigger only once per key press.
	if engine.KeyR {
		s |= (stateAction3 | stateInvincible)
		engine.EntityFrameOffset[0] = 0
		engine.EntityFrameTime[0] = 0
		engine.EntitySpeedFactor[0] = 4.0
		engine.PlaySound(0, 1.0, false)
		action3CooldownDt = action3Cooldown
	}

	return s
}

// handleMovement checks if the player is within the bounds of the arena and updates their position.
func handleMovement(s uint64) (next uint64) {
	// Handle player movement.
	x, y := engine.EntityX[0], engine.EntityY[0]
	if x < 32 || x > worldW-32 {
		engine.EntityX[0] = math.Max(32, math.Min(x, worldW-32))
	}
	if y < 32 || y > worldH-32 {
		engine.EntityY[0] = math.Max(32, math.Min(y, worldH-32))
	}

	// Update monster movement.
	// Make the monster visible if its within the bounds of the arena.
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]
		if s&engine.StateEntityVisible != engine.StateEntityVisible &&
			s&stateAggressive == stateAggressive {
			x, y := engine.EntityX[i], engine.EntityY[i]
			s := engine.EntityState[i]

			// Make it visible if within bounds.
			if x >= 32 && x <= worldW-32 && y >= 32 && y <= worldH-32 {
				s |= engine.StateEntityVisible
				engine.EntityState[i] = s
			}
		}
	}

	return s
}

// Initialize the game state.
func initializeGame() {
	// Initialize game state.
	playerLives = 4
	monstersKilled = 0

	// Load the assets.
	engine.LoadImages("/assets/spritesheet.png", "/assets/tileset.png", "/assets/ui.png", "/assets/boss.png")
	engine.LoadSounds("/assets/attack.wav", "/assets/hit.wav", "/assets/music.ogg")

	// Load the state mapping.
	engine.RowIndexMask = stateAction1 | stateAction2 | stateAction3 |
		stateAggressive | stateDead |
		engine.StateEntityFaceLeft | engine.StateEntityFaceRight |
		engine.StateEntityIdle | engine.StateEntityMove

	engine.RowIndexForState = map[uint64]int{
		engine.StateEntityFaceRight | stateAction1:           indexRowAction1Right,
		engine.StateEntityFaceLeft | stateAction1:            indexRowAction1Left,
		engine.StateEntityFaceRight | stateAction3:           indexRowAction3Right,
		engine.StateEntityFaceLeft | stateAction3:            indexRowAction3Left,
		engine.StateEntityFaceRight | engine.StateEntityIdle: indexRowIdleRight,
		engine.StateEntityFaceLeft | engine.StateEntityIdle:  indexRowIdleLeft,
		engine.StateEntityFaceRight | engine.StateEntityMove: indexRowMoveRight,
		engine.StateEntityFaceLeft | engine.StateEntityMove:  indexRowMoveLeft,
		stateDead: indexRowDeath,
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

	// Initialize the game state.
	engine.InitializeEntities()

	addPlayer()
	addUi()
	addMonsters()

	engine.AddTilemap(indexImageTileset, tiles, tilemapCols, tilemapRows, tilesetCols, tilesetRows, tileW, tileH)
}

// killMonster stops rendering and updating this monster.
func killMonster(i int) {
	s := engine.EntityState[i]

	// Remove behavior & movement, faces and loops.
	s &^= (stateAggressive |
		engine.StateEntityMove | engine.StateEntityIdle |
		engine.StateEntityMoveDown | engine.StateEntityMoveLeft |
		engine.StateEntityMoveRight | engine.StateEntityMoveUp |
		engine.StateEntityAnimatedLoop |
		engine.StateEntityFaceLeft | engine.StateEntityFaceRight)

	// Mark as dead and start one-shot animation (marked as auto-hide).
	s |= stateDead | engine.StateEntityAnimated | engine.StateEntityAutoHide | engine.StateEntityVisible
	engine.EntityState[i] = s

	// Create a hit-stop at the 6th attack frame of the player's attack animation.
	engine.EntityFrameOffset[0] = 5
	engine.HitStopRemaining = 70

	// Increment the monsters killed counter.
	monstersKilled++
}

// moveMonsters moves the monsters towards the player position.
func moveMonsters(dt float64) {
	px, py := engine.EntityX[0], engine.EntityY[0]
	const pxPerMs = 0.05
	for i := 1; i < len(engine.EntityState); i++ {
		if engine.EntityState[i]&stateAggressive == stateAggressive {
			dx := px - engine.EntityX[i]
			dy := py - engine.EntityY[i]
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 0 {
				step := pxPerMs * dt
				engine.EntityX[i] += (dx / dist) * step
				engine.EntityY[i] += (dy / dist) * step
			}
		}
	}
}

// moveProjectiles moves projectiles and handles collisions and despawn.
func moveProjectiles() {
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]

		// Only care about visible projectiles.
		if s&stateProjectile == 0 || s&engine.StateEntityVisible == 0 {
			continue
		}

		x, y := engine.EntityX[i], engine.EntityY[i]

		// Despawn when leaving the world bounds.
		if x < 0 || x > worldW || y < 0 || y > worldH {
			s &^= (engine.StateEntityVisible |
				engine.StateEntityMove |
				engine.StateEntityMoveUp |
				engine.StateEntityMoveDown |
				engine.StateEntityMoveLeft |
				engine.StateEntityMoveRight)
			engine.EntityState[i] = s
			continue
		}

		// Check collision with aggressive monsters.
		for j := 1; j < len(engine.EntityState); j++ {
			// Skip self.
			if i == j {
				continue
			}

			// Skip non-aggressive or non-visible monsters.
			ms := engine.EntityState[j]
			if ms&stateAggressive == 0 ||
				ms&engine.StateEntityVisible == 0 {
				continue
			}

			// Kill the monster if it's hit by the projectile.
			if engine.HasCollision(i, j) {
				engine.PlaySound(1, 1.0, false)
				killMonster(j)
				s &^= engine.StateEntityVisible
				engine.EntityState[i] = s
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

	if action3CooldownDt > 0 {
		action3CooldownDt -= dt
		if action3CooldownDt < 0 {
			action3CooldownDt = 0
		}
	}
}

// renderUI renders the UI elements.
func renderUI() {

	// Skip rendering if the game is not running.
	if !engine.HasPlayerInput {
		engine.RenderText(engine.CanvasWidth/2, engine.CanvasHeight/2, "Click to start the game", "white", "24px Arial", "center")
		return
	}

	alive := 0
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]
		if s&stateAggressive == stateAggressive &&
			s&stateDead != stateDead &&
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
		engine.RenderText(engine.CanvasWidth/2, engine.CanvasHeight/2+24, "Press N for new game", "red", "16px Arial", "center")
	}
}

// spawnOrReuseProjectile spawns or reuses a projectile.
func spawnOrReuseProjectile(state uint64, imgIdx, col, row int, x, y float64) int {
	// Try to reuse invisible projectile.
	for i := 1; i < len(engine.EntityState); i++ {
		if engine.EntityState[i]&stateProjectile != 0 &&
			engine.EntityState[i]&engine.StateEntityVisible == 0 {

			// Reuse this one.
			engine.EntityState[i] = state
			engine.EntityX[i] = x
			engine.EntityY[i] = y
			engine.EntityFrameOffset[i] = 0
			engine.EntityFrameTime[i] = 0
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
		engine.EntityAlpha[indexUiPlayerAction1] = 0.25
	} else {
		engine.EntityAlpha[indexUiPlayerAction1] = 1.0
	}

	// Handle action2 button.
	if action2CooldownDt > 0 {
		engine.EntityAlpha[indexUiPlayerAction2] = 0.25
	} else {
		engine.EntityAlpha[indexUiPlayerAction2] = 1.0
	}

	// Handle action3 button.
	if action3CooldownDt > 0 {
		engine.EntityAlpha[indexUiPlayerAction3] = 0.25
	} else {
		engine.EntityAlpha[indexUiPlayerAction3] = 1.0
	}

	// Update the player's lives.
	switch playerLives {
	case 1:
		engine.EntityAlpha[indexUiPlayerLive1] = 1.0
		engine.EntityAlpha[indexUiPlayerLive2] = 0.0
		engine.EntityAlpha[indexUiPlayerLive3] = 0.0
		engine.EntityAlpha[indexUiPlayerLive4] = 0.0
	case 2:
		engine.EntityAlpha[indexUiPlayerLive1] = 1.0
		engine.EntityAlpha[indexUiPlayerLive2] = 1.0
		engine.EntityAlpha[indexUiPlayerLive3] = 0.0
		engine.EntityAlpha[indexUiPlayerLive4] = 0.0
	case 3:
		engine.EntityAlpha[indexUiPlayerLive1] = 1.0
		engine.EntityAlpha[indexUiPlayerLive2] = 1.0
		engine.EntityAlpha[indexUiPlayerLive3] = 1.0
		engine.EntityAlpha[indexUiPlayerLive4] = 0.0
	case 4:
		engine.EntityAlpha[indexUiPlayerLive1] = 1.0
		engine.EntityAlpha[indexUiPlayerLive2] = 1.0
		engine.EntityAlpha[indexUiPlayerLive3] = 1.0
		engine.EntityAlpha[indexUiPlayerLive4] = 1.0
	default:
		engine.EntityAlpha[indexUiPlayerLive1] = 0.0
		engine.EntityAlpha[indexUiPlayerLive2] = 0.0
		engine.EntityAlpha[indexUiPlayerLive3] = 0.0
		engine.EntityAlpha[indexUiPlayerLive4] = 0.0
		gameOver = true
	}
}
