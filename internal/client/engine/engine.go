//go:build js && wasm

package engine

import (
	"math/rand/v2"
	"sort"
	"syscall/js"
)

const (
	// Wisp uses animations with 8 frames and a duration of 125 ms.
	// It provides a simple and efficient way to create animations.
	// It is a sweet spot between smoothness and performance.
	AnimationFrameCount    = 8
	AnimationFrameDuration = 125
	// A canvas size of 640x360 is often used for pixel art games.
	// This resolution can be easily scaled up or down without losing quality.
	// We use some default constants here to make the implementation simpler.
	// This engine has a simple concept and does not compete with other engines.
	CanvasWidth  = 640
	CanvasHeight = 360
)

const (
	// An entity has different states which determine its behavior.
	// The engine uses these states to provide basic functionality.
	// Only visible entities will be rendered.
	// Only animated entities will receive frame updates.
	StateEntityAnimated = uint64(1 << iota)
	StateEntityAnimatedLoop
	StateEntityVisible
)

var (
	// Control a 2D camera.
	camBoundsSet         bool
	camX, camY           float64
	camMinX, camMinY     float64
	camMaxX, camMaxY     float64
	camShakeX, camShakeY float64
	CamShakeMagnitude    float64
	CamShakeTime         float64
	CamTarget            int // -1 no target
	// canvas and ctx will be used to draw.
	// doc will be used to access the document.
	canvas js.Value
	ctx    js.Value
	doc    js.Value
	// Flags to control the engine's behavior.
	hasPlayerInput bool
	lastToggleMs   float64
	// Store the images.
	images       []js.Value
	imagesLoaded int
	// Key input related.
	Key1     bool
	Key2     bool
	Key3     bool
	Key4     bool
	KeyDown  bool
	KeyE     bool
	KeyLeft  bool
	KeyQ     bool
	KeyR     bool
	KeyRight bool
	KeyT     bool
	KeyUp    bool
	// The last timestamp of the animation frame.
	lastTs           float64
	HitStopRemaining float64
	// Ensure that the loop function is not garbage collected.
	loopFn js.Func
	// Mouse input related.
	MouseDown bool
	MouseX    float64
	MouseY    float64
	// Store the sounds.
	sounds       []js.Value
	soundsLoaded int
	// Store the entities.
	States []uint64
	As     []float64 // alpha (opacity)
	Ics    []int     // image col
	Irs    []int     // image row
	Iis    []int     // image index
	Ws     []float64 // sprite width
	Hs     []float64 // sprite height
	Xs     []float64
	Ys     []float64
	Zs     []int
	// Store the animations frames.
	Fos []int     // frame offsets.
	Fts []float64 // frame times.
	// draw order-related.
	drawOrder []int // order of the entities.
)

// AddEntity adds a new entity to the engine.
func AddEntity(state uint64, imgIndex, imgCol, imgRow int, w, h, x, y, alpha float64, z int) {
	States = append(States, state)
	As = append(As, alpha)
	Iis = append(Iis, imgIndex)
	Ics = append(Ics, imgCol)
	Irs = append(Irs, imgRow)
	Ws = append(Ws, w)
	Hs = append(Hs, h)
	Xs = append(Xs, x)
	Ys = append(Ys, y)
	Zs = append(Zs, z)
	Fos = append(Fos, 0)
	Fts = append(Fts, 0)
	// Add the current index to the draw order.
	drawOrder = append(drawOrder, len(States)-1)
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
	if sounds[index].Get("paused").Bool() {
		sounds[index].Set("loop", loop)
		sounds[index].Set("volume", volume)
		sounds[index].Call("play")
	}
}

// StopSound stops a sound from the given index.
func StopSound(index int) {
	if !sounds[index].Get("paused").Bool() {
		sounds[index].Set("currentTime", 0)
		sounds[index].Call("pause")
	}
}

// Run initializes the engine and starts the main loop.
// This function is called every frame with the delta time (in ms).
func Run(updateScene func(dt float64)) {
	// Create a few shortcuts.
	doc = js.Global().Get("document")
	perf := js.Global().Get("performance")
	// Initialize the canvas element first.
	canvas = doc.Call("createElement", "canvas")
	// Set the canvas element's width and height.
	canvas.Set("width", CanvasWidth)
	canvas.Set("height", CanvasHeight)
	// Add the canvas element to the document body.
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
		now := perf.Call("now").Float()
		dt := now - lastTs
		// Limit the maximum delta time to 50 milliseconds.
		if dt > 50 {
			dt = 50
		}
		lastTs = now
		// Record the time since the last toggle.
		lastToggleMs += dt
		// Check if hitstop is active and freeze the game.
		if HitStopRemaining > 0 {
			HitStopRemaining -= dt
			if HitStopRemaining <= 0 {
				HitStopRemaining = 0
			}
			// Freeze during hitstop.
			dt = 0
		}
		// Updates the data and handles the logic.
		updateScene(dt)
		// Update the camera position and shake even if hitstop is active.
		updateCamera(dt)
		// Clear the canvas.
		ctx.Call("clearRect", 0, 0, CanvasWidth, CanvasHeight)
		// Check if all assets are loaded.
		allAssetsLoaded := imagesLoaded == len(images) && soundsLoaded == len(sounds)
		// Skip rendering the entities if not all assets are loaded.
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
		ctx.Call("save")
		ctx.Call("translate", ox, oy)
		// Render the entities on the canvas.
		renderEntities(dt)
		// Undo the camera transform to display UI elements.
		ctx.Call("restore")
		// Show "Click to start the game" message if there is no player input.
		// We need a player input to play sound effects (security reason).
		if !hasPlayerInput {
			ctx.Set("fillStyle", "white")
			ctx.Set("font", "24px Arial")
			ctx.Set("textAlign", "center")
			ctx.Set("textBaseline", "middle")
			ctx.Call("fillText", "Click to start the game", CanvasWidth/2, CanvasHeight/2+80)
		}
		// Call the loop function recursively.
		js.Global().Call("requestAnimationFrame", loopFn)
		return nil
	})
	js.Global().Call("requestAnimationFrame", loopFn)
}

// SetWorldSize sets the world size.
func SetWorldSize(width, height float64) {
	camMinX = 0
	camMinY = 0
	camMaxX = width
	camMaxY = height
	camBoundsSet = true
}

// SetSoundVolume sets the volume of a sound effect.
func SetSoundVolume(index int, volume float64) {
	sounds[index].Set("volume", volume)
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
			if (event == "keydown" || event == "mousedown") && !hasPlayerInput {
				hasPlayerInput = true
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
func renderEntities(dt float64) {
	// Sort the entities based on their draw order (Painter's algorithm).
	sort.SliceStable(drawOrder, func(a, b int) bool {
		ai := drawOrder[a]
		bi := drawOrder[b]
		// Check for lower Z layer first.
		if Zs[ai] != Zs[bi] {
			return Zs[ai] < Zs[bi]
		}
		// Check for painter's order by bottom edge (Y position + height).
		// The destination rectangle is centered around the entity's position.
		// Thus the Y order is a bit tricky because we need to consider
		// the height of the entity and the position of the entity's center.
		ya := Ys[ai] + Hs[ai]/2
		yb := Ys[bi] + Hs[bi]/2
		if ya != yb {
			// Draw entities with lower Y coordinate first.
			return ya < yb
		}
		return ai < bi
	})
	// Calculate viewport dimensions.
	vw, vh := float64(CanvasWidth), float64(CanvasHeight)
	vLeft, vTop := camX-camShakeX, camY-camShakeY
	vRight, vBottom := vLeft+vw, vTop+vh
	// Draw the entities with Z+Y sorting.
	alpha := 1.0
	for _, i := range drawOrder {
		img := images[Iis[i]]
		// Skip entities without loaded images.
		if !img.Truthy() {
			continue
		}
		// Calculate the destination rectangle coordinates by using entity position and size.
		// The destination rectangle is centered around the entity's position.
		dstX := Xs[i] - Ws[i]/2
		dstY := Ys[i] - Hs[i]/2
		// Skip entities outside the viewport or which are explicitly invisible.
		if (dstX+Ws[i] < vLeft || dstX > vRight || dstY+Hs[i] < vTop || dstY > vBottom) ||
			States[i]&StateEntityVisible != StateEntityVisible {
			continue
		}
		// Update the animation frame if sprite is animated.
		if States[i]&StateEntityAnimated == StateEntityAnimated {
			Fts[i] += dt
			// Check if the animation frame has reached the maximum duration of 125 ms.
			if Fts[i] >= AnimationFrameDuration {
				Fts[i] = 0
				Fos[i]++
			}
			// Check if the animation frame has reached the maximum number of frames.
			if Fos[i] >= AnimationFrameCount {
				Fos[i] = 0
				// Remove animation state (if not looping).
				if States[i]&StateEntityAnimatedLoop != StateEntityAnimatedLoop {
					States[i] &= ^StateEntityAnimated
				}
			}
		}
		// Calculate the source rectangle coordinates by using sprite position within the image
		// and the animation frame offset (no animation = offset 0).
		// Thus we can use spritesheets and tilesets in production and do not need to split sprites
		// and tiles into multiple images.
		srcX := float64(Ics[i])*Ws[i] + float64(Fos[i])*Ws[i]
		srcY := float64(Irs[i]) * Hs[i]
		// Set the alpha value for the image if less than 1.
		if As[i] != alpha {
			ctx.Set("globalAlpha", As[i])
			alpha = As[i]
		}
		// Draw the image on the canvas (centered).
		ctx.Call("drawImage", img,
			srcX, srcY, Ws[i], Hs[i],
			dstX, dstY, Ws[i], Hs[i])
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
		targetX := Xs[CamTarget]
		targetY := Ys[CamTarget]
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
	} else { // Or remove shake effect if shake time is 0.
		camShakeX, camShakeY = 0, 0
		CamShakeMagnitude = 0
		CamShakeTime = 0
	}
}
