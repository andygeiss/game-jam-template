//go:build js && wasm

// The game: a hero in a walled arena, ten monsters that walk in from the
// edges, and a boss once they are gone. Everything here is game logic; the
// engine package owns the browser.
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/andygeiss/game-jam-template/internal/engine"
)

// Images, in the order LoadImages receives them.
const (
	indexImageSpritesheet = iota
	indexImageTileset
	indexImageUi
	indexImageBoss
)

// Rows of the spritesheet, one animation each.
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
	indexRowBossAttack
)

// Game-specific state bits, above the engine's own.
const (
	stateAction1 = uint64(1 << (iota + 16))
	stateAction2
	stateAction3
	stateAggressive
	stateDead
	stateInvincible
	stateProjectile
	stateBossProjectile
)

const (
	action1Cooldown    = 1000.0
	action2Cooldown    = 3000.0
	action3Cooldown    = 5000.0
	bossAttackCooldown = 3000.0
	bossDamageDelay    = 1000.0 // the boss takes at most one hit per second
	bossMaxLives       = 10
	monstersMax        = 10
	playerHurtDelay    = 1000.0 // the hero loses at most one life per second
	playerMaxLives     = 4
	projectileSpeed    = 3.0
	tilemapCols        = 33
	tilemapRows        = 21
	tilesetCols        = 3
	tilesetRows        = 5
	tileW              = 32
	tileH              = 32
	worldW             = float64(tilemapCols) * tileW
	worldH             = float64(tilemapRows) * tileH
)

var (
	action1CooldownDt    float64
	action2CooldownDt    float64
	action3CooldownDt    float64
	bossAttackCooldownDt float64
	bossDamageDt         float64
	bossIndex            int
	bossLives            int
	bossSpawned          bool
	gameOver             bool
	gameWon              bool
	indexUiPlayer        int
	indexUiPlayerAction1 int
	indexUiPlayerAction2 int
	indexUiPlayerAction3 int
	indexUiPlayerLives   [playerMaxLives]int
	monstersKilled       int
	playerHurtDt         float64
	playerLives          int
)

func main() {
	// Assets and lookup tables load once; a restart only rebuilds the scene.
	engine.LoadImages(
		"/static/img/spritesheet.png",
		"/static/img/tileset.png",
		"/static/img/ui.png",
		"/static/img/boss.png",
	)
	engine.LoadSounds(
		"/static/audio/attack.wav",
		"/static/audio/hit.wav",
		"/static/audio/music.ogg",
	)

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

	engine.SetRenderUi(renderUI)
	engine.CamTarget = 0
	engine.SetWorldSize(worldW, worldH)

	enterScene()
	engine.Run(update)

	// Keep the Go runtime alive; the browser drives the frame loop.
	select {}
}

// update runs once per frame.
func update(dt float64) {
	if !engine.HasPlayerInput {
		return
	}

	if engine.KeyN {
		engine.KeyN = false
		enterScene()
	}

	if gameOver || gameWon {
		return
	}

	if monstersKilled >= monstersMax && !bossSpawned {
		addBoss()
	}

	engine.PlaySound(2, 0.25, true)

	reduceCooldowns(dt)
	moveMonsters(dt)
	updateBoss(dt)
	moveProjectiles()
	checkCollision()
	updateButtons()

	s := engine.EntityState[0]
	s = handleAction1(s)
	s = handleAction2(s)
	s = handleAction3(s)
	s = handleMovement(s)
	engine.EntityState[0] = s

	if playerLives <= 0 {
		gameOver = true
	}
	if gameOver || gameWon {
		freezePlayer()
	}
}

// addBoss puts the boss at the arena center, announced by a screen shake. It
// appears on top of the hero if the hero is still there, so the hurt cooldown
// covers that moment.
func addBoss() {
	bossIndex = engine.AddEntity(
		engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|stateAggressive,
		indexImageBoss, 0, 0, 96, 96,
		worldW/2, worldH/2, 1, 1,
	)
	bossLives = bossMaxLives
	bossSpawned = true
	bossAttackCooldownDt = 1000.0
	playerHurtDt = playerHurtDelay
	engine.CamShakeMagnitude = 10.0
	engine.CamShakeTime = 1000.0
}

// addMonsters places the monsters outside the arena, spread along the four
// sides, so they walk in one after another.
func addMonsters() {
	const r = 64
	for i := range monstersMax {
		space := rand.Float64() * r
		var x, y float64
		switch i % 4 {
		case 0:
			x, y = -float64(i*r)+space, worldH/2
		case 1:
			x, y = worldW/2, -float64(i*r)+space
		case 2:
			x, y = worldW+float64(i*r)+space, worldH/2
		case 3:
			x, y = worldW/2, worldH+float64(i*r)+space
		}
		// Invisible until it enters the arena (see handleMovement).
		engine.AddEntity(
			engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|stateAggressive,
			indexImageSpritesheet, 0, indexRowMonsterMove, 32, 32,
			x, y, 1, 1,
		)
	}
}

// addPlayer adds the hero as entity 0 at the arena center.
func addPlayer() {
	engine.AddEntity(
		engine.StateEntityAnimated|engine.StateEntityAnimatedLoop|
			engine.StateEntityFaceRight|engine.StateEntityIdle|engine.StateEntityVisible,
		indexImageSpritesheet, 0, 0, 32, 32,
		worldW/2, worldH/2, 1, 1,
	)
}

// addUi builds the HUD: portrait and hearts at the top, ability buttons at
// the bottom.
func addUi() {
	center := float64(engine.CanvasWidth / 2)
	baseY := float64(engine.CanvasHeight)

	ui := func(imgCol, imgRow int, w, h, x, y float64, z int) int {
		return engine.AddUI(engine.StateEntityVisible, indexImageUi, imgCol, imgRow, w, h, x, y, 1, z)
	}
	uiButton := func(bgCol, bgRow, iconCol, iconRow int, x, y float64) int {
		ui(bgCol, bgRow, 32, 32, x, y, 990)
		return ui(iconCol, iconRow, 32, 32, x, y, 999)
	}

	ui(0, 0, 96, 32, center, 32, 990)
	indexUiPlayer = ui(0, 2, 16, 16, center-32, 32, 999)
	for i := range indexUiPlayerLives {
		indexUiPlayerLives[i] = ui(5, 2, 16, 16, center-16+float64(i*16), 32, 999)
	}

	indexUiPlayerAction1 = uiButton(3, 0, 0, 2, center-32, baseY-32)
	indexUiPlayerAction2 = uiButton(3, 0, 1, 2, center, baseY-32)
	indexUiPlayerAction3 = uiButton(3, 0, 2, 2, center+32, baseY-32)
}

// checkCollision handles the hero touching a monster or the boss: the hero
// loses a life, and the monster dies or the boss takes a hit.
func checkCollision() {
	if engine.EntityState[0]&stateInvincible != 0 {
		return
	}
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]
		if s&stateAggressive == 0 || s&stateDead != 0 || !engine.HasCollision(0, i) {
			continue
		}
		hurtPlayer(150, 150)
		if i == bossIndex {
			damageBoss()
		} else {
			killMonster(i)
		}
	}
}

// damageBoss takes one life from the boss, at most once per bossDamageDelay.
func damageBoss() {
	if !bossSpawned || bossIndex < 0 || bossDamageDt > 0 || bossLives <= 0 {
		return
	}
	bossDamageDt = bossDamageDelay
	bossLives--

	engine.CamShakeMagnitude = 4.0
	engine.CamShakeTime = 150
	engine.HitStopRemaining = 70

	if bossLives <= 0 {
		killMonster(bossIndex)
		gameWon = true
	}
}

// enterScene builds a fresh game: every counter and cooldown reset, every
// entity re-created. It runs at start and on N.
func enterScene() {
	action1CooldownDt = 0
	action2CooldownDt = 0
	action3CooldownDt = 0
	bossAttackCooldownDt = 0
	bossDamageDt = 0
	bossIndex = -1
	bossLives = 0
	bossSpawned = false
	gameOver = false
	gameWon = false
	monstersKilled = 0
	playerHurtDt = 0
	playerLives = playerMaxLives

	engine.CamShakeMagnitude = 0
	engine.CamShakeTime = 0
	engine.HitStopRemaining = 0

	// 0 = top-left corner, 1 = top wall, 4 = floor, 11/12 = doors, and so on
	// through the 3x5 tileset.
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

	engine.InitializeEntities()
	addPlayer()
	addUi()
	addMonsters()
	engine.AddTilemap(indexImageTileset, tiles, tilemapCols, tilemapRows, tilesetCols, tilesetRows, tileW, tileH)

	engine.InputTarget = 0
}

// fireAttack2Projectiles sends four projectiles out from the hero: north,
// east, south, and west.
func fireAttack2Projectiles() {
	px, py := engine.EntityX[0], engine.EntityY[0]
	for _, dir := range []uint64{
		engine.StateEntityMoveUp,
		engine.StateEntityMoveRight,
		engine.StateEntityMoveDown,
		engine.StateEntityMoveLeft,
	} {
		state := engine.StateEntityAnimated | engine.StateEntityAnimatedLoop |
			engine.StateEntityVisible | stateProjectile | dir
		idx := spawnOrReuseProjectile(state, indexImageSpritesheet, 0, indexRowAction2, px, py)
		engine.EntitySpeedFactor[idx] = projectileSpeed
	}
}

// fireBossEnergyBalls sends n projectiles out from the boss, spread over the
// eight compass directions.
func fireBossEnergyBalls(n int) {
	if !bossSpawned || bossIndex < 0 || n <= 0 {
		return
	}
	bx, by := engine.EntityX[bossIndex], engine.EntityY[bossIndex]

	directions := []uint64{
		engine.StateEntityMoveUp,
		engine.StateEntityMoveUp | engine.StateEntityMoveRight,
		engine.StateEntityMoveRight,
		engine.StateEntityMoveDown | engine.StateEntityMoveRight,
		engine.StateEntityMoveDown,
		engine.StateEntityMoveDown | engine.StateEntityMoveLeft,
		engine.StateEntityMoveLeft,
		engine.StateEntityMoveUp | engine.StateEntityMoveLeft,
	}
	n = min(n, len(directions))
	step := max(len(directions)/n, 1)

	for i, d := 0, 0; i < n && d < len(directions); i, d = i+1, d+step {
		state := engine.StateEntityAnimated | engine.StateEntityAnimatedLoop |
			engine.StateEntityVisible | stateProjectile | stateBossProjectile | directions[d]
		idx := spawnOrReuseProjectile(state, indexImageSpritesheet, 0, indexRowBossAttack, bx, by)
		engine.EntitySpeedFactor[idx] = projectileSpeed * 0.8
	}
}

// freezePlayer stops the hero once the game has ended. A held key must not
// keep it sliding, so the move bits go too. After a loss the hero is hidden.
func freezePlayer() {
	engine.InputTarget = -1
	s := engine.EntityState[0]
	s &^= stateAction1 | stateAction2 | stateAction3 | stateInvincible |
		engine.StateEntityMoveDown | engine.StateEntityMoveLeft |
		engine.StateEntityMoveRight | engine.StateEntityMoveUp
	if gameOver {
		s = 0
	}
	engine.EntityState[0] = s
	engine.EntitySpeedFactor[0] = 1.0
}

// handleAction1 is the melee strike: a short dash forward with invincibility,
// hitting anything in reach on frames 4 to 6.
func handleAction1(s uint64) (next uint64) {
	if s&stateAction1 != 0 {
		frame := engine.EntityFrameOffset[0]
		if frame == 3 {
			engine.CamShakeMagnitude = 2.5
			engine.CamShakeTime = 100
			engine.EntitySpeedFactor[0] = 2.0
		}
		if frame >= 4 && frame <= 6 {
			for i := 1; i < len(engine.EntityState); i++ {
				ms := engine.EntityState[i]
				if ms&stateAggressive == 0 || ms&engine.StateEntityVisible == 0 || !engine.HasCollision(0, i) {
					continue
				}
				engine.PlaySound(1, 1.0, false)
				if i == bossIndex {
					damageBoss()
				} else {
					killMonster(i)
				}
			}
		}
		if frame == 7 {
			s &^= stateAction1 | stateInvincible
			engine.EntitySpeedFactor[0] = 1.0
		}
	}

	if action1CooldownDt > 0 {
		return s
	}
	if engine.KeyQ && s&stateAction1 == 0 {
		s = s&^engine.StateEntityIdle | stateAction1 | stateInvincible
		engine.EntityFrameOffset[0] = 0
		engine.EntityFrameTime[0] = 0
		engine.PlaySound(0, 1.0, false)
		action1CooldownDt = action1Cooldown
	}
	return s
}

// handleAction2 is the projectile burst. It fires once per key press.
func handleAction2(s uint64) (next uint64) {
	if action2CooldownDt > 0 {
		return s &^ stateAction2
	}
	if engine.KeyE && s&stateAction2 == 0 {
		s |= stateAction2
		fireAttack2Projectiles()
		engine.PlaySound(0, 1.0, false)
		action2CooldownDt = action2Cooldown
	}
	if !engine.KeyE {
		s &^= stateAction2
	}
	return s
}

// handleAction3 is the dash: four times the speed and invincible for one
// animation.
func handleAction3(s uint64) (next uint64) {
	if s&stateAction3 != 0 {
		if engine.EntityFrameOffset[0] == engine.AnimationFrameCount-1 {
			s &^= stateAction3 | stateInvincible
			engine.EntitySpeedFactor[0] = 1.0
		}
		return s
	}
	if action3CooldownDt > 0 {
		return s
	}
	if engine.KeyR {
		s |= stateAction3 | stateInvincible
		engine.EntityFrameOffset[0] = 0
		engine.EntityFrameTime[0] = 0
		engine.EntitySpeedFactor[0] = 4.0
		engine.PlaySound(0, 1.0, false)
		action3CooldownDt = action3Cooldown
	}
	return s
}

// handleMovement keeps the hero inside the walls and reveals monsters that
// have walked into the arena.
func handleMovement(s uint64) (next uint64) {
	engine.EntityX[0] = math.Max(32, math.Min(engine.EntityX[0], worldW-32))
	engine.EntityY[0] = math.Max(32, math.Min(engine.EntityY[0], worldH-32))

	for i := 1; i < len(engine.EntityState); i++ {
		ms := engine.EntityState[i]
		if ms&engine.StateEntityVisible != 0 || ms&stateAggressive == 0 {
			continue
		}
		x, y := engine.EntityX[i], engine.EntityY[i]
		if x >= 32 && x <= worldW-32 && y >= 32 && y <= worldH-32 {
			engine.EntityState[i] = ms | engine.StateEntityVisible
		}
	}
	return s
}

// hurtPlayer takes one life, at most once per playerHurtDelay, with a shake
// and a hit stop as feedback.
func hurtPlayer(shakeTime, hitStop float64) {
	if playerHurtDt > 0 {
		return
	}
	playerHurtDt = playerHurtDelay
	playerLives--
	engine.CamShakeMagnitude = 4.0
	engine.CamShakeTime = shakeTime
	engine.HitStopRemaining = hitStop
}

// killMonster starts the death animation, after which the entity hides
// itself. It stays in the arrays; projectiles reuse hidden slots, monsters
// do not, and there are only ten.
func killMonster(i int) {
	s := engine.EntityState[i]
	s &^= stateAggressive |
		engine.StateEntityMove | engine.StateEntityIdle |
		engine.StateEntityMoveDown | engine.StateEntityMoveLeft |
		engine.StateEntityMoveRight | engine.StateEntityMoveUp |
		engine.StateEntityAnimatedLoop |
		engine.StateEntityFaceLeft | engine.StateEntityFaceRight
	s |= stateDead | engine.StateEntityAnimated | engine.StateEntityAutoHide | engine.StateEntityVisible
	engine.EntityState[i] = s

	engine.HitStopRemaining = 70
	if i != bossIndex {
		monstersKilled++
	}
}

// moveMonsters walks every live monster, the boss included, toward the
// hero. The boss fires at range as well; closing in is what makes it a fight.
func moveMonsters(dt float64) {
	const pxPerMs = 0.05
	px, py := engine.EntityX[0], engine.EntityY[0]
	for i := 1; i < len(engine.EntityState); i++ {
		if engine.EntityState[i]&stateAggressive == 0 {
			continue
		}
		dx := px - engine.EntityX[i]
		dy := py - engine.EntityY[i]
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 0 {
			step := pxPerMs * dt
			engine.EntityX[i] += dx / dist * step
			engine.EntityY[i] += dy / dist * step
		}
	}
}

// moveProjectiles resolves projectile hits and hides projectiles that leave
// the world. The engine moves them; this only checks where they are.
func moveProjectiles() {
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]
		if s&stateProjectile == 0 || s&engine.StateEntityVisible == 0 {
			continue
		}

		x, y := engine.EntityX[i], engine.EntityY[i]
		if x < 0 || x > worldW || y < 0 || y > worldH {
			engine.EntityState[i] = s &^ (engine.StateEntityVisible | engine.StateEntityMove |
				engine.StateEntityMoveUp | engine.StateEntityMoveDown |
				engine.StateEntityMoveLeft | engine.StateEntityMoveRight)
			continue
		}

		if s&stateBossProjectile != 0 {
			if engine.EntityState[0]&stateInvincible == 0 && engine.HasCollision(i, 0) {
				hurtPlayer(200, 100)
				engine.EntityState[i] = s &^ engine.StateEntityVisible
			}
			continue
		}

		for j := 1; j < len(engine.EntityState); j++ {
			ms := engine.EntityState[j]
			if j == i || ms&stateAggressive == 0 || ms&engine.StateEntityVisible == 0 || !engine.HasCollision(i, j) {
				continue
			}
			engine.PlaySound(1, 1.0, false)
			if j == bossIndex {
				damageBoss()
			} else {
				killMonster(j)
			}
			engine.EntityState[i] = s &^ engine.StateEntityVisible
			break
		}
	}
}

// reduceCooldowns counts every timer down to zero.
func reduceCooldowns(dt float64) {
	for _, t := range []*float64{&action1CooldownDt, &action2CooldownDt, &action3CooldownDt, &bossDamageDt, &playerHurtDt} {
		if *t > 0 {
			*t = math.Max(*t-dt, 0)
		}
	}
}

// renderUI draws the text layer: the prompt, the counters, the boss bar, and
// the end screens.
func renderUI() {
	const font = "16px system-ui, sans-serif"
	const bigFont = "24px system-ui, sans-serif"
	centerX := float64(engine.CanvasWidth / 2)
	centerY := float64(engine.CanvasHeight / 2)

	if !engine.HasPlayerInput {
		engine.RenderText(centerX, centerY, "Click to start the game", "white", bigFont, "center")
		return
	}

	alive := 0
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]
		if s&stateAggressive != 0 && s&stateDead == 0 && s&engine.StateEntityVisible != 0 {
			alive++
		}
	}
	engine.RenderText(8, 16, fmt.Sprintf("Monsters alive: %d", alive), "white", font, "left")
	engine.RenderText(8, 32, fmt.Sprintf("Monsters killed: %d", monstersKilled), "yellow", font, "left")

	if bossSpawned && bossLives > 0 {
		bar := "[" + strings.Repeat("█", bossLives) + strings.Repeat("░", bossMaxLives-bossLives) + "]"
		engine.RenderText(centerX, 64, bar, "red", font, "center")
	}

	switch {
	case gameWon:
		engine.RenderText(centerX, centerY, "You win!", "yellow", bigFont, "center")
		engine.RenderText(centerX, centerY+24, "Press N for a new game", "yellow", font, "center")
	case gameOver:
		engine.RenderText(centerX, centerY, "Game over", "red", bigFont, "center")
		engine.RenderText(centerX, centerY+24, "Press N for a new game", "red", font, "center")
	}
}

// spawnOrReuseProjectile reuses a hidden projectile slot when there is one,
// so the entity arrays stop growing after the first volley.
func spawnOrReuseProjectile(state uint64, imgIdx, col, row int, x, y float64) int {
	for i := 1; i < len(engine.EntityState); i++ {
		s := engine.EntityState[i]
		if s&stateProjectile == 0 || s&engine.StateEntityVisible != 0 {
			continue
		}
		engine.EntityFrameOffset[i] = 0
		engine.EntityFrameTime[i] = 0
		engine.EntityImageColumn[i] = col
		engine.EntityImageRow[i] = row
		engine.EntityState[i] = state
		engine.EntityX[i] = x
		engine.EntityY[i] = y
		return i
	}
	return engine.AddEntity(state, imgIdx, col, row, 32, 32, x, y, 1.0, 2)
}

// updateBoss fires a volley of eight energy balls every bossAttackCooldown.
func updateBoss(dt float64) {
	if !bossSpawned || bossIndex < 0 || engine.EntityState[bossIndex]&stateDead != 0 {
		return
	}
	bossAttackCooldownDt -= dt
	if bossAttackCooldownDt > 0 {
		return
	}
	fireBossEnergyBalls(8)
	engine.PlaySound(0, 1.0, false)
	bossAttackCooldownDt = bossAttackCooldown
}

// updateButtons dims an ability button while it cools down and hides the
// hearts the hero has lost.
func updateButtons() {
	buttonAlpha := func(cooldown float64) float64 {
		if cooldown > 0 {
			return 0.25
		}
		return 1.0
	}
	engine.EntityAlpha[indexUiPlayerAction1] = buttonAlpha(action1CooldownDt)
	engine.EntityAlpha[indexUiPlayerAction2] = buttonAlpha(action2CooldownDt)
	engine.EntityAlpha[indexUiPlayerAction3] = buttonAlpha(action3CooldownDt)

	for i, idx := range indexUiPlayerLives {
		if playerLives > i {
			engine.EntityAlpha[idx] = 1.0
		} else {
			engine.EntityAlpha[idx] = 0.0
		}
	}
}
