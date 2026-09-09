# Spec: Wisp Engine – Game Jam Template

The project-level brief. Every task brief is a delta against this file.

## Job

Someone with a jam deadline clones this repository and ships their own playable
browser game the same day.

## Why

Most jam templates hand over an empty window, so the first day goes to a render
loop, an input system and a WASM build instead of to the game.

## Guardrails

- The waived baseline rules and the conformance notes live in the README,
  *Baseline deviations*. Nothing is waived unless it is written there.
- The page's design decisions — colors, layout, surface style — live in
  `DESIGN.md`. The stylesheet and that file move together.
- The game keeps compiling with TinyGo. TinyGo is what makes the module small
  enough to load in a second, and it supports less of the standard library than
  Go does. `make wasm` is the check.
- `web/static/game.wasm` is committed, so `make run`, `make ci` and the
  container build work without TinyGo. Rebuild it in the same commit that
  changes `cmd/client` or the engine version in `go.mod`.
- The engine is a dependency, not a directory: `github.com/andygeiss/wisp-engine`.
  A fix that belongs to the engine goes there and comes back as a version
  bump, so every game built on it gets the fix rather than this one alone.
- It stays a template someone reads in an afternoon. A feature that costs the
  reader clarity belongs in a fork, not here.

## Done means

- The [web-application checklist](https://github.com/andygeiss/baseline/blob/main/checklists/web-application.md)
  is walked: every box checked, or waived in the README.
- `make ci` is green on the commit being pushed.
- The game still plays. `make wasm`, `make run`, then clear the arena and take
  the boss's ten lives at <http://127.0.0.1:8080/>.
