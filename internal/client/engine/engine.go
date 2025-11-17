//go:build js && wasm

package engine

import (
	"math"
	"math/rand/v2"
	"sort"
	"syscall/js"
)

const (
	// Wisp uses animations with 8 frames and a duration of 100 ms.
	// It provides a simple and efficient way to create animations.
	// It is a sweet spot between smoothness and performance.
	AnimationFrameCount    = 8
	AnimationFrameDuration = 100

	// BoundingBoxNegativeMargin is the margin used for bounding box calculations.
	// It is used to shrink the bounding box to make it more "crisp".
	BoundingBoxNegativeMargin = 12.0

	// A canvas size of 640x360 is often used for pixel art games.
	// This resolution can be easily scaled up or down without losing quality.
	// We use some default constants here to make the implementation simpler.
	// This engine has a simple concept and does not compete with other engines.
	CanvasWidth  = 640
	CanvasHeight = 360

	// Base speed for the entity movement.
	EntitySpeed = 0.125
)

const (
	// An entity has different states which determine its behavior.
	// The engine uses these states to provide basic functionality.
	// Only visible entities will be rendered.
	// Only animated entities will receive frame updates.
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

var (
	// Define public attributes.
	CamShakeMagnitude float64
	CamShakeTime      float64
	CamTarget         int // -1 no target
	HasPlayerInput    bool
	HitStopRemaining  float64
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
	RowIndexForState  map[uint64]int
	RowIndexMask      uint64

	// Structure of Arrays to store the entities.
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
	// Define private attributes.
	camBoundsSet         bool
	camX, camY           float64
	camMinX, camMinY     float64
	camMaxX, camMaxY     float64
	camShakeX, camShakeY float64
	canvas               js.Value
	ctx                  js.Value
	doc                  js.Value
	drawOrder            []int
	lastToggleMs         float64
	images               []js.Value
	imagesLoaded         int
	lastTs               float64
	loopFn               js.Func
	sounds               []js.Value
	soundsLoaded         int
	renderUi             func()
)

// AddEntity adds a new entity to the engine and returns its index.
func AddEntity(state uint64, imgIndex, imgCol, imgRow int, w, h, x, y, alpha float64, z int) (index int) {
	index = len(EntityState)
	EntityAlpha = append(EntityAlpha, alpha)
	EntityFrameOffset = append(EntityFrameOffset, 0)
	EntityFrameTime = append(EntityFrameTime, 0)
	EntitySpriteHeight = append(EntitySpriteHeight, h)
	EntityImageIndex = append(EntityImageIndex, imgIndex)
	EntityImageColumn = append(EntityImageColumn, imgCol)
	EntityImageRow = append(EntityImageRow, imgRow)
	EntitySpeedFactor = append(EntitySpeedFactor, 1.0)
	EntityState = append(EntityState, state)
	EntityRenderAsUi = append(EntityRenderAsUi, false) // false = world-space
	EntitySpriteWidth = append(EntitySpriteWidth, w)
	EntityX = append(EntityX, x)
	EntityY = append(EntityY, y)
	EntityZ = append(EntityZ, z)
	drawOrder = append(drawOrder, index)
	return index
}

// AddTilemap creates a grid of tile entities from a flat index array.
func AddTilemap(imgIndex int, tiles []int, tilemapCols, tilemapRows, tilesetCols, tilesetRows int, tileW, tileH float64) {
	maxTile := tilesetCols * tilesetRows
	for j := 0; j < tilemapRows; j++ {
		for i := 0; i < tilemapCols; i++ {
			// Calculate the tile index in the flat array.
			idx := j*tilemapCols + i
			if idx < 0 || idx >= len(tiles) {
				continue
			}

			// Get the tile from the flat array.
			t := tiles[idx]
			if t < 0 || t >= maxTile {
				// Skip empty tiles.
				continue
			}

			// Place tiles on grid centers (engine draws centered).
			imgCol := t % tilesetCols
			imgRow := t / tilesetCols
			x := float64(i)*tileW + tileW/2
			y := float64(j)*tileH + tileH/2
			AddEntity(StateEntityVisible, imgIndex, imgCol, imgRow, tileW, tileH, x, y, 1, 0)
		}
	}
}

// AddUI adds a UI (screen-space) entity and returns its index.
func AddUI(state uint64, imgIndex, imgCol, imgRow int, w, h, x, y, alpha float64, z int) (index int) {
	index = len(EntityState)
	AddEntity(state, imgIndex, imgCol, imgRow, w, h, x, y, alpha, z)
	SetScreenSpace(index, true)
	return index
}

// BoundingBox returns the left, top, right, bottom for entity i.
func BoundingBox(i int) (l, t, r, b float64) {
	l = (EntityX[i] - EntitySpriteWidth[i]/2) + BoundingBoxNegativeMargin
	t = EntityY[i] - EntitySpriteHeight[i]/2 + BoundingBoxNegativeMargin
	r = l + EntitySpriteWidth[i] - BoundingBoxNegativeMargin
	b = t + EntitySpriteHeight[i] - BoundingBoxNegativeMargin
	return
}

// DeleteEntity removes entity i from the entity list.
func DeleteEntity(i int) {
	EntityAlpha = append(EntityAlpha[:i], EntityAlpha[i+1:]...)
	EntityFrameOffset = append(EntityFrameOffset[:i], EntityFrameOffset[i+1:]...)
	EntityFrameTime = append(EntityFrameTime[:i], EntityFrameTime[i+1:]...)
	EntitySpriteHeight = append(EntitySpriteHeight[:i], EntitySpriteHeight[i+1:]...)
	EntityImageIndex = append(EntityImageIndex[:i], EntityImageIndex[i+1:]...)
	EntityImageColumn = append(EntityImageColumn[:i], EntityImageColumn[i+1:]...)
	EntityImageRow = append(EntityImageRow[:i], EntityImageRow[i+1:]...)
	EntitySpeedFactor = append(EntitySpeedFactor[:i], EntitySpeedFactor[i+1:]...)
	EntityState = append(EntityState[:i], EntityState[i+1:]...)
	EntityRenderAsUi = append(EntityRenderAsUi[:i], EntityRenderAsUi[i+1:]...)
	EntitySpriteWidth = append(EntitySpriteWidth[:i], EntitySpriteWidth[i+1:]...)
	EntityX = append(EntityX[:i], EntityX[i+1:]...)
	EntityY = append(EntityY[:i], EntityY[i+1:]...)
	EntityZ = append(EntityZ[:i], EntityZ[i+1:]...)
	drawOrder = append(drawOrder[:i], drawOrder[i+1:]...)
}

// HasCollision returns true if entity i collides with entity j.
func HasCollision(i, j int) bool {
	il, it, ir, ib := BoundingBox(i)
	jl, jt, jr, jb := BoundingBox(j)
	return il < jr && ir > jl && it < jb && ib > jt
}

// InitializeEntities initializes the entity state.
func InitializeEntities() {
	EntityAlpha = make([]float64, 0)
	EntityFrameOffset = make([]int, 0)
	EntityFrameTime = make([]float64, 0)
	EntitySpriteHeight = make([]float64, 0)
	EntityImageIndex = make([]int, 0)
	EntityImageColumn = make([]int, 0)
	EntityImageRow = make([]int, 0)
	EntitySpeedFactor = make([]float64, 0)
	EntityState = make([]uint64, 0)
	EntityRenderAsUi = make([]bool, 0)
	EntitySpriteWidth = make([]float64, 0)
	EntityX = make([]float64, 0)
	EntityY = make([]float64, 0)
	EntityZ = make([]int, 0)
	drawOrder = make([]int, 0)
}

// LoadImages loads an image from the given path.
func LoadImages(paths ...string) {
	for _, path := range paths {
		val := js.Global().Get("Image").New()
		val.Set("src", path)
		val.Set("onload", js.FuncOf(func(this js.Value, args []js.Value) any {
			imagesLoaded++
			return nil
		}))
		images = append(images, val)
	}
}

// LoadSounds loads a sound from the given path.
func LoadSounds(paths ...string) {
	for _, path := range paths {
		val := js.Global().Get("Audio").New()
		val.Set("src", path)
		val.Set("oncanplaythrough", js.FuncOf(func(this js.Value, args []js.Value) any {
			soundsLoaded++
			return nil
		}))
		sounds = append(sounds, val)
	}
}

// PlaySound plays a sound from the given index.
func PlaySound(index int, volume float64, loop bool) {
	if sounds[index].Truthy() && sounds[index].Get("paused").Bool() {
		sounds[index].Set("loop", loop)
		sounds[index].Set("volume", volume)
		sounds[index].Call("play")
	}
}

// RenderText renders text at the given position with the specified color.
func RenderText(x, y float64, text, color, font, align string) {
	ctx.Set("fillStyle", color)
	ctx.Set("font", font)
	ctx.Set("textAlign", align)
	ctx.Set("textBaseline", "middle")
	ctx.Call("fillText", text, x, y)
}

// Run initializes the engine and starts the main loop.
// This function is called every frame with the delta time (in ms).
func Run(updateScene func(dt float64)) {
	// Create a few shortcuts.
	doc = js.Global().Get("document")
	perf := js.Global().Get("performance")

	// Initialize the canvas element first.
	canvas = doc.Call("createElement", "canvas")
	canvas.Set("width", CanvasWidth)
	canvas.Set("height", CanvasHeight)
	doc.Get("body").Call("appendChild", canvas)

	// Get the canvas element's context.
	ctx = canvas.Call("getContext", "2d")

	// Initialize the last timestamp of the animation frame.
	lastTs = perf.Call("now").Float()
	lastToggleMs = lastTs

	// Set the camera target entity to index -1 initially (none).
	CamTarget = -1
	SetWorldSize(CanvasWidth, CanvasHeight)

	// Add event listeners for keyboard and mouse events.
	addEventListeners()

	// Ensure that the loop function is not garbage collected.
	loopFn = js.FuncOf(func(this js.Value, args []js.Value) any {
		// Limit the maximum delta time to 50 milliseconds.
		// Record the time since the last toggle.
		now := perf.Call("now").Float()
		dt := now - lastTs
		if dt > 50 {
			dt = 50
		}
		lastTs = now
		lastToggleMs += dt

		// Check if hit stop is active and freeze the game.
		if HitStopRemaining > 0 {
			HitStopRemaining -= dt
			if HitStopRemaining <= 0 {
				HitStopRemaining = 0
			}
			dt = 0
		}

		updateScene(dt)
		updateStates(dt)
		updateCamera(dt)

		// Clear the canvas.
		ctx.Call("clearRect", 0, 0, CanvasWidth, CanvasHeight)

		// Check if all assets are loaded.
		// Skip rendering the entities if not all assets are loaded.
		allAssetsLoaded := imagesLoaded == len(images) && soundsLoaded == len(sounds)
		if !allAssetsLoaded {
			ctx.Set("fillStyle", "white")
			ctx.Set("font", "24px Arial")
			ctx.Set("textAlign", "center")
			ctx.Set("textBaseline", "middle")
			ctx.Call("fillText", "Loading...", CanvasWidth/2, CanvasHeight/2)
			js.Global().Call("requestAnimationFrame", loopFn)
			return nil
		}

		// Get the camera position and shake values.
		// This will be used to calculate the visible entities
		// and apply the camera transform.
		ox := -camX + camShakeX
		oy := -camY + camShakeY

		// Snap camera movement (including shake) to integer pixels.
		// This fixes the “pixel seams” from sub-pixel camera movement.
		tx := math.Round(ox)
		ty := math.Round(oy)
		ctx.Call("save")
		ctx.Call("translate", tx, ty)

		// Render the entities within the world-space.
		renderEntities(dt, false)

		// Undo the camera transform to display UI elements.
		ctx.Call("restore")

		// Render the UI elements.
		renderEntities(dt, true)
		renderUi()

		// Call the loop function recursively.
		js.Global().Call("requestAnimationFrame", loopFn)
		return nil
	})
	js.Global().Call("requestAnimationFrame", loopFn)
}

// SetRenderUi sets the function to render UI elements.
func SetRenderUi(fn func()) {
	renderUi = fn
}

// SetScreenSpace sets the entity at the given index to be in screen space or world space.
func SetScreenSpace(i int, screenSpace bool) {
	EntityRenderAsUi[i] = screenSpace
}

// SetSoundVolume sets the volume of a sound effect.
func SetSoundVolume(index int, volume float64) {
	sounds[index].Set("volume", volume)
}

// SetWorldSize sets the world size.
func SetWorldSize(width, height float64) {
	camMinX = 0
	camMinY = 0
	camMaxX = width
	camMaxY = height
	camBoundsSet = true
}

// StopSound stops a sound from the given index.
func StopSound(index int) {
	if sounds[index].Truthy() && !sounds[index].Get("paused").Bool() {
		sounds[index].Set("currentTime", 0)
		sounds[index].Call("pause")
	}
}

// addEventListeners adds event listeners for each event type.
func addEventListeners() {
	// Add event listeners for each event type.
	events := []string{"keydown", "keyup", "mousedown", "mousemove", "mouseup"}
	for _, e := range events {
		event := e
		target := js.Global()

		// Ensure that the mouse event listener is added to the canvas element.
		if event == "mousedown" || event == "mousemove" || event == "mouseup" {
			target = canvas
		}
		target.Call("addEventListener", event, js.FuncOf(func(this js.Value, args []js.Value) any {
			// Important: We cannot play audio until the user has interacted once.
			// Thus we use this flag later to determine if we can play audio.
			if (event == "keydown" || event == "mousedown") && !HasPlayerInput {
				HasPlayerInput = true
			}

			// Handle the event based on its type.
			switch event {
			case "keydown":
				key := args[0].Get("key").String()
				// Toggle fullscreen mode if the user presses the F key.
				if key == "f" || key == "F" {
					toggleFullscreen()
				}
				handleKeys(key, true)
			case "keyup":
				key := args[0].Get("key").String()
				handleKeys(key, false)
			case "mousedown":
				MouseDown = true
			case "mousemove":
				// Calculate the mouse position relative to the canvas.
				rect := canvas.Call("getBoundingClientRect")
				scaleX := float64(CanvasWidth) / rect.Get("width").Float()
				scaleY := float64(CanvasHeight) / rect.Get("height").Float()
				mx := (args[0].Get("clientX").Float() - rect.Get("left").Float()) * scaleX
				my := (args[0].Get("clientY").Float() - rect.Get("top").Float()) * scaleY

				// Calculate the mouse position relative to the world event with shake.
				MouseX = mx + camX - camShakeX
				MouseY = my + camY - camShakeY
			case "mouseup":
				MouseDown = false
			}

			// Prevent default behavior.
			if args != nil && args[0].Truthy() {
				args[0].Call("preventDefault")
			}
			return nil
		}))
	}
}

// handleKeys handles key events.
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

// isFullscreen checks if the canvas is in fullscreen mode.
func isFullscreen() bool {
	// Check if the fullscreen element is the canvas element.
	fullscreen := doc.Get("fullscreenElement")
	if fullscreen.Truthy() && fullscreen.Equal(canvas) {
		return true
	}
	webkitFullscreenElement := doc.Get("webkitFullscreenElement")
	if webkitFullscreenElement.Truthy() && webkitFullscreenElement.Equal(canvas) {
		return true
	}
	return false
}

// renderEntities renders the entities on the canvas.
// Sprites and tiles will be rendered in the world-space.
// UI entities will be rendered in the screen-space (if true).
func renderEntities(dt float64, screenSpace bool) {
	// Sort the entities based on their draw order (Painter's algorithm).
	sort.SliceStable(drawOrder, func(a, b int) bool {
		ai := drawOrder[a]
		bi := drawOrder[b]

		// Check for the lower Z layer first.
		if EntityZ[ai] != EntityZ[bi] {
			return EntityZ[ai] < EntityZ[bi]
		}

		// Check for painter's order by bottom edge (Y position and height).
		// The destination rectangle is centered around the entity's position.
		// Thus, the Y order is a bit tricky because we need to consider
		// the height of the entity and the position of the entity's center.
		ya := EntityY[ai] + EntitySpriteHeight[ai]/2
		yb := EntityY[bi] + EntitySpriteHeight[bi]/2
		if ya != yb {
			// Draw entities with lower Y coordinate first.
			return ya < yb
		}
		return ai < bi
	})

	// Calculate viewport dimensions.
	// - UI uses screen-space culling.
	// - World-space uses camera.
	vw, vh := float64(CanvasWidth), float64(CanvasHeight)
	var vLeft, vTop, vRight, vBottom float64
	if screenSpace {
		vLeft, vTop, vRight, vBottom = 0, 0, vw, vh
	} else {
		vLeft, vTop = camX-camShakeX, camY-camShakeY
		vRight, vBottom = vLeft+vw, vTop+vh
	}

	// Draw the entities with Z+Y sorting.
	alpha := 1.0
	for _, i := range drawOrder {
		// Skip UI elements or invisible entities.
		if EntityRenderAsUi[i] != screenSpace ||
			EntityState[i]&StateEntityVisible != StateEntityVisible {
			continue
		}
		img := images[EntityImageIndex[i]]

		// Skip entities without loaded images.
		if !img.Truthy() {
			continue
		}

		// Calculate the destination rectangle coordinates by using entity position and size.
		// The destination rectangle is centered around the entity's position.
		dstX := EntityX[i] - EntitySpriteWidth[i]/2
		dstY := EntityY[i] - EntitySpriteHeight[i]/2

		// Skip entities outside the viewport or which are explicitly invisible.
		if (dstX+EntitySpriteWidth[i] < vLeft || dstX > vRight || dstY+EntitySpriteHeight[i] < vTop || dstY > vBottom) ||
			EntityState[i]&StateEntityVisible != StateEntityVisible {
			continue
		}

		// Update the animation frame if sprite is animated.
		if EntityState[i]&StateEntityAnimated == StateEntityAnimated {
			EntityFrameTime[i] += dt

			// Check if the animation frame has reached the maximum duration.
			if EntityFrameTime[i] >= AnimationFrameDuration {
				EntityFrameTime[i] = 0
				EntityFrameOffset[i]++
			}

			// Check if the animation frame has reached the maximum number of frames.
			if EntityFrameOffset[i] >= AnimationFrameCount {
				EntityFrameOffset[i] = 0

				// Remove animation state (if not looping).
				if EntityState[i]&StateEntityAnimatedLoop != StateEntityAnimatedLoop {
					EntityState[i] &= ^StateEntityAnimated

					// Hide entity automatically after one-shot animation.
					if EntityState[i]&StateEntityAutoHide == StateEntityAutoHide {
						EntityState[i] &= ^StateEntityVisible
					}
				}
			}
		}

		// Calculate the source rectangle coordinates by using sprite position within the image
		// and the animation frame offset (no animation = offset 0).
		// Thus, we can use spritesheets and tilesets in production and do not need to split sprites
		// and tiles into multiple images.
		srcX := float64(EntityImageColumn[i])*EntitySpriteWidth[i] + float64(EntityFrameOffset[i])*EntitySpriteWidth[i]
		srcY := float64(EntityImageRow[i]) * EntitySpriteHeight[i]

		// Set the alpha value for the image if less than 1.
		if EntityAlpha[i] != alpha {
			ctx.Set("globalAlpha", EntityAlpha[i])
			alpha = EntityAlpha[i]
		}

		// Draw the image on the canvas (centered).
		ctx.Call("drawImage", img, srcX, srcY, EntitySpriteWidth[i], EntitySpriteHeight[i], dstX, dstY, EntitySpriteWidth[i], EntitySpriteHeight[i])
	}
	if alpha != 1.0 {
		ctx.Set("globalAlpha", 1.0)
	}
}

// toggleFullscreen toggles the fullscreen mode of the engine.
func toggleFullscreen() {
	// Skip if the last toggle was too recent.
	if lastToggleMs <= 500 {
		return
	}
	if isFullscreen() {
		// Exit fullscreen mode using the standard API.
		if doc.Get("exitFullscreen").Truthy() {
			doc.Call("exitFullscreen")
		} else if doc.Get("webkitExitFullscreen").Truthy() { // Safari
			doc.Call("webkitExitFullscreen")
		}
	} else {
		// Use the standard API if available.
		if canvas.Get("requestFullscreen").Truthy() {
			canvas.Call("requestFullscreen")
		} else if canvas.Get("webkitRequestFullscreen").Truthy() { // Safari
			canvas.Call("webkitRequestFullscreen")
		}
	}
	// Ensure to reset the last toggle time.
	lastToggleMs = 0
}

// updateCamera updates the camera position.
func updateCamera(dt float64) {
	// Updates the camera position.
	width, height := float64(CanvasWidth), float64(CanvasHeight)

	// Center the camera on the target if it exists.
	if CamTarget >= 0 {
		targetX := EntityX[CamTarget]
		targetY := EntityY[CamTarget]
		camX = targetX - width/2.0
		camY = targetY - height/2.0
	}

	// Ensure the camera position is within bounds.
	if camBoundsSet {
		worldW := camMaxX - camMinX
		worldH := camMaxY - camMinY

		// If the world is smaller than the view, center the world in the view.
		// Center the camera X.
		if worldW <= width {
			camX = camMinX + (worldW-width)/2.0
		} else {
			min := camMinX
			max := camMaxX - width
			if camX < min {
				camX = min
			}
			if camX > max {
				camX = max
			}
		}

		// Center the camera Y.
		if worldH <= height {
			camY = camMinY + (worldH-height)/2.0
		} else {
			min := camMinY
			max := camMaxY - height
			if camY < min {
				camY = min
			}
			if camY > max {
				camY = max
			}
		}
	}

	// Apply shake effect if active.
	if CamShakeTime > 0 {
		CamShakeTime -= dt

		// Center to [-1, 1], scale by magnitude.
		camShakeX = (rand.Float64()*2.0 - 1.0) * CamShakeMagnitude
		camShakeY = (rand.Float64()*2.0 - 1.0) * CamShakeMagnitude
	} else { // Or remove the shake effect if shake time is 0.
		camShakeX, camShakeY = 0, 0
		CamShakeMagnitude = 0
		CamShakeTime = 0
	}
}

// updateStates transitions every entity from one state to another.
func updateStates(dt float64) {
	// Handle input for the player entity (0).
	s := EntityState[0]

	// Detect if an "external action" is active (e.g. attack, dash, death, ...).
	// We treat any bit in RowIndexMask that is not a face/idle/move bit as external.
	key := s & RowIndexMask
	externalAction := key &^ (StateEntityFaceLeft | StateEntityFaceRight | StateEntityIdle | StateEntityMove)
	lockFacing := externalAction != 0

	// Always update movement flags from WASD.
	s &^= (StateEntityMoveDown | StateEntityMoveLeft | StateEntityMoveRight | StateEntityMoveUp)
	if KeyLeft {
		s |= StateEntityMoveLeft
		if !lockFacing {
			s |= StateEntityFaceLeft
			s &^= StateEntityFaceRight
		}
	}
	if KeyRight {
		s |= StateEntityMoveRight
		if !lockFacing {
			s |= StateEntityFaceRight
			s &^= StateEntityFaceLeft
		}
	}
	if KeyUp {
		s |= StateEntityMoveUp
	}
	if KeyDown {
		s |= StateEntityMoveDown
	}
	EntityState[0] = s

	// Update the state of each entity.
	for i, s := range EntityState {
		// Handle engine-known states.
		vx, vy := 0.0, 0.0
		if s&StateEntityMoveLeft != 0 {
			vx -= 1
		}
		if s&StateEntityMoveRight != 0 {
			vx += 1
		}
		if s&StateEntityMoveUp != 0 {
			vy -= 1
		}
		if s&StateEntityMoveDown != 0 {
			vy += 1
		}

		// Clear idle and set move state if moving.
		// Normalize the velocity and use the sprite speed factor.
		if n := vx*vx + vy*vy; n > 0 {
			inv := 1.0 / math.Sqrt(n)
			sf := EntitySpeedFactor[i]
			vx *= inv * sf
			vy *= inv * sf
			EntityX[i] += vx * EntitySpeed * dt
			EntityY[i] += vy * EntitySpeed * dt
			s &^= StateEntityIdle
			s |= StateEntityMove
		} else {
			// Clear move and set idle state if not moving.
			s &^= StateEntityMove
			s |= StateEntityIdle
		}

		// Switch spritesheet row if there is an external action
		// by using the provided lookup table.
		key := s & RowIndexMask
		externalAction := key &^ (StateEntityFaceLeft | StateEntityFaceRight | StateEntityIdle | StateEntityMove)
		if externalAction != 0 {
			key &^= (StateEntityMove | StateEntityIdle)
		}
		if row, ok := RowIndexForState[key]; ok && EntityImageRow[i] != row {
			EntityImageRow[i] = row
			EntityFrameOffset[i] = 0
			EntityFrameTime[i] = 0
		}

		// Save the new state.
		EntityState[i] = s
	}
}
