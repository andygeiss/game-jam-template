//go:build js && wasm

// The browser half of the engine: the canvas, images, sounds, input events,
// and the frame loop. Everything that touches syscall/js lives here.

package engine

import (
	"math"
	"syscall/js"
)

var (
	assetVersion     string
	assetVersionRead bool
	canvas           js.Value
	ctx              js.Value
	doc              js.Value
	images           []js.Value
	imagesLoaded     int
	lastToggleMs     float64
	lastTs           float64
	loopFn           js.Func
	renderUi         func()
	sounds           []js.Value
	soundsLoaded     int
)

// LoadImages starts loading each image. Rendering waits until every image
// and sound has arrived.
func LoadImages(paths ...string) {
	for _, path := range paths {
		img := js.Global().Get("Image").New()
		img.Set("onload", js.FuncOf(func(this js.Value, args []js.Value) any {
			imagesLoaded++
			return nil
		}))
		img.Set("src", assetURL(path))
		images = append(images, img)
	}
}

// LoadSounds starts loading each sound.
func LoadSounds(paths ...string) {
	for _, path := range paths {
		audio := js.Global().Get("Audio").New()
		audio.Set("oncanplaythrough", js.FuncOf(func(this js.Value, args []js.Value) any {
			soundsLoaded++
			return nil
		}))
		audio.Set("src", assetURL(path))
		sounds = append(sounds, audio)
	}
}

// PauseSound pauses sound index where it is. PlaySound picks it up from
// there, which is what a pause menu wants; StopSound rewinds instead.
func PauseSound(index int) {
	if sounds[index].Truthy() && !sounds[index].Get("paused").Bool() {
		sounds[index].Call("pause")
	}
}

// PlaySound plays sound index unless it is already playing.
func PlaySound(index int, volume float64, loop bool) {
	if sounds[index].Truthy() && sounds[index].Get("paused").Bool() {
		sounds[index].Set("loop", loop)
		sounds[index].Set("volume", volume)
		sounds[index].Call("play")
	}
}

// RenderRect fills a rectangle in screen space. Call it from the UI
// callback; a menu needs a backdrop, and the engine draws no shapes of its
// own.
func RenderRect(x, y, w, h float64, color string) {
	ctx.Set("fillStyle", color)
	ctx.Call("fillRect", x, y, w, h)
}

// RenderText draws text in screen space. Call it from the UI callback.
func RenderText(x, y float64, text, color, font, align string) {
	ctx.Set("fillStyle", color)
	ctx.Set("font", font)
	ctx.Set("textAlign", align)
	ctx.Set("textBaseline", "middle")
	ctx.Call("fillText", text, x, y)
}

// Run creates the canvas inside <main> and starts the frame loop. updateScene
// runs every frame with the elapsed milliseconds, before the engine moves
// entities and draws.
func Run(updateScene func(dt float64)) {
	doc = js.Global().Get("document")
	perf := js.Global().Get("performance")

	canvas = doc.Call("createElement", "canvas")
	canvas.Set("width", CanvasWidth)
	canvas.Set("height", CanvasHeight)
	mount := doc.Call("querySelector", "main")
	if !mount.Truthy() {
		mount = doc.Get("body")
	}
	mount.Call("appendChild", canvas)
	ctx = canvas.Call("getContext", "2d")

	lastTs = perf.Call("now").Float()
	lastToggleMs = lastTs

	addEventListeners()

	// loopFn is kept in a package variable so the garbage collector does not
	// free the callback while the browser still holds it.
	loopFn = js.FuncOf(func(this js.Value, args []js.Value) any {
		// Cap the step at 50 ms so a paused tab does not teleport everything.
		now := perf.Call("now").Float()
		dt := min(now-lastTs, 50)
		lastTs = now
		lastToggleMs += dt

		// Hit stop freezes the world for a few frames after a hit.
		if HitStopRemaining > 0 {
			HitStopRemaining = max(HitStopRemaining-dt, 0)
			dt = 0
		}

		// A pause holds the world where it is: no step, no camera, and no
		// animation, because renderEntities advances frames by dt too.
		if Paused {
			dt = 0
		}

		updateScene(dt)
		if !Paused {
			updateStates(dt)
			updateCamera(dt)
		}

		ctx.Call("clearRect", 0, 0, CanvasWidth, CanvasHeight)

		if imagesLoaded < len(images) || soundsLoaded < len(sounds) {
			RenderText(CanvasWidth/2, CanvasHeight/2, "Loading...", "white", "24px system-ui, sans-serif", "center")
			js.Global().Call("requestAnimationFrame", loopFn)
			return nil
		}

		sortDrawOrder()

		// The camera moves in whole pixels, shake included, or tile edges
		// show seams.
		ctx.Call("save")
		ctx.Call("translate", math.Round(-camX+camShakeX), math.Round(-camY+camShakeY))
		renderEntities(dt, false)
		ctx.Call("restore")

		renderEntities(dt, true)
		if renderUi != nil {
			renderUi()
		}

		js.Global().Call("requestAnimationFrame", loopFn)
		return nil
	})
	js.Global().Call("requestAnimationFrame", loopFn)
}

// SetRenderUi sets the callback that draws text and other screen-space
// content after the entities.
func SetRenderUi(fn func()) {
	renderUi = fn
}

// SetSoundVolume sets the volume of sound index.
func SetSoundVolume(index int, volume float64) {
	sounds[index].Set("volume", volume)
}

// StopSound stops sound index and rewinds it.
func StopSound(index int) {
	if sounds[index].Truthy() && !sounds[index].Get("paused").Bool() {
		sounds[index].Set("currentTime", 0)
		sounds[index].Call("pause")
	}
}

// addEventListeners wires keyboard events on the window and mouse events on
// the canvas.
func addEventListeners() {
	for _, e := range []string{"keydown", "keyup", "mousedown", "mousemove", "mouseup"} {
		event := e
		target := js.Global()
		if event == "mousedown" || event == "mousemove" || event == "mouseup" {
			target = canvas
		}
		target.Call("addEventListener", event, js.FuncOf(func(this js.Value, args []js.Value) any {
			// Browsers allow audio only after the first key or click.
			if event == "keydown" || event == "mousedown" {
				HasPlayerInput = true
			}

			switch event {
			case "keydown":
				key := args[0].Get("key").String()
				if key == "f" || key == "F" {
					toggleFullscreen()
				}
				handleKeys(key, true)
			case "keyup":
				handleKeys(args[0].Get("key").String(), false)
			case "mousedown":
				MouseDown = true
			case "mousemove":
				// The canvas is scaled by CSS, so map the pointer back to
				// canvas pixels, then into the world.
				rect := canvas.Call("getBoundingClientRect")
				scaleX := float64(CanvasWidth) / rect.Get("width").Float()
				scaleY := float64(CanvasHeight) / rect.Get("height").Float()
				mx := (args[0].Get("clientX").Float() - rect.Get("left").Float()) * scaleX
				my := (args[0].Get("clientY").Float() - rect.Get("top").Float()) * scaleY
				MouseX = mx + camX - camShakeX
				MouseY = my + camY - camShakeY
			case "mouseup":
				MouseDown = false
			}

			if len(args) > 0 && args[0].Truthy() {
				args[0].Call("preventDefault")
			}
			return nil
		}))
	}
}

// assetURL adds the build version to an asset path. Static assets are served
// with a one-year immutable cache, so the version in the URL is what makes a
// browser fetch a new spritesheet after a deploy. The page puts the version
// in data-version on <html>.
func assetURL(path string) string {
	if !assetVersionRead {
		assetVersionRead = true
		v := js.Global().Get("document").Get("documentElement").Get("dataset").Get("version")
		if v.Truthy() {
			assetVersion = v.String()
		}
	}
	if assetVersion == "" {
		return path
	}
	return path + "?v=" + assetVersion
}

// isFullscreen reports whether the canvas is the fullscreen element.
func isFullscreen() bool {
	for _, name := range []string{"fullscreenElement", "webkitFullscreenElement"} {
		el := doc.Get(name)
		if el.Truthy() && el.Equal(canvas) {
			return true
		}
	}
	return false
}

// renderEntities draws one pass: world space (screenSpace false) under the
// camera transform, or screen space (true) after it. Entities outside the
// view are skipped, and only drawn entities advance their animation.
func renderEntities(dt float64, screenSpace bool) {
	vw, vh := float64(CanvasWidth), float64(CanvasHeight)
	vLeft, vTop := 0.0, 0.0
	if !screenSpace {
		vLeft, vTop = camX-camShakeX, camY-camShakeY
	}
	vRight, vBottom := vLeft+vw, vTop+vh

	alpha := 1.0
	for _, i := range drawOrder {
		if EntityRenderAsUi[i] != screenSpace || EntityState[i]&StateEntityVisible == 0 {
			continue
		}
		img := images[EntityImageIndex[i]]
		if !img.Truthy() {
			continue
		}

		w, h := EntitySpriteWidth[i], EntitySpriteHeight[i]
		dstX := EntityX[i] - w/2
		dstY := EntityY[i] - h/2
		if dstX+w < vLeft || dstX > vRight || dstY+h < vTop || dstY > vBottom {
			continue
		}

		advanceAnimation(i, dt)

		// Column picks the sprite, frame offset picks the animation frame,
		// so one spritesheet holds every animation of every sprite.
		srcX := float64(EntityImageColumn[i]+EntityFrameOffset[i]) * w
		srcY := float64(EntityImageRow[i]) * h

		if EntityAlpha[i] != alpha {
			alpha = EntityAlpha[i]
			ctx.Set("globalAlpha", alpha)
		}
		ctx.Call("drawImage", img, srcX, srcY, w, h, dstX, dstY, w, h)
	}
	if alpha != 1.0 {
		ctx.Set("globalAlpha", 1.0)
	}
}

// toggleFullscreen enters or leaves fullscreen, at most twice a second.
func toggleFullscreen() {
	if lastToggleMs <= 500 {
		return
	}
	lastToggleMs = 0
	if isFullscreen() {
		if doc.Get("exitFullscreen").Truthy() {
			doc.Call("exitFullscreen")
		} else if doc.Get("webkitExitFullscreen").Truthy() {
			doc.Call("webkitExitFullscreen")
		}
		return
	}
	if canvas.Get("requestFullscreen").Truthy() {
		canvas.Call("requestFullscreen")
	} else if canvas.Get("webkitRequestFullscreen").Truthy() {
		canvas.Call("webkitRequestFullscreen")
	}
}
