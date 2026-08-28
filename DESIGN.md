---
version: alpha
name: Wisp Engine – Game Jam Template
description: One page that hosts a pixel-art arena game rendered by WebAssembly on a canvas.
colors:
  bg: "oklch(99% 0.002 260)"
  text: "oklch(22% 0.02 260)"
  canvas: "oklch(18% 0.01 260)"
  bg-dark: "oklch(18% 0.01 260)"
  text-dark: "oklch(93% 0.01 260)"
typography:
  body:
    fontFamily: "system-ui, sans-serif"
components:
  canvas:
    backgroundColor: "{colors.canvas}"
---

# Design: Wisp Engine – Game Jam Template

## Overview

The page is a frame around one canvas. The game draws everything inside the
canvas; the page only centers it, scales it, and follows the reader's color
scheme. Surface style: minimal.

## Colors

Two page roles, light and dark, plus one fixed canvas ground. Every value is
the same string as in `web/static/css/app.css`.

| Role | Light | Dark | Contrast |
|---|---|---|---|
| `--color-bg` | `oklch(99% 0.002 260)` | `oklch(18% 0.01 260)` | — |
| `--color-text` | `oklch(22% 0.02 260)` | `oklch(93% 0.01 260)` | 16.6:1 light, 14.9:1 dark |
| `--color-canvas` | `oklch(18% 0.01 260)` | same | white canvas text 15.6:1 |

## Typography

`system-ui, sans-serif` at the browser's default size. The page shows no
visible text; the `<h1>` is for screen readers.

## Layout

`main` is a grid that centers the canvas and pads it by `--space`
(`clamp(1rem, 0.5rem + 2vw, 2rem)`). The canvas is `min(100%, 640px)` wide
with a 16:9 aspect ratio, so it fits a 320 px phone and stays sharp on a
desktop.

## Elevation & Depth

None. One flat surface.

## Shapes

Square corners. Pixel art has no radius.

## Components

- **canvas** — the game. Dark ground, pixelated scaling, fullscreen on `F`.

## Do's and Don'ts

- Do keep the canvas at a whole-number scale where you can; pixel art blurs
  at fractional scales.
- Don't put visible UI on the page. The HUD is drawn by the game, so it is
  inside the canvas in fullscreen too.
