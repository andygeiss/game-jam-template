//go:build js && wasm

// The game: a hero in a walled arena, ten monsters that walk in from the
// edges, and a boss once they are gone. Everything here is game logic; the
// wisp module owns the browser.
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/andygeiss/wisp-engine"
)

// Images, in the order LoadImages receives them.
const (
	indexImageSpritesheet = iota
	indexImageTileset
	indexImageUi
	indexImageBoss
)

// Sounds, in the order LoadSounds receives them.
const (
	indexSoundAttack = iota
	indexSoundHit
	indexSoundMusic
)

// Rows of the spritesheet, one animation each. The engine can also take an
// Aseprite export, where a row is a named tag; this game stays on the grid,
// so a row index is all it needs. See Engine.Sheets.
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

// Menu entries, in the order they are drawn.
const (
	menuEntryPlay = iota
	menuEntryNewGame
	menuEntryMusic
	menuEntryCount
)

// Game-specific state bits, above the engine's own. The engine claims bits 0
// to 13 and reserves 14 and 15 to grow into, so a game starts at 16.
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

// The three fonts the HUD and the menu draw with.
const (
	uiFont      = "16px system-ui, sans-serif"
	uiFontBig   = "24px system-ui, sans-serif"
	uiFontSmall = "12px system-ui, sans-serif"
)

// zFloor is the layer the tiles land on, and it is below zero on purpose: a
// wisp.Sprite left without a Z lands on layer 0, and inside one layer the
// draw order is by baseline, so a floor sharing a layer with an actor paints
// over whatever stands above its own middle.
const zFloor = -1

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
	gameStarted          bool
	gameWon              bool
	indexUiPlayer        int
	indexUiPlayerAction1 int
	indexUiPlayerAction2 int
	indexUiPlayerAction3 int
	indexUiPlayerLives   [playerMaxLives]int
	menuOpen             bool
	menuSelected         int
	monstersKilled       int
	musicOn              = true
	playerHurtDt         float64
	playerLives          int
)

func main() {
	e := wisp.New(wisp.Config{})

	// Assets and lookup tables load once; a restart only rebuilds the scene.
	e.LoadImages(
		"/static/img/spritesheet.png",
		"/static/img/tileset.png",
		"/static/img/ui.png",
		"/static/img/boss.png",
	)
	e.LoadSounds(
		"/static/audio/attack.wav",
		"/static/audio/hit.wav",
		"/static/audio/music.ogg",
	)

	e.RowMask = wisp.MaskPose |
		stateAction1 | stateAction2 | stateAction3 | stateAggressive | stateDead
	e.RowForState = map[uint64]int{
		wisp.StateFaceRight | stateAction1:   indexRowAction1Right,
		wisp.StateFaceLeft | stateAction1:    indexRowAction1Left,
		wisp.StateFaceRight | stateAction3:   indexRowAction3Right,
		wisp.StateFaceLeft | stateAction3:    indexRowAction3Left,
		wisp.StateFaceRight | wisp.StateIdle: indexRowIdleRight,
		wisp.StateFaceLeft | wisp.StateIdle:  indexRowIdleLeft,
		wisp.StateFaceRight | wisp.StateMove: indexRowMoveRight,
		wisp.StateFaceLeft | wisp.StateMove:  indexRowMoveLeft,
		stateDead:                            indexRowDeath,
	}

	e.RenderUI = func() { renderUI(e) }
	e.SetWorldSize(worldW, worldH)

	enterScene(e)
	openMenu(e)

	// Run blocks until Engine.Stop, so keeping the Go runtime alive is the
	// engine's job rather than a select{} here.
	e.Run(func(dt float64) { update(e, dt) })
}

// update runs once per frame, with the time the world actually moved by.
func update(e *wisp.Engine, dt float64) {
	if !e.Input.Started {
		return
	}

	if e.Input.JustPressed("n") {
		enterScene(e)
		closeMenu(e)
	}

	// P opens the menu and closes it again. There is nothing to pause once
	// the game has ended, so the end screens keep it shut.
	if e.Input.JustPressed("p") {
		switch {
		case menuOpen:
			closeMenu(e)
		case !gameOver && !gameWon:
			openMenu(e)
		}
	}

	if menuOpen {
		updateMenu(e)
		return
	}

	if gameOver || gameWon {
		return
	}

	if monstersKilled >= monstersMax && !bossSpawned {
		addBoss(e)
	}

	if musicOn {
		// The 0.25 the old call site carried is Audio.MusicVolume now, which
		// the tuning menu can reach.
		e.PlayMusic(indexSoundMusic, 1)
	}

	reduceCooldowns(dt)
	moveMonsters(e, dt)
	updateBoss(e, dt)
	moveProjectiles(e)
	checkCollision(e)
	updateButtons(e)

	s := e.State[0]
	s = handleAction1(e, s)
	s = handleAction2(e, s)
	s = handleAction3(e, s)
	s = handleMovement(e, s)
	e.State[0] = s

	if playerLives <= 0 {
		gameOver = true
	}
	if gameOver || gameWon {
		freezePlayer(e)
	}
}

// addBoss puts the boss at the arena center, announced by a screen shake. It
// appears on top of the hero if the hero is still there, so the hurt cooldown
// covers that moment.
func addBoss(e *wisp.Engine) {
	bossIndex = e.Add(wisp.Sprite{
		Height: 96,
		Image:  indexImageBoss,
		State:  wisp.StateAnimated | wisp.StateAnimatedLoop | stateAggressive,
		Width:  96,
		X:      worldW / 2,
		Y:      worldH / 2,
		Z:      1,
	})
	bossLives = bossMaxLives
	bossSpawned = true
	bossAttackCooldownDt = 1000.0
	playerHurtDt = playerHurtDelay
	e.Impact(e.Feel.Heavy)
}

// addMonsters places the monsters outside the arena, spread along the four
// sides, so they walk in one after another.
func addMonsters(e *wisp.Engine) {
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
		e.Add(wisp.Sprite{
			Height: 32,
			Image:  indexImageSpritesheet,
			Row:    indexRowMonsterMove,
			State:  wisp.StateAnimated | wisp.StateAnimatedLoop | stateAggressive,
			Width:  32,
			X:      x,
			Y:      y,
			Z:      1,
		})
	}
}

// addPlayer adds the hero as entity 0 at the arena center.
func addPlayer(e *wisp.Engine) {
	e.Add(wisp.Sprite{
		Height: 32,
		Image:  indexImageSpritesheet,
		State: wisp.StateAnimated | wisp.StateAnimatedLoop |
			wisp.StateFaceRight | wisp.StateIdle | wisp.StateVisible,
		Width: 32,
		X:     worldW / 2,
		Y:     worldH / 2,
		Z:     1,
	})
}

// addUi builds the HUD: portrait and hearts at the top, ability buttons at
// the bottom.
func addUi(e *wisp.Engine) {
	center := e.Width / 2
	baseY := e.Height

	ui := func(imgCol, imgRow int, w, h, x, y float64, z int) int {
		return e.Add(wisp.Sprite{
			Column:      imgCol,
			Height:      h,
			Image:       indexImageUi,
			Row:         imgRow,
			ScreenSpace: true,
			State:       wisp.StateVisible,
			Width:       w,
			X:           x,
			Y:           y,
			Z:           z,
		})
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
func checkCollision(e *wisp.Engine) {
	if e.State[0]&stateInvincible != 0 {
		return
	}
	for i := 1; i < e.Slots(); i++ {
		if !e.Live(i) {
			continue
		}
		s := e.State[i]
		if s&stateAggressive == 0 || s&stateDead != 0 || !e.HasCollision(0, i) {
			continue
		}
		hurtPlayer(e)
		if i == bossIndex {
			damageBoss(e)
		} else {
			killMonster(e, i)
		}
	}
}

// closeMenu hides the menu and starts time again. The hero gets the controls
// back unless the game has already ended.
func closeMenu(e *wisp.Engine) {
	menuOpen = false
	gameStarted = true
	e.Paused = false
	if !gameOver && !gameWon {
		e.InputTarget = 0
	}
}

// damageBoss takes one life from the boss, at most once per bossDamageDelay.
func damageBoss(e *wisp.Engine) {
	if !bossSpawned || bossIndex < 0 || bossDamageDt > 0 || bossLives <= 0 {
		return
	}
	bossDamageDt = bossDamageDelay
	bossLives--

	e.Impact(e.Feel.Medium)

	if bossLives <= 0 {
		killMonster(e, bossIndex)
		gameWon = true
	}
}

// enterScene builds a fresh game: every counter and cooldown reset, every
// entity re-created. It runs at start and on N.
func enterScene(e *wisp.Engine) {
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

	// A duration of zero cancels whichever of the two is still running.
	e.HitStop(0, 0)
	e.Shake(0, 0)

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

	e.Reset()
	addPlayer(e)
	addUi(e)
	addMonsters(e)
	e.AddTilemap(wisp.Tilemap{
		Cols:        tilemapCols,
		Height:      tileH,
		Image:       indexImageTileset,
		Rows:        tilemapRows,
		Tiles:       tiles,
		TilesetCols: tilesetCols,
		TilesetRows: tilesetRows,
		Width:       tileW,
		Z:           zFloor,
	})

	// Reset clears both, so the hero has to be claimed again after it.
	e.CamTarget = 0
	e.InputTarget = 0
}

// fireAttack2Projectiles sends four projectiles out from the hero: north,
// east, south, and west.
func fireAttack2Projectiles(e *wisp.Engine) {
	px, py := e.X[0], e.Y[0]
	for _, dir := range []uint64{
		wisp.StateMoveUp,
		wisp.StateMoveRight,
		wisp.StateMoveDown,
		wisp.StateMoveLeft,
	} {
		state := wisp.StateAnimated | wisp.StateAnimatedLoop |
			wisp.StateVisible | stateProjectile | dir
		idx := spawnOrReuseProjectile(e, state, indexImageSpritesheet, 0, indexRowAction2, px, py)
		e.SpeedFactor[idx] = projectileSpeed
	}
}

// fireBossEnergyBalls sends n projectiles out from the boss, spread over the
// eight compass directions.
func fireBossEnergyBalls(e *wisp.Engine, n int) {
	if !bossSpawned || bossIndex < 0 || n <= 0 {
		return
	}
	bx, by := e.X[bossIndex], e.Y[bossIndex]

	directions := []uint64{
		wisp.StateMoveUp,
		wisp.StateMoveUp | wisp.StateMoveRight,
		wisp.StateMoveRight,
		wisp.StateMoveDown | wisp.StateMoveRight,
		wisp.StateMoveDown,
		wisp.StateMoveDown | wisp.StateMoveLeft,
		wisp.StateMoveLeft,
		wisp.StateMoveUp | wisp.StateMoveLeft,
	}
	n = min(n, len(directions))
	step := max(len(directions)/n, 1)

	for i, d := 0, 0; i < n && d < len(directions); i, d = i+1, d+step {
		state := wisp.StateAnimated | wisp.StateAnimatedLoop |
			wisp.StateVisible | stateProjectile | stateBossProjectile | directions[d]
		idx := spawnOrReuseProjectile(e, state, indexImageSpritesheet, 0, indexRowBossAttack, bx, by)
		e.SpeedFactor[idx] = projectileSpeed * 0.8
	}
}

// freezePlayer stops the hero once the game has ended. A held key must not
// keep it sliding, so the move bits go too. After a loss the hero is hidden.
func freezePlayer(e *wisp.Engine) {
	e.InputTarget = -1
	s := e.State[0]
	s &^= stateAction1 | stateAction2 | stateAction3 | stateInvincible | wisp.MaskMove
	if gameOver {
		s = 0
	}
	e.State[0] = s
	e.SpeedFactor[0] = 1.0
}

// handleAction1 is the melee strike: a short dash forward with invincibility,
// hitting anything in reach on frames 4 to 6.
func handleAction1(e *wisp.Engine, s uint64) (next uint64) {
	if s&stateAction1 != 0 {
		frame := e.FrameOffset[0]
		if frame == 3 {
			e.Shake(e.Feel.Light.ShakeMagnitude, e.Feel.Light.ShakeDuration)
			e.SpeedFactor[0] = 2.0
		}
		if frame >= 4 && frame <= 6 {
			for i := 1; i < e.Slots(); i++ {
				if !e.Live(i) {
					continue
				}
				ms := e.State[i]
				if ms&stateAggressive == 0 || ms&wisp.StateVisible == 0 || !e.HasCollision(0, i) {
					continue
				}
				e.PlaySound(indexSoundHit, 1.0)
				if i == bossIndex {
					damageBoss(e)
				} else {
					killMonster(e, i)
				}
			}
		}
		// The last frame, rather than the literal 7: Animation.FrameCount is
		// a knob the tuning menu can move.
		if frame == e.Animation.FrameCount-1 {
			s &^= stateAction1 | stateInvincible
			e.SpeedFactor[0] = 1.0
		}
	}

	if action1CooldownDt > 0 {
		return s
	}
	if e.Input.JustPressed("q") && s&stateAction1 == 0 {
		s = s&^wisp.StateIdle | stateAction1 | stateInvincible
		e.FrameOffset[0] = 0
		e.FrameTime[0] = 0
		e.PlaySound(indexSoundAttack, 1.0)
		action1CooldownDt = action1Cooldown
	}
	return s
}

// handleAction2 is the projectile burst. It fires once per key press; the
// state bit outlives the press only to keep the hero facing where it fired.
func handleAction2(e *wisp.Engine, s uint64) (next uint64) {
	if action2CooldownDt > 0 {
		return s &^ stateAction2
	}
	if e.Input.JustPressed("e") {
		s |= stateAction2
		fireAttack2Projectiles(e)
		e.PlaySound(indexSoundAttack, 1.0)
		action2CooldownDt = action2Cooldown
	}
	if !e.Input.Down("e") {
		s &^= stateAction2
	}
	return s
}

// handleAction3 is the dash: four times the speed and invincible for one
// animation.
func handleAction3(e *wisp.Engine, s uint64) (next uint64) {
	if s&stateAction3 != 0 {
		if e.FrameOffset[0] == e.Animation.FrameCount-1 {
			s &^= stateAction3 | stateInvincible
			e.SpeedFactor[0] = 1.0
		}
		return s
	}
	if action3CooldownDt > 0 {
		return s
	}
	if e.Input.JustPressed("r") {
		s |= stateAction3 | stateInvincible
		e.FrameOffset[0] = 0
		e.FrameTime[0] = 0
		e.SpeedFactor[0] = 4.0
		e.PlaySound(indexSoundAttack, 1.0)
		action3CooldownDt = action3Cooldown
	}
	return s
}

// handleMovement keeps the hero inside the walls and reveals monsters that
// have walked into the arena.
func handleMovement(e *wisp.Engine, s uint64) (next uint64) {
	e.X[0] = math.Max(32, math.Min(e.X[0], worldW-32))
	e.Y[0] = math.Max(32, math.Min(e.Y[0], worldH-32))

	for i := 1; i < e.Slots(); i++ {
		if !e.Live(i) {
			continue
		}
		ms := e.State[i]
		if ms&wisp.StateVisible != 0 || ms&stateAggressive == 0 {
			continue
		}
		x, y := e.X[i], e.Y[i]
		if x >= 32 && x <= worldW-32 && y >= 32 && y <= worldH-32 {
			e.State[i] = ms | wisp.StateVisible
		}
	}
	return s
}

// hurtPlayer takes one life, at most once per playerHurtDelay, with a shake
// and a hit stop as feedback.
func hurtPlayer(e *wisp.Engine) {
	if playerHurtDt > 0 {
		return
	}
	playerHurtDt = playerHurtDelay
	playerLives--
	e.Impact(e.Feel.Medium)
}

// killMonster starts the death animation, after which the entity hides
// itself. It stays in the arrays; projectiles reuse hidden slots, monsters
// do not, and there are only ten.
func killMonster(e *wisp.Engine, i int) {
	s := e.State[i]
	s &^= stateAggressive |
		wisp.StateMove | wisp.StateIdle | wisp.MaskMove |
		wisp.StateAnimatedLoop |
		wisp.StateFaceLeft | wisp.StateFaceRight
	s |= stateDead | wisp.StateAnimated | wisp.StateAutoHide | wisp.StateVisible
	e.State[i] = s

	e.Impact(e.Feel.Light)
	if i != bossIndex {
		monstersKilled++
	}
}

// moveMonsters walks every live monster, the boss included, toward the
// hero. The boss fires at range as well; closing in is what makes it a fight.
func moveMonsters(e *wisp.Engine, dt float64) {
	const pxPerMs = 0.05
	px, py := e.X[0], e.Y[0]
	for i := 1; i < e.Slots(); i++ {
		if !e.Live(i) || e.State[i]&stateAggressive == 0 {
			continue
		}
		dx := px - e.X[i]
		dy := py - e.Y[i]
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 0 {
			step := pxPerMs * dt
			e.X[i] += dx / dist * step
			e.Y[i] += dy / dist * step
		}
	}
}

// moveProjectiles resolves projectile hits and hides projectiles that leave
// the world. The engine moves them; this only checks where they are.
func moveProjectiles(e *wisp.Engine) {
	for i := 1; i < e.Slots(); i++ {
		if !e.Live(i) {
			continue
		}
		s := e.State[i]
		if s&stateProjectile == 0 || s&wisp.StateVisible == 0 {
			continue
		}

		x, y := e.X[i], e.Y[i]
		if x < 0 || x > worldW || y < 0 || y > worldH {
			e.State[i] = s &^ (wisp.StateVisible | wisp.StateMove | wisp.MaskMove)
			continue
		}

		if s&stateBossProjectile != 0 {
			if e.State[0]&stateInvincible == 0 && e.HasCollision(i, 0) {
				hurtPlayer(e)
				e.State[i] = s &^ wisp.StateVisible
			}
			continue
		}

		for j := 1; j < e.Slots(); j++ {
			if !e.Live(j) {
				continue
			}
			ms := e.State[j]
			if j == i || ms&stateAggressive == 0 || ms&wisp.StateVisible == 0 || !e.HasCollision(i, j) {
				continue
			}
			e.PlaySound(indexSoundHit, 1.0)
			if j == bossIndex {
				damageBoss(e)
			} else {
				killMonster(e, j)
			}
			e.State[i] = s &^ wisp.StateVisible
			break
		}
	}
}

// openMenu freezes the game and shows the menu. The hero's move bits go with
// it, so a key held down when the menu opens does not keep it walking once
// the menu closes.
func openMenu(e *wisp.Engine) {
	menuOpen = true
	menuSelected = menuEntryPlay
	e.Paused = true
	e.InputTarget = -1
	e.State[0] &^= wisp.MaskMove
	e.PauseMusic()
}

// reduceCooldowns counts every timer down to zero.
func reduceCooldowns(dt float64) {
	for _, t := range []*float64{&action1CooldownDt, &action2CooldownDt, &action3CooldownDt, &bossDamageDt, &playerHurtDt} {
		if *t > 0 {
			*t = math.Max(*t-dt, 0)
		}
	}
}

// renderMenu draws the menu over the frozen game: a dimmed backdrop, a panel,
// the entries, and the keys that work everywhere else.
func renderMenu(e *wisp.Engine) {
	const panelW, panelH = 320.0, 208.0
	const dim = "rgba(255, 255, 255, 0.55)"

	centerX := e.Width / 2
	panelX := centerX - panelW/2
	panelY := (e.Height - panelH) / 2

	// The border is a slightly larger rectangle behind the panel; the engine
	// fills rectangles and draws no outlines.
	e.Rect(0, 0, e.Width, e.Height, "rgba(0, 0, 0, 0.72)")
	e.Rect(panelX-2, panelY-2, panelW+4, panelH+4, "rgba(255, 255, 255, 0.25)")
	e.Rect(panelX, panelY, panelW, panelH, "rgba(18, 20, 26, 0.94)")

	title := "Wisp Engine"
	if gameStarted {
		title = "Paused"
	}
	e.Text(centerX, panelY+36, title, "yellow", uiFontBig, "center")

	entries := [menuEntryCount]string{
		menuEntryPlay:    "Start game",
		menuEntryNewGame: "New game",
		menuEntryMusic:   "Music: on",
	}
	if gameStarted {
		entries[menuEntryPlay] = "Resume"
	}
	if !musicOn {
		entries[menuEntryMusic] = "Music: off"
	}

	// Entries are left-aligned at a fixed column, so the marker in front of
	// the highlighted one does not shift the words.
	entryX := centerX - 72
	for i, label := range entries {
		y := panelY + 88 + float64(i)*26
		if i != menuSelected {
			e.Text(entryX, y, label, dim, uiFont, "left")
			continue
		}
		e.Text(entryX-20, y, "▶", "yellow", uiFont, "left")
		e.Text(entryX, y, label, "yellow", uiFont, "left")
	}

	e.Text(centerX, panelY+panelH-28, "W and S choose, Enter confirms", dim, uiFontSmall, "center")
	e.Text(centerX, e.Height-18,
		"WASD move   Q strike   E burst   R dash   P menu   N new game   M tune   F fullscreen",
		dim, uiFontSmall, "center")
}

// renderUI draws the text layer: the prompt, the counters, the boss bar, and
// the end screens.
func renderUI(e *wisp.Engine) {
	centerX := e.Width / 2
	centerY := e.Height / 2

	if !e.Input.Started {
		e.Text(centerX, centerY, "Click to start the game", "white", uiFontBig, "center")
		return
	}

	alive := 0
	for i := 1; i < e.Slots(); i++ {
		if !e.Live(i) {
			continue
		}
		s := e.State[i]
		if s&stateAggressive != 0 && s&stateDead == 0 && s&wisp.StateVisible != 0 {
			alive++
		}
	}
	e.Text(8, 16, fmt.Sprintf("Monsters alive: %d", alive), "white", uiFont, "left")
	e.Text(8, 32, fmt.Sprintf("Monsters killed: %d", monstersKilled), "yellow", uiFont, "left")

	if bossSpawned && bossLives > 0 {
		bar := "[" + strings.Repeat("█", bossLives) + strings.Repeat("░", bossMaxLives-bossLives) + "]"
		e.Text(centerX, 64, bar, "red", uiFont, "center")
	}

	switch {
	case menuOpen:
		renderMenu(e)
	case gameWon:
		e.Text(centerX, centerY, "You win!", "yellow", uiFontBig, "center")
		e.Text(centerX, centerY+24, "Press N for a new game", "yellow", uiFont, "center")
	case gameOver:
		e.Text(centerX, centerY, "Game over", "red", uiFontBig, "center")
		e.Text(centerX, centerY+24, "Press N for a new game", "red", uiFont, "center")
	}
}

// spawnOrReuseProjectile reuses a hidden projectile slot when there is one,
// so the entity arrays stop growing after the first volley.
func spawnOrReuseProjectile(e *wisp.Engine, state uint64, imgIdx, col, row int, x, y float64) int {
	for i := 1; i < e.Slots(); i++ {
		if !e.Live(i) {
			continue
		}
		s := e.State[i]
		if s&stateProjectile == 0 || s&wisp.StateVisible != 0 {
			continue
		}
		e.FrameOffset[i] = 0
		e.FrameTime[i] = 0
		e.ImageColumn[i] = col
		e.ImageRow[i] = row
		e.State[i] = state
		e.X[i] = x
		e.Y[i] = y
		return i
	}
	return e.Add(wisp.Sprite{
		Column: col,
		Height: 32,
		Image:  imgIdx,
		Row:    row,
		State:  state,
		Width:  32,
		X:      x,
		Y:      y,
		Z:      2,
	})
}

// updateBoss fires a volley of eight energy balls every bossAttackCooldown.
func updateBoss(e *wisp.Engine, dt float64) {
	if !bossSpawned || bossIndex < 0 || e.State[bossIndex]&stateDead != 0 {
		return
	}
	bossAttackCooldownDt -= dt
	if bossAttackCooldownDt > 0 {
		return
	}
	fireBossEnergyBalls(e, 8)
	e.PlaySound(indexSoundAttack, 1.0)
	bossAttackCooldownDt = bossAttackCooldown
}

// updateButtons dims an ability button while it cools down and hides the
// hearts the hero has lost.
func updateButtons(e *wisp.Engine) {
	buttonAlpha := func(cooldown float64) float64 {
		if cooldown > 0 {
			return 0.25
		}
		return 1.0
	}
	e.Alpha[indexUiPlayerAction1] = buttonAlpha(action1CooldownDt)
	e.Alpha[indexUiPlayerAction2] = buttonAlpha(action2CooldownDt)
	e.Alpha[indexUiPlayerAction3] = buttonAlpha(action3CooldownDt)

	for i, idx := range indexUiPlayerLives {
		if playerLives > i {
			e.Alpha[idx] = 1.0
		} else {
			e.Alpha[idx] = 0.0
		}
	}
}

// updateMenu moves the highlight with W and S and runs the highlighted entry
// on Enter. Each key is an edge, so one press is one step and the hero does
// not act on the key that picked an entry.
func updateMenu(e *wisp.Engine) {
	if e.Input.JustPressed("w") {
		menuSelected = (menuSelected + menuEntryCount - 1) % menuEntryCount
	}
	if e.Input.JustPressed("s") {
		menuSelected = (menuSelected + 1) % menuEntryCount
	}
	if !e.Input.JustPressed("Enter") {
		return
	}

	switch menuSelected {
	case menuEntryPlay:
		closeMenu(e)
	case menuEntryNewGame:
		enterScene(e)
		closeMenu(e)
	case menuEntryMusic:
		musicOn = !musicOn
		if !musicOn {
			e.PauseMusic()
		}
	}
}
