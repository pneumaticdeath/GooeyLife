# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

GooeyLife is a GUI frontend for Conway's Game of Life, built with the [Fyne](https://fyne.io/) toolkit. It uses [github.com/pneumaticdeath/golife](https://github.com/pneumaticdeath/golife) as the underlying simulation engine. The app targets desktop (macOS, Windows, Linux), mobile (iOS, Android), and web.

## Build & Run Commands

```bash
# Run locally
go run .

# Build for current platform
go build .

# Cross-compile for all platforms (requires fyne-cross)
./build.sh

# Release build for desktop (macOS) — requires .env with cert/profile vars
./release_desktop.sh

# Release build for mobile (iOS + Android) — requires .env with cert/keystore vars
./release_mobile.sh
```

The `.env` file (not committed) must export: `ANDROIDKEYSTORE`, `ANDROIDKEYNAME`, `ANDROIDKEYPASS`, `IOSDISTROCERT`, `IOSDISTROPROF`, `MACOSDEVCERT`, `MACOSDEVPROF`.

Cross-compilation uses `fyne-cross`. The `build.sh` script also patches `FyneApp.toml` to add a `[LinuxAndBSD]` block (from `linux-block.toml`) needed to work around a bug in the Android cross-compile container.

## App Metadata

App ID: `io.patenaude.gooeylife`
Version and build number are in `FyneApp.toml`.

## Architecture

The app is a single `main` package. Each file has a focused responsibility:

| File | Purpose |
|------|---------|
| `main.go` | App entry point, menus, keyboard shortcuts, drag-and-drop |
| `container.go` | `LifeContainer` — top-level widget per tab, composes Sim + ControlBar + StatusBar |
| `sim.go` | `LifeSim` — the Fyne widget that renders cells on a canvas and handles user interaction (tap to edit, drag to pan, scroll to zoom/pan) |
| `control.go` | `ControlBar` — play/pause/step buttons, zoom, speed slider; drives `LifeSimClock` |
| `clock.go` | `LifeSimClock` (per-tab, advances generations) and `DisplayUpdateClock` (singleton, drives screen redraws at configurable Hz) |
| `status.go` | `StatusBar` — displays generation, cell count, step/draw timing, GPS stats |
| `tabs.go` | `LifeTabs` — wraps Fyne `DocTabs`, manages multiple simultaneous simulations |
| `preferences.go` | `ConfigT` global — wraps `fyne.Preferences` for all settings (colors, history size, refresh rate, saved games) |
| `tour.go` | Guided tour shown on first launch |
| `file_ext.go` | `LongExtensionsFileFilter` — file filter supporting multi-part extensions like `.rle.txt` |

### Key Data Flow

```
golife.Game (engine) ← LifeSim (renders) ← LifeContainer (layout)
                                                    ↑
                              ControlBar ──── LifeSimClock (steps generations)
                              StatusBar  ← DisplayUpdateClock (redraws at ~30/60 Hz)
```

- **`LifeSimClock`** runs a goroutine per tab that blocks on a channel; `ControlBar.StepForward()` sends a tick, the goroutine calls `golife.Game.Next()` and sets `LifeSim.Dirty = true`.
- **`DisplayUpdateClock`** is a singleton goroutine that calls `LifeSim.Draw()` on the active tab at the configured refresh rate. `Draw()` only repaints when `Dirty` is true.
- **`fyne.Do`/`fyne.DoAndWait`** are used wherever UI state must be mutated from a background goroutine (required by Fyne v2.4+).

### Concurrency Notes

`LifeSim.Draw()` holds `drawLock` (sync.Mutex) to prevent concurrent draws. Simulation stepping and display are on separate goroutines; `Dirty` is the handoff flag. When a tab is closed, `LifeContainer.StopClocks()` shuts down all associated goroutines before the tab is removed.

### Supported File Formats

Import/export via the `golife` library: `.rle`, `.life`, `.cells` (and `.txt` variants). Export is always RLE. Drag-and-drop of pattern files is supported on desktop.

### Preferences Storage

`ConfigT` wraps `fyne.App.Preferences()`. Saved games are stored as a string list of RLE-encoded patterns. Colors are stored as `[]int` with RGBA components.
