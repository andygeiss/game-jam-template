<p align="center">
  <img src="docs/logo.png" alt="Wisp Engine – Game Jam Template" width="360" />
</p>

<h1 align="center">Wisp Engine – Game Jam Template</h1>

<p align="center">
  <strong>A complete, playable 2D arena game in Go + TinyGo + WebAssembly — clone it, reskin it, ship it before the jam ends.</strong>
</p>

<p align="center">
  <a href="https://go.dev/"><img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" /></a>
  <a href="https://tinygo.org/"><img alt="TinyGo" src="https://img.shields.io/badge/TinyGo-WASM-2E2E2E?logo=webassembly&logoColor=white" /></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg" /></a>
  <a href="https://github.com/andygeiss/game-jam-template/generate"><img alt="Use this template" src="https://img.shields.io/badge/Use%20this-template-006edb" /></a>
</p>

---

Most game jam templates hand you an empty window and wish you luck. This one hands you a **finished game**: a hero, ten monsters, a boss fight, three abilities with cooldowns, hearts, hit sounds, a music loop, camera shake, a game-over screen and a restart key — all rendering in the browser from a WASM binary measured in kilobytes.

Your job is to make it *yours*: swap the sprites, tweak the numbers, add a mechanic. The boring parts are done.

## ✨ What's in the box

| | |
|---|---|
| 🎮 **A complete game loop** | Idle → fight → boss → game over → restart. No placeholder screens. |
| 🗡️ **Three abilities** | Melee strike, projectile burst and a dash with invincibility frames — each with its own cooldown shown in the HUD. |
| 👹 **A boss encounter** | Spawns once the arena is cleared, has 10 lives, fires projectiles, and announces itself with a screen shake. |
| 🎨 **Pixel-art assets, sources included** | `.aseprite` files for the hero, boss, tileset and UI so you can edit instead of redraw. |
| 🔊 **Sound & music** | Attack and hit effects plus a looping soundtrack, wired up and ready to replace. |
| ⚡ **Tiny, fast WASM** | Built with TinyGo and squeezed with `wasm-opt`. Data-oriented, near-zero allocations, minimal JS↔WASM crossings. |
| 🖥️ **Production-ready server** | Go HTTP server with embedded assets, liveness/readiness probes, OIDC login, structured logging and a `FROM scratch` Docker image. |
| 📦 **One-command build** | `just run` compiles the client, copies the assets, builds the server and starts it. |

## 🕹️ Controls

Click the canvas to start. Clear all ten monsters to summon the boss.

| Key | Action |
|-----|--------|
| `W` `A` `S` `D` | Move |
| `Q` | Strike (1 s cooldown) |
| `E` | Projectile burst (3 s cooldown) |
| `R` | Dash — 4× speed, invincible (5 s cooldown) |
| `N` | Restart after game over |

## 🚀 Quick start

**Prerequisites:** [Go 1.25+](https://go.dev/dl/), [TinyGo](https://tinygo.org/getting-started/install/), [Binaryen](https://github.com/WebAssembly/binaryen) (`wasm-opt`) and [just](https://github.com/casey/just).

```sh
# 1. Create your own repo from this template, then clone it
git clone https://github.com/<you>/<your-game>.git
cd <your-game>

# 2. Install TinyGo and copy wasm_exec.js (macOS / Homebrew)
just setup

# 3. Configure the server (see below)
cp .env.example .env   # or create .env by hand

# 4. Build the WASM client + server and run it
just run
```

Then open <http://localhost:8080/game>.

### Server configuration

The server reads its settings from the environment (`.env` is loaded automatically by `just`):

| Variable | Purpose |
|----------|---------|
| `PORT` | HTTP port, e.g. `8080` |
| `OIDC_ISSUER` | OpenID Connect issuer URL |
| `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` | OIDC client credentials |
| `OIDC_REDIRECT_URL` | Callback URL registered with your provider (`http://localhost:8080/auth/callback`) |
| `REDIRECT_URL` | Where players land after login (`http://localhost:8080/game`) |

The `/game` route is protected by OIDC out of the box. If your jam doesn't need accounts, drop the `security.WithAuth(...)` wrapper in `internal/server/adapters/ingres/router.go`.

### All recipes

```sh
just build       # WASM client + server binaries (macOS and linux/amd64)
just run         # build, then start the server
just test        # run tests with coverage
just dockerize   # build and push a scratch-based container image
```

## 🧭 Project layout

```
assets/                 Sprites (.png + .aseprite), sounds and music
cmd/client/main.go      The game — entities, abilities, boss, HUD (compiled to WASM)
cmd/server/main.go      HTTP server that embeds and serves the game
cmd/server/assets/      index.html, styles.css, wasm_exec.js + copied assets
internal/client/engine/ The Wisp engine: game loop, entities, tilemaps, camera, input, audio
internal/server/        Config, routing and the HTML view handler
docs/                   Logo and documentation
Dockerfile              FROM scratch — the binary is the whole image
.justfile               Build recipes
```

## 🛠️ Make it yours

1. **Reskin** — open `assets/*.aseprite`, redraw, export to the `.png` next to it. Keep the frame layout and everything just works.
2. **Rebalance** — every cooldown, life count and speed lives in the constants at the top of `cmd/client/main.go`.
3. **New arena** — edit the tile indices in `enterScene()` or replace `tileset.png`.
4. **New mechanic** — add a state bit, a handler like `handleAction3`, and a HUD icon. The dash is a good example to copy.
5. **New enemy** — `addMonsters()` and `addBoss()` show how entities, animations and collision masks fit together.

## 💡 About Wisp

Wisp is a minimal 2D engine written in Go and built for the single-threaded WASM runtime. It doesn't try to compete with big engines — it's a lean, readable foundation for learning how render loops, input, entity systems and cameras actually work, and for shipping small games fast. If you can read Go, you can read the whole engine in an afternoon.

## 📄 License

[MIT](LICENSE) — use it for your jam, your course, or your next commercial pixel-art hit.
