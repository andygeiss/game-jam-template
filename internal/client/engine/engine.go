//go:build js && wasm

package engine

import (
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
	StateEntityAlive = (1 << iota)
	StateEntityAnimated
	StateEntityVisible
)

var (
	// The canvas element and its context will be used to draw.
	canvas js.Value
	ctx    js.Value
	// The number of entities.
	Entities int
	// Store the images.
	images []js.Value
	// The last timestamp of the animation frame.
	lastTs float64
	// Ensure that the loop function is not garbage collected.
	loopFn js.Func
	// Store the entities.
	states []uint64
	as     []float64 // alpha (opacity)
	ics    []int     // image col
	irs    []int     // image row
	iis    []int     // image index
	ws     []float64 // sprite width
	hs     []float64 // sprite height
	xs     []float64
	ys     []float64
	zs     []float64
	// Store the animations frames.
	fcs []float64 // frame counts.
	fos []float64 // frame offsets.
)

// AddEntity adds a new entity to the engine.
func AddEntity(state uint64, imgIndex, imgCol, imgRow int, w, h, x, y, z, alpha float64) {
	states = append(states, state)
	as = append(as, alpha)
	iis = append(iis, imgIndex)
	ics = append(ics, imgCol)
	irs = append(irs, imgRow)
	ws = append(ws, w)
	hs = append(hs, h)
	xs = append(xs, x)
	ys = append(ys, y)
	zs = append(zs, z)
	fcs = append(fcs, 1)
	fos = append(fos, 0)
	Entities++
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
		// Draw the entities.
		for i := range Entities {
			img := images[iis[i]]
			if !img.Truthy() {
				continue
			}
			// Calculate the source rectangle coordinates by using sprite position within the image
			// and the animation frame offset (no animation = offset 0).
			// Thus we can use spritesheets and tilesets in production and do not need to split sprites
			// and tiles into multiple images.
			srcX := float64(ics[i])*ws[i] + float64(fos[i])*ws[i]
			srcY := float64(irs[i]) * hs[i]
			// The destination rectangle coordinates are calculated by subtracting half of the width and height
			// from the entity's position to center the image on the entity.
			// Thus we use the entity's position as the center point for the image.
			dstX := xs[i] - ws[i]/2
			dstY := ys[i] - hs[i]/2
			// Set the alpha value for the image if less than 1.
			if as[i] < 1 {
				ctx.Set("globalAlpha", as[i])
			}
			// Draw the image on the canvas (centered).
			ctx.Call("drawImage", img,
				srcX, srcY, ws[i], hs[i],
				dstX, dstY, ws[i], hs[i])
		}
		// Call the loop function recursively.
		js.Global().Call("requestAnimationFrame", loopFn)
		return nil
	})
	js.Global().Call("requestAnimationFrame", loopFn)
}
