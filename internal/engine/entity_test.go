package engine

import (
	"math"
	"testing"
)

// reset clears every package-level value a test could depend on. The engine
// is global state by design, so these tests do not run in parallel.
func reset(t *testing.T) {
	t.Helper()
	InitializeEntities()
	CamTarget, InputTarget = -1, -1
	RowIndexMask, RowIndexForState = 0, nil
	KeyLeft, KeyRight, KeyUp, KeyDown = false, false, false, false
	CamShakeTime, CamShakeMagnitude = 0, 0
	camX, camY = 0, 0
	SetWorldSize(CanvasWidth, CanvasHeight)
}

// entityCount returns the entity count and fails if any array disagrees.
func entityCount(t *testing.T) int {
	t.Helper()
	n := len(EntityState)
	for name, got := range map[string]int{
		"EntityAlpha":        len(EntityAlpha),
		"EntityFrameOffset":  len(EntityFrameOffset),
		"EntityFrameTime":    len(EntityFrameTime),
		"EntityImageColumn":  len(EntityImageColumn),
		"EntityImageIndex":   len(EntityImageIndex),
		"EntityImageRow":     len(EntityImageRow),
		"EntityRenderAsUi":   len(EntityRenderAsUi),
		"EntitySpeedFactor":  len(EntitySpeedFactor),
		"EntitySpriteHeight": len(EntitySpriteHeight),
		"EntitySpriteWidth":  len(EntitySpriteWidth),
		"EntityX":            len(EntityX),
		"EntityY":            len(EntityY),
		"EntityZ":            len(EntityZ),
		"drawOrder":          len(drawOrder),
	} {
		if got != n {
			t.Errorf("len(%s) = %d, want %d", name, got, n)
		}
	}
	return n
}

func TestAddEntity(t *testing.T) {
	reset(t)
	a := AddEntity(StateEntityVisible, 4, 1, 2, 32, 16, 100, 200, 0.5, 3)
	b := AddUI(StateEntityVisible, 0, 0, 0, 8, 8, 1, 1, 1, 0)

	if a != 0 || b != 1 {
		t.Fatalf("indices = %d, %d, want 0, 1", a, b)
	}
	if n := entityCount(t); n != 2 {
		t.Fatalf("entities = %d, want 2", n)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"image index", EntityImageIndex[a], 4},
		{"image column", EntityImageColumn[a], 1},
		{"image row", EntityImageRow[a], 2},
		{"width", EntitySpriteWidth[a], 32.0},
		{"height", EntitySpriteHeight[a], 16.0},
		{"x", EntityX[a], 100.0},
		{"y", EntityY[a], 200.0},
		{"alpha", EntityAlpha[a], 0.5},
		{"z", EntityZ[a], 3},
		{"speed factor", EntitySpeedFactor[a], 1.0},
		{"world space", EntityRenderAsUi[a], false},
		{"ui is screen space", EntityRenderAsUi[b], true},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestDeleteEntity(t *testing.T) {
	reset(t)
	// Z runs backwards so the sorted draw order is not the index order.
	for i, x := range []float64{10, 20, 30} {
		AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, x, 0, 1, 2-i)
	}
	sortDrawOrder()
	CamTarget, InputTarget = 2, 1

	DeleteEntity(1)

	if n := entityCount(t); n != 2 {
		t.Fatalf("entities = %d, want 2", n)
	}
	if EntityX[0] != 10 || EntityX[1] != 30 {
		t.Errorf("EntityX = %v, want [10 30]", EntityX)
	}
	if len(drawOrder) != 2 || drawOrder[0] != 1 || drawOrder[1] != 0 {
		t.Errorf("drawOrder = %v, want [1 0]", drawOrder)
	}
	for _, idx := range drawOrder {
		if idx < 0 || idx >= len(EntityState) {
			t.Fatalf("drawOrder holds %d, outside 0..%d", idx, len(EntityState)-1)
		}
	}
	if CamTarget != 1 {
		t.Errorf("CamTarget = %d, want 1 (shifted down)", CamTarget)
	}
	if InputTarget != -1 {
		t.Errorf("InputTarget = %d, want -1 (it was deleted)", InputTarget)
	}

	CamTarget = 0
	DeleteEntity(0)
	if CamTarget != -1 {
		t.Errorf("CamTarget = %d, want -1 after deleting its entity", CamTarget)
	}
	if n := entityCount(t); n != 1 || EntityX[0] != 30 {
		t.Errorf("entities = %d, EntityX = %v, want 1 entity at 30", n, EntityX)
	}
}

func TestBoundingBoxAndCollision(t *testing.T) {
	reset(t)
	a := AddEntity(0, 0, 0, 0, 32, 32, 100, 100, 1, 0)
	near := AddEntity(0, 0, 0, 0, 32, 32, 110, 100, 1, 0)
	far := AddEntity(0, 0, 0, 0, 32, 32, 200, 100, 1, 0)

	l, top, r, b := BoundingBox(a)
	if l != 96 || top != 96 || r != 116 || b != 116 {
		t.Errorf("BoundingBox = %v %v %v %v, want 96 96 116 116", l, top, r, b)
	}
	if !HasCollision(a, near) || !HasCollision(near, a) {
		t.Error("overlapping entities do not collide")
	}
	if HasCollision(a, far) {
		t.Error("entities 100 px apart collide")
	}
}

func TestAddTilemap(t *testing.T) {
	t.Run("skips empty and out-of-range tiles", func(t *testing.T) {
		reset(t)
		AddTilemap(7, []int{0, -1, 99, 4}, 2, 2, 3, 5, 32, 32)

		if n := entityCount(t); n != 2 {
			t.Fatalf("entities = %d, want 2", n)
		}
		if EntityX[0] != 16 || EntityY[0] != 16 || EntityImageColumn[0] != 0 || EntityImageRow[0] != 0 {
			t.Errorf("tile 0 at (%v,%v) col %d row %d, want (16,16) col 0 row 0", EntityX[0], EntityY[0], EntityImageColumn[0], EntityImageRow[0])
		}
		if EntityX[1] != 48 || EntityY[1] != 48 || EntityImageColumn[1] != 1 || EntityImageRow[1] != 1 {
			t.Errorf("tile 4 at (%v,%v) col %d row %d, want (48,48) col 1 row 1", EntityX[1], EntityY[1], EntityImageColumn[1], EntityImageRow[1])
		}
		if EntityImageIndex[0] != 7 || EntityZ[0] != 0 || EntityState[0] != StateEntityVisible {
			t.Errorf("tile image %d z %d state %b, want 7 0 visible", EntityImageIndex[0], EntityZ[0], EntityState[0])
		}
	})

	t.Run("a short array fills what it can", func(t *testing.T) {
		reset(t)
		AddTilemap(0, []int{1}, 3, 3, 3, 5, 32, 32)
		if n := entityCount(t); n != 1 {
			t.Errorf("entities = %d, want 1", n)
		}
	})
}

func TestUpdateCamera(t *testing.T) {
	tests := []struct {
		name           string
		worldW, worldH float64
		x, y           float64
		wantX, wantY   float64
	}{
		{"clamps at the top-left corner", 2000, 2000, 10, 10, 0, 0},
		{"clamps at the bottom-right corner", 2000, 2000, 1990, 1990, 2000 - CanvasWidth, 2000 - CanvasHeight},
		{"keeps the target centered inside", 2000, 2000, 1000, 1000, 1000 - CanvasWidth/2, 1000 - CanvasHeight/2},
		{"centers a world smaller than the view", 320, 180, 50, 50, -160, -90},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset(t)
			SetWorldSize(tt.worldW, tt.worldH)
			CamTarget = AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, tt.x, tt.y, 1, 0)

			updateCamera(16)

			if camX != tt.wantX || camY != tt.wantY {
				t.Errorf("camera = (%v, %v), want (%v, %v)", camX, camY, tt.wantX, tt.wantY)
			}
		})
	}

	t.Run("shake runs out", func(t *testing.T) {
		reset(t)
		CamShakeTime, CamShakeMagnitude = 100, 5

		updateCamera(16)
		if CamShakeTime != 84 || math.Abs(camShakeX) > 5 || math.Abs(camShakeY) > 5 {
			t.Errorf("after 16 ms: time %v shake (%v, %v), want 84 and |shake| <= 5", CamShakeTime, camShakeX, camShakeY)
		}

		// The frame that runs the timer out still shakes; the next one is still.
		updateCamera(100)
		if CamShakeTime > 0 || CamShakeMagnitude != 5 {
			t.Errorf("after 116 ms: time %v magnitude %v, want the timer run out and the magnitude kept for this frame", CamShakeTime, CamShakeMagnitude)
		}
		updateCamera(16)
		if CamShakeTime != 0 || CamShakeMagnitude != 0 || camShakeX != 0 || camShakeY != 0 {
			t.Errorf("after 132 ms: time %v magnitude %v shake (%v, %v), want all 0", CamShakeTime, CamShakeMagnitude, camShakeX, camShakeY)
		}
	})

	t.Run("a target past the end is ignored", func(t *testing.T) {
		reset(t)
		CamTarget = 5
		updateCamera(16)
	})
}

func TestUpdateStates(t *testing.T) {
	const attack = uint64(1 << 20)
	const step = EntitySpeed * 100 // one 100 ms frame at speed factor 1

	t.Run("no entities and no target does not panic", func(t *testing.T) {
		reset(t)
		updateStates(16)
		InputTarget = 0
		updateStates(16)
	})

	t.Run("WASD moves the target and picks the row", func(t *testing.T) {
		reset(t)
		RowIndexMask = poseMask
		RowIndexForState = map[uint64]int{
			StateEntityFaceRight | StateEntityIdle: 0,
			StateEntityFaceLeft | StateEntityIdle:  1,
			StateEntityFaceRight | StateEntityMove: 2,
			StateEntityFaceLeft | StateEntityMove:  3,
		}
		p := AddEntity(StateEntityFaceRight|StateEntityIdle|StateEntityVisible, 0, 0, 0, 32, 32, 100, 100, 1, 0)
		InputTarget = p
		EntityFrameOffset[p] = 5

		KeyLeft = true
		updateStates(100)

		if EntityX[p] != 100-step {
			t.Errorf("x = %v, want %v", EntityX[p], 100-step)
		}
		s := EntityState[p]
		if s&StateEntityMoveLeft == 0 || s&StateEntityFaceLeft == 0 || s&StateEntityMove == 0 {
			t.Errorf("state %b lacks MoveLeft, FaceLeft, or Move", s)
		}
		if s&StateEntityFaceRight != 0 || s&StateEntityIdle != 0 {
			t.Errorf("state %b still has FaceRight or Idle", s)
		}
		if EntityImageRow[p] != 3 || EntityFrameOffset[p] != 0 {
			t.Errorf("row %d frame %d, want row 3 and the animation restarted", EntityImageRow[p], EntityFrameOffset[p])
		}

		KeyLeft = false
		updateStates(100)

		if EntityX[p] != 100-step {
			t.Errorf("x = %v after release, want %v", EntityX[p], 100-step)
		}
		s = EntityState[p]
		if s&StateEntityIdle == 0 || s&StateEntityMove != 0 || s&StateEntityFaceLeft == 0 {
			t.Errorf("state %b, want Idle and FaceLeft without Move", s)
		}
		if EntityImageRow[p] != 1 {
			t.Errorf("row = %d, want 1", EntityImageRow[p])
		}
	})

	t.Run("a diagonal is as fast as a straight line", func(t *testing.T) {
		reset(t)
		p := AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, 100, 100, 1, 0)
		InputTarget = p
		KeyLeft, KeyUp = true, true

		updateStates(100)

		want := step / math.Sqrt2
		if math.Abs(EntityX[p]-(100-want)) > 1e-9 || math.Abs(EntityY[p]-(100-want)) > 1e-9 {
			t.Errorf("moved to (%v, %v), want (%v, %v)", EntityX[p], EntityY[p], 100-want, 100-want)
		}
	})

	t.Run("the speed factor scales the step", func(t *testing.T) {
		reset(t)
		p := AddEntity(StateEntityVisible|StateEntityMoveRight, 0, 0, 0, 32, 32, 100, 100, 1, 0)
		EntitySpeedFactor[p] = 2

		updateStates(100)

		if EntityX[p] != 100+2*step {
			t.Errorf("x = %v, want %v", EntityX[p], 100+2*step)
		}
	})

	t.Run("an action locks the facing and picks its own row", func(t *testing.T) {
		reset(t)
		RowIndexMask = poseMask | attack
		RowIndexForState = map[uint64]int{attack | StateEntityFaceRight: 9}
		p := AddEntity(StateEntityFaceRight|attack|StateEntityVisible, 0, 0, 0, 32, 32, 100, 100, 1, 0)
		InputTarget = p
		KeyLeft = true

		updateStates(100)

		s := EntityState[p]
		if s&StateEntityFaceRight == 0 || s&StateEntityFaceLeft != 0 {
			t.Errorf("state %b turned around during an action", s)
		}
		if s&StateEntityMoveLeft == 0 {
			t.Errorf("state %b did not move", s)
		}
		if EntityImageRow[p] != 9 {
			t.Errorf("row = %d, want 9 (the action row, idle/move dropped from the key)", EntityImageRow[p])
		}
	})

	t.Run("static entities are left alone", func(t *testing.T) {
		reset(t)
		RowIndexMask = poseMask
		tile := AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, 16, 16, 1, 0)

		updateStates(100)

		if EntityState[tile] != StateEntityVisible || EntityX[tile] != 16 {
			t.Errorf("tile state %b at x %v, want untouched", EntityState[tile], EntityX[tile])
		}
	})

	t.Run("move bits move an entity that is not the input target", func(t *testing.T) {
		reset(t)
		p := AddEntity(StateEntityVisible|StateEntityMoveRight, 0, 0, 0, 32, 32, 100, 100, 1, 0)

		updateStates(100)

		if EntityX[p] != 100+step {
			t.Errorf("x = %v, want %v", EntityX[p], 100+step)
		}
	})
}

func TestSortDrawOrder(t *testing.T) {
	reset(t)
	AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, 0, 10, 1, 1)  // 0: z 1
	AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, 0, 100, 1, 0) // 1: z 0, low on screen
	AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, 0, 50, 1, 0)  // 2: z 0, higher up
	AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, 0, 50, 1, 0)  // 3: ties with 2

	sortDrawOrder()

	want := []int{2, 3, 1, 0}
	for i := range want {
		if drawOrder[i] != want[i] {
			t.Fatalf("drawOrder = %v, want %v", drawOrder, want)
		}
	}
}

func TestAdvanceAnimation(t *testing.T) {
	t.Run("a one-shot animation hides its entity at the end", func(t *testing.T) {
		reset(t)
		e := AddEntity(StateEntityAnimated|StateEntityAutoHide|StateEntityVisible, 0, 0, 0, 32, 32, 0, 0, 1, 0)

		for range AnimationFrameCount - 1 {
			advanceAnimation(e, AnimationFrameDuration)
		}
		if EntityFrameOffset[e] != AnimationFrameCount-1 || EntityState[e]&StateEntityVisible == 0 {
			t.Fatalf("before the last frame: offset %d state %b, want offset %d and visible", EntityFrameOffset[e], EntityState[e], AnimationFrameCount-1)
		}

		advanceAnimation(e, AnimationFrameDuration)
		if EntityFrameOffset[e] != 0 || EntityState[e]&(StateEntityAnimated|StateEntityVisible) != 0 {
			t.Errorf("after the last frame: offset %d state %b, want offset 0, not animated, hidden", EntityFrameOffset[e], EntityState[e])
		}
	})

	t.Run("a looping animation starts over", func(t *testing.T) {
		reset(t)
		e := AddEntity(StateEntityAnimated|StateEntityAnimatedLoop|StateEntityVisible, 0, 0, 0, 32, 32, 0, 0, 1, 0)

		for range AnimationFrameCount {
			advanceAnimation(e, AnimationFrameDuration)
		}
		if EntityFrameOffset[e] != 0 || EntityState[e]&StateEntityAnimated == 0 || EntityState[e]&StateEntityVisible == 0 {
			t.Errorf("offset %d state %b, want offset 0, animated, visible", EntityFrameOffset[e], EntityState[e])
		}
	})

	t.Run("frame time accumulates across frames", func(t *testing.T) {
		reset(t)
		e := AddEntity(StateEntityAnimated, 0, 0, 0, 32, 32, 0, 0, 1, 0)

		advanceAnimation(e, 60)
		if EntityFrameOffset[e] != 0 || EntityFrameTime[e] != 60 {
			t.Errorf("after 60 ms: offset %d time %v, want 0 and 60", EntityFrameOffset[e], EntityFrameTime[e])
		}
		advanceAnimation(e, 60)
		if EntityFrameOffset[e] != 1 || EntityFrameTime[e] != 0 {
			t.Errorf("after 120 ms: offset %d time %v, want 1 and 0", EntityFrameOffset[e], EntityFrameTime[e])
		}
	})

	t.Run("a still entity stays on its frame", func(t *testing.T) {
		reset(t)
		e := AddEntity(StateEntityVisible, 0, 0, 0, 32, 32, 0, 0, 1, 0)
		advanceAnimation(e, 500)
		if EntityFrameOffset[e] != 0 || EntityFrameTime[e] != 0 {
			t.Errorf("offset %d time %v, want both 0", EntityFrameOffset[e], EntityFrameTime[e])
		}
	})
}

func TestHandleKeys(t *testing.T) {
	keys := []struct {
		key  string
		flag *bool
	}{
		{"a", &KeyLeft}, {"D", &KeyRight}, {"w", &KeyUp}, {"S", &KeyDown},
		{"q", &KeyQ}, {"E", &KeyE}, {"r", &KeyR}, {"T", &KeyT}, {"n", &KeyN},
		{"1", &Key1}, {"2", &Key2}, {"3", &Key3}, {"4", &Key4},
	}
	for _, k := range keys {
		t.Run(k.key, func(t *testing.T) {
			handleKeys(k.key, true)
			if !*k.flag {
				t.Errorf("pressing %q did not set its flag", k.key)
			}
			handleKeys(k.key, false)
			if *k.flag {
				t.Errorf("releasing %q did not clear its flag", k.key)
			}
		})
	}

	t.Run("an unknown key changes nothing", func(t *testing.T) {
		handleKeys("z", true)
		for _, k := range keys {
			if *k.flag {
				t.Errorf("%q was set by an unknown key", k.key)
			}
		}
	})
}
