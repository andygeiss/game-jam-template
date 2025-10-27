//go:build js && wasm

package engine

import "syscall/js"

const (
	// A canvas size of 640x360 is often used for pixel art games.
	// This resolution can be easily scaled up or down without losing quality.
	// We use some default constants here to make the implementation simpler.
	// This engine has a simple concept and does not compete with other engines.
	CanvasWidth  = 640
	CanvasHeight = 360
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
	states []uint64
	is     []int
	ws     []float64
	hs     []float64
	vxs    []float64
	vys    []float64
	xs     []float64
	ys     []float64
	zs     []float64
)

// AddEntity adds a new entity to the engine.
func AddEntity(state uint64, imgIndex int, w, h, vx, vy, x, y, z float64) {
	states = append(states, state)
	is = append(is, imgIndex)
	ws = append(ws, w)
	hs = append(hs, h)
	vxs = append(vxs, vx)
	vys = append(vys, vy)
	xs = append(xs, x)
	ys = append(ys, y)
	zs = append(zs, z)
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
		// Draw the sprites.
		for i := 0; i < len(hs); i++ {
			img := images[is[i]]
			if !img.Truthy() {
				continue
			}
			ctx.Call("drawImage", img, xs[i], ys[i], ws[i], hs[i])
		}
		// Call the loop function recursively.
		js.Global().Call("requestAnimationFrame", loopFn)
		return nil
	})
	js.Global().Call("requestAnimationFrame", loopFn)
}
