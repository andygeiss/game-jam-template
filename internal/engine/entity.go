// Package engine is a small 2D engine for games that TinyGo compiles to
// WebAssembly. This file is the half that needs no browser: entity storage,
// the camera, keyboard state, and the per-frame state machine. It builds and
// tests on every platform. runtime.go is the browser half.
//
// Entities live in a structure of arrays: every attribute is its own slice and
// an entity is an index into all of them. That keeps allocations near zero,
// which matters in TinyGo, and it keeps the code readable — a game reaches
// into engine.EntityX[i] instead of calling a getter.
package engine

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
)

const (
	// Every animation has 8 frames of 100 ms. One fixed shape keeps the
	// spritesheet layout simple: one row per animation, eight columns.
	AnimationFrameCount    = 8
	AnimationFrameDuration = 100

	// BoundingBoxNegativeMargin shrinks every hit box by this many pixels on
	// each side, so sprites have to visibly overlap before they collide.
	BoundingBoxNegativeMargin = 12.0

	// 640x360 is a common pixel-art resolution: it scales to 1080p and 4K by
	// whole numbers, so nothing blurs.
	CanvasWidth  = 640
	CanvasHeight = 360

	// EntitySpeed is the base movement speed in pixels per millisecond.
	EntitySpeed = 0.125
)

// Entity state bits. The engine reads the ones it knows; a game adds its own
// bits above them (see RowIndexMask).
const (
	StateEntityAnimated = uint64(1 << iota)
	StateEntityAnimatedLoop
	StateEntityAutoHide
	StateEntityFaceDown
	StateEntityFaceLeft
	StateEntityFaceRight
	StateEntityFaceUp
	StateEntityIdle
	StateEntityMove
	StateEntityMoveDown
	StateEntityMoveLeft
	StateEntityMoveRight
	StateEntityMoveUp
	StateEntityVisible
)

const (
	moveMask   = StateEntityMoveDown | StateEntityMoveLeft | StateEntityMoveRight | StateEntityMoveUp
	poseMask   = StateEntityFaceLeft | StateEntityFaceRight | StateEntityIdle | StateEntityMove
	facingMask = StateEntityFaceLeft | StateEntityFaceRight
)

// Attributes a game reads and writes directly.
var (
	CamShakeMagnitude float64
	CamShakeTime      float64
	CamTarget         = -1 // entity the camera follows; -1 for none
	HasPlayerInput    bool // true after the first key or click; browsers allow audio only then
	HitStopRemaining  float64
	InputTarget       = -1 // entity that WASD moves; -1 for none
	Key1              bool
	Key2              bool
	Key3              bool
	Key4              bool
	KeyDown           bool
	KeyE              bool
	KeyLeft           bool
	KeyN              bool
	KeyQ              bool
	KeyR              bool
	KeyRight          bool
	KeyT              bool
	KeyUp             bool
	MouseDown         bool
	MouseX            float64
	MouseY            float64

	// RowIndexForState maps a state (masked by RowIndexMask) to the
	// spritesheet row that draws it. The game fills it; the engine only looks
	// things up, so it never needs to know what an "attack" is.
	RowIndexForState map[uint64]int
	RowIndexMask     uint64
)

// Structure of arrays: one slice per attribute, one index per entity.
var (
	EntityAlpha        []float64
	EntityFrameOffset  []int
	EntityFrameTime    []float64
	EntityImageColumn  []int
	EntityImageIndex   []int
	EntityImageRow     []int
	EntityRenderAsUi   []bool
	EntitySpeedFactor  []float64
	EntitySpriteHeight []float64
	EntitySpriteWidth  []float64
	EntityState        []uint64
	EntityX            []float64
	EntityY            []float64
	EntityZ            []int
)

var (
	camBoundsSet         = true
	camX, camY           float64
	camMinX, camMinY     float64
	camMaxX              = float64(CanvasWidth)
	camMaxY              = float64(CanvasHeight)
	camShakeX, camShakeY float64
	drawOrder            []int
)

// AddEntity adds an entity and returns its index. The sprite is drawn
// centered on (x, y).
func AddEntity(state uint64, imgIndex, imgCol, imgRow int, w, h, x, y, alpha float64, z int) (index int) {
	index = len(EntityState)
	EntityAlpha = append(EntityAlpha, alpha)
	EntityFrameOffset = append(EntityFrameOffset, 0)
	EntityFrameTime = append(EntityFrameTime, 0)
	EntityImageColumn = append(EntityImageColumn, imgCol)
	EntityImageIndex = append(EntityImageIndex, imgIndex)
	EntityImageRow = append(EntityImageRow, imgRow)
	EntityRenderAsUi = append(EntityRenderAsUi, false)
	EntitySpeedFactor = append(EntitySpeedFactor, 1.0)
	EntitySpriteHeight = append(EntitySpriteHeight, h)
	EntitySpriteWidth = append(EntitySpriteWidth, w)
	EntityState = append(EntityState, state)
	EntityX = append(EntityX, x)
	EntityY = append(EntityY, y)
	EntityZ = append(EntityZ, z)
	drawOrder = append(drawOrder, index)
	return index
}

// AddTilemap adds one entity per tile from a flat index array. A negative
// index or one past the tileset leaves that cell empty.
func AddTilemap(imgIndex int, tiles []int, tilemapCols, tilemapRows, tilesetCols, tilesetRows int, tileW, tileH float64) {
	maxTile := tilesetCols * tilesetRows
	for j := range tilemapRows {
		for i := range tilemapCols {
			idx := j*tilemapCols + i
			if idx >= len(tiles) {
				return
			}
			t := tiles[idx]
			if t < 0 || t >= maxTile {
				continue
			}
			// Sprites draw centered, so a tile sits at the center of its cell.
			x := float64(i)*tileW + tileW/2
			y := float64(j)*tileH + tileH/2
			AddEntity(StateEntityVisible, imgIndex, t%tilesetCols, t/tilesetCols, tileW, tileH, x, y, 1, 0)
		}
	}
}

// AddUI adds a screen-space entity (the camera does not move it) and returns
// its index.
func AddUI(state uint64, imgIndex, imgCol, imgRow int, w, h, x, y, alpha float64, z int) (index int) {
	index = AddEntity(state, imgIndex, imgCol, imgRow, w, h, x, y, alpha, z)
	SetScreenSpace(index, true)
	return index
}

// BoundingBox returns the left, top, right, and bottom edge of entity i's
// hit box.
func BoundingBox(i int) (l, t, r, b float64) {
	l = EntityX[i] - EntitySpriteWidth[i]/2 + BoundingBoxNegativeMargin
	t = EntityY[i] - EntitySpriteHeight[i]/2 + BoundingBoxNegativeMargin
	r = l + EntitySpriteWidth[i] - BoundingBoxNegativeMargin
	b = t + EntitySpriteHeight[i] - BoundingBoxNegativeMargin
	return l, t, r, b
}

// DeleteEntity removes entity i. Every index above i shifts down by one, so a
// game holding indices must shift them too; CamTarget and InputTarget are
// shifted here. Hiding an entity and reusing it later is cheaper than
// deleting it, which is what a game with many short-lived entities should do.
func DeleteEntity(i int) {
	EntityAlpha = slices.Delete(EntityAlpha, i, i+1)
	EntityFrameOffset = slices.Delete(EntityFrameOffset, i, i+1)
	EntityFrameTime = slices.Delete(EntityFrameTime, i, i+1)
	EntityImageColumn = slices.Delete(EntityImageColumn, i, i+1)
	EntityImageIndex = slices.Delete(EntityImageIndex, i, i+1)
	EntityImageRow = slices.Delete(EntityImageRow, i, i+1)
	EntityRenderAsUi = slices.Delete(EntityRenderAsUi, i, i+1)
	EntitySpeedFactor = slices.Delete(EntitySpeedFactor, i, i+1)
	EntitySpriteHeight = slices.Delete(EntitySpriteHeight, i, i+1)
	EntitySpriteWidth = slices.Delete(EntitySpriteWidth, i, i+1)
	EntityState = slices.Delete(EntityState, i, i+1)
	EntityX = slices.Delete(EntityX, i, i+1)
	EntityY = slices.Delete(EntityY, i, i+1)
	EntityZ = slices.Delete(EntityZ, i, i+1)

	// drawOrder holds indices, not positions, so it is rebuilt in place.
	n := 0
	for _, idx := range drawOrder {
		if idx == i {
			continue
		}
		if idx > i {
			idx--
		}
		drawOrder[n] = idx
		n++
	}
	drawOrder = drawOrder[:n]

	CamTarget = shiftIndex(CamTarget, i)
	InputTarget = shiftIndex(InputTarget, i)
}

// HasCollision reports whether the hit boxes of entities i and j overlap.
func HasCollision(i, j int) bool {
	il, it, ir, ib := BoundingBox(i)
	jl, jt, jr, jb := BoundingBox(j)
	return il < jr && ir > jl && it < jb && ib > jt
}

// InitializeEntities removes every entity. A game calls it once per scene.
func InitializeEntities() {
	EntityAlpha = EntityAlpha[:0]
	EntityFrameOffset = EntityFrameOffset[:0]
	EntityFrameTime = EntityFrameTime[:0]
	EntityImageColumn = EntityImageColumn[:0]
	EntityImageIndex = EntityImageIndex[:0]
	EntityImageRow = EntityImageRow[:0]
	EntityRenderAsUi = EntityRenderAsUi[:0]
	EntitySpeedFactor = EntitySpeedFactor[:0]
	EntitySpriteHeight = EntitySpriteHeight[:0]
	EntitySpriteWidth = EntitySpriteWidth[:0]
	EntityState = EntityState[:0]
	EntityX = EntityX[:0]
	EntityY = EntityY[:0]
	EntityZ = EntityZ[:0]
	drawOrder = drawOrder[:0]
}

// SetScreenSpace switches entity i between screen space (true) and world
// space (false).
func SetScreenSpace(i int, screenSpace bool) {
	EntityRenderAsUi[i] = screenSpace
}

// SetWorldSize sets the area the camera may show. A world smaller than the
// canvas is centered.
func SetWorldSize(width, height float64) {
	camMinX, camMinY = 0, 0
	camMaxX, camMaxY = width, height
	camBoundsSet = true
}

// advanceAnimation moves entity i to its next frame when its frame time is
// up. A one-shot animation stops after the last frame and, with
// StateEntityAutoHide, hides the entity.
func advanceAnimation(i int, dt float64) {
	if EntityState[i]&StateEntityAnimated == 0 {
		return
	}
	EntityFrameTime[i] += dt
	if EntityFrameTime[i] >= AnimationFrameDuration {
		EntityFrameTime[i] = 0
		EntityFrameOffset[i]++
	}
	if EntityFrameOffset[i] < AnimationFrameCount {
		return
	}
	EntityFrameOffset[i] = 0
	if EntityState[i]&StateEntityAnimatedLoop != 0 {
		return
	}
	EntityState[i] &^= StateEntityAnimated
	if EntityState[i]&StateEntityAutoHide != 0 {
		EntityState[i] &^= StateEntityVisible
	}
}

// applyInput writes the WASD keys into entity i's move and facing bits. Facing
// is locked while an action outside the pose bits (an attack, a dash) runs,
// so the sprite does not turn mid-swing.
func applyInput(i int) {
	s := EntityState[i]
	lockFacing := s&RowIndexMask&^poseMask != 0

	s &^= moveMask
	if KeyLeft {
		s |= StateEntityMoveLeft
		if !lockFacing {
			s = s&^facingMask | StateEntityFaceLeft
		}
	}
	if KeyRight {
		s |= StateEntityMoveRight
		if !lockFacing {
			s = s&^facingMask | StateEntityFaceRight
		}
	}
	if KeyUp {
		s |= StateEntityMoveUp
	}
	if KeyDown {
		s |= StateEntityMoveDown
	}
	EntityState[i] = s
}

// handleKeys records a key press or release.
func handleKeys(key string, isDown bool) {
	switch key {
	case "1":
		Key1 = isDown
	case "2":
		Key2 = isDown
	case "3":
		Key3 = isDown
	case "4":
		Key4 = isDown
	case "w", "W":
		KeyUp = isDown
	case "s", "S":
		KeyDown = isDown
	case "a", "A":
		KeyLeft = isDown
	case "d", "D":
		KeyRight = isDown
	case "q", "Q":
		KeyQ = isDown
	case "e", "E":
		KeyE = isDown
	case "r", "R":
		KeyR = isDown
	case "t", "T":
		KeyT = isDown
	case "n", "N":
		KeyN = isDown
	}
}

// shiftIndex returns where a held index points after entity deleted is gone.
func shiftIndex(idx, deleted int) int {
	switch {
	case idx == deleted:
		return -1
	case idx > deleted:
		return idx - 1
	}
	return idx
}

// sortDrawOrder orders entities back to front: lower Z first, then lower
// bottom edge (painter's algorithm), then lower index. It runs once per frame.
func sortDrawOrder() {
	slices.SortStableFunc(drawOrder, func(a, b int) int {
		if EntityZ[a] != EntityZ[b] {
			return cmp.Compare(EntityZ[a], EntityZ[b])
		}
		ya := EntityY[a] + EntitySpriteHeight[a]/2
		yb := EntityY[b] + EntitySpriteHeight[b]/2
		if ya != yb {
			return cmp.Compare(ya, yb)
		}
		return cmp.Compare(a, b)
	})
}

// updateCamera follows CamTarget, keeps the view inside the world, and
// applies the shake.
func updateCamera(dt float64) {
	width, height := float64(CanvasWidth), float64(CanvasHeight)

	if CamTarget >= 0 && CamTarget < len(EntityState) {
		camX = EntityX[CamTarget] - width/2
		camY = EntityY[CamTarget] - height/2
	}

	if camBoundsSet {
		worldW := camMaxX - camMinX
		worldH := camMaxY - camMinY
		if worldW <= width {
			camX = camMinX + (worldW-width)/2
		} else {
			camX = min(max(camX, camMinX), camMaxX-width)
		}
		if worldH <= height {
			camY = camMinY + (worldH-height)/2
		} else {
			camY = min(max(camY, camMinY), camMaxY-height)
		}
	}

	if CamShakeTime > 0 {
		CamShakeTime -= dt
		camShakeX = (rand.Float64()*2 - 1) * CamShakeMagnitude
		camShakeY = (rand.Float64()*2 - 1) * CamShakeMagnitude
		return
	}
	camShakeX, camShakeY = 0, 0
	CamShakeMagnitude = 0
	CamShakeTime = 0
}

// updateStates moves every entity by its move bits and picks its spritesheet
// row. Entities with no move bits and no bit in RowIndexMask — tiles, UI
// sprites — are skipped, which is most of them.
func updateStates(dt float64) {
	if InputTarget >= 0 && InputTarget < len(EntityState) {
		applyInput(InputTarget)
	}

	for i, s := range EntityState {
		if s&RowIndexMask == 0 && s&moveMask == 0 {
			continue
		}

		vx, vy := 0.0, 0.0
		if s&StateEntityMoveLeft != 0 {
			vx--
		}
		if s&StateEntityMoveRight != 0 {
			vx++
		}
		if s&StateEntityMoveUp != 0 {
			vy--
		}
		if s&StateEntityMoveDown != 0 {
			vy++
		}

		if n := vx*vx + vy*vy; n > 0 {
			// Normalize so a diagonal is not faster than a straight line.
			scale := EntitySpeedFactor[i] * EntitySpeed * dt / math.Sqrt(n)
			EntityX[i] += vx * scale
			EntityY[i] += vy * scale
			s = s&^StateEntityIdle | StateEntityMove
		} else {
			s = s&^StateEntityMove | StateEntityIdle
		}

		// An action outside the pose bits picks the row on its own; idle and
		// move are dropped from the key so one action needs one table entry.
		key := s & RowIndexMask
		if key&^poseMask != 0 {
			key &^= StateEntityMove | StateEntityIdle
		}
		if row, ok := RowIndexForState[key]; ok && EntityImageRow[i] != row {
			EntityImageRow[i] = row
			EntityFrameOffset[i] = 0
			EntityFrameTime[i] = 0
		}

		EntityState[i] = s
	}
}
