//go:build js && wasm

package engine

import (
	"sort"
	"syscall/js"
)

const (
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
	// Only alive entities can be updated and rendered.
	// Only visible entities will be rendered.
	// Only animated entities will receive frame updates.
	StateEntityAlive = uint64(1 << iota)
	StateEntityAnimated
	StateEntityAnimatedLoop
	StateEntityVisible
)

var (
	// The canvas element and its context will be used to draw.
	canvas js.Value
	ctx    js.Value
	// Store the images.
	images []js.Value
	// The last timestamp of the animation frame.
	lastTs float64
	// Ensure that the loop function is not garbage collected.
	loopFn js.Func
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
	Fcs []int     // frame counts.
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
	Fcs = append(Fcs, 1)
	Fos = append(Fos, 0)
	Fts = append(Fts, 0)
	// Add the current index to the draw order.
	drawOrder = append(drawOrder, len(States)-1)
}

// LoadImages loads an image from the given path.
func LoadImages(paths ...string) {
	for _, path := range paths {
		// Check if the image is already stored.
		for _, img := range images {
			if img.Get("src").String() == path {
				return
			}
		}
		// Create a new image element.
		val := js.Global().Get("Image").New()
		val.Set("src", "/assets/"+path)
		images = append(images, val)
	}
}

// Run initializes the engine and starts the main loop.
func Run(update func(dt float64)) {
	// Create a few shortcuts.
	doc := js.Global().Get("document")
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
	// Ensure that the loop function is not garbage collected.
	loopFn = js.FuncOf(func(this js.Value, args []js.Value) any {
		now := perf.Call("now").Float()
		dt := now - lastTs
		// Limit the maximum delta time to 50 milliseconds.
		if dt > 50 {
			dt = 50
		}
		lastTs = now
		// Updates the data and handles the logic.
		update(dt)
		// Clear the canvas.
		ctx.Call("clearRect", 0, 0, CanvasWidth, CanvasHeight)
		// Sort the entities based on their draw order.
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
		// Draw the entities (with 4 layers).
		alphaResetNeeded := false
		for _, i := range drawOrder {
			img := images[Iis[i]]
			// Skip entities without loaded images.
			if !img.Truthy() {
				continue
			}
			// Update the animation frame if sprite is animated.
			if States[i]&StateEntityAnimated == StateEntityAnimated {
				Fts[i] += dt
				// Check if the animation frame has reached the maximum duration of 150 ms.
				if Fts[i] >= 150 {
					Fts[i] = 0
					Fos[i]++
				}
				// Check if the animation frame has reached the maximum number of frames.
				if Fos[i] >= Fcs[i] {
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
			// Calculate the destination rectangle coordinates by using entity position and size.
			// The destination rectangle is centered around the entity's position.
			dstX := Xs[i] - Ws[i]/2
			dstY := Ys[i] - Hs[i]/2
			// Set the alpha value for the image if less than 1.
			if As[i] < 1 {
				ctx.Set("globalAlpha", As[i])
				alphaResetNeeded = true
			}
			// Draw the image on the canvas (centered).
			ctx.Call("drawImage", img,
				srcX, srcY, Ws[i], Hs[i],
				dstX, dstY, Ws[i], Hs[i])
			// Reset the alpha value if needed.
			if alphaResetNeeded {
				ctx.Set("globalAlpha", 1)
				alphaResetNeeded = false
			}
		}
		// Call the loop function recursively.
		js.Global().Call("requestAnimationFrame", loopFn)
		return nil
	})
	js.Global().Call("requestAnimationFrame", loopFn)
}
