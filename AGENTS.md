# AGENTS.md - WiZ TUI (gowiz) Agent Guidelines

Welcome AI Coding Assistant / Agent! This repository contains `gowiz`, a terminal user interface (TUI) smart light control client for WiZ devices written in Go using `charmbracelet/bubbletea` and `charmbracelet/lipgloss`.

---

## Repository Overview

- **`main.go`**: Entry point for CLI flags (`--check`, `--ip`, `--mock`) and TUI initialization.
- **`internal/wiz/`**: Core WiZ smart light domain package:
  - `protocol.go`: JSON-RPC payload builders and response decoders for UDP port 38899.
  - `scenes.go`: Registry of all 32 dynamic WiZ scenes with search filtering.
  - `device.go`: Device data model and thread-safe `DeviceRegistry`.
  - `client.go` & `client_mock.go`: UDP network client interface and mock test harness.
  - `discovery.go`: Subnet UDP broadcast scanner.
- **`internal/tui/`**: Vim-like terminal user interface:
  - `model.go`: Bubble Tea root model handling modes (`NORMAL`, `VISUAL`, `COMMAND`, `SEARCH`, `HELP`).
  - `keybindings.go`: Modal definitions.
  - `styles/styles.go`: Catppuccin Macchiato Lipgloss theme palette.
  - `views/`: Modular layout components (`device_list.go`, `control_panel.go`, `scene_picker.go`, `status_bar.go`, `help_overlay.go`).
- **`scripts/verify.sh`**: 3-Tier Verification Pipeline.

---

## Core Rules & Quality Requirements

1. **Maintainability & Modular Architecture**:
   - Keep protocol logic inside `internal/wiz/` and UI views in `internal/tui/views/`.
   - Never place heavy protocol math or network calls directly in UI rendering functions.

2. **Tiered Verification**:
   - Before completing any task, you **MUST** run `./scripts/verify.sh` or `make verify` and ensure all 3 Tiers pass:
     - **Tier 1**: `gofmt`, `go vet`, unit tests.
     - **Tier 2**: `go test -race ./...` (race detector).
     - **Tier 3**: `go build` & headless execution `--check`.

3. **Vim Workflow Consistency**:
   - Maintain pure keyboard navigation (`hjkl`, `tab`, `:`, `/`, `v`, `?`, `u`).
   - Every user action should have a clear keyboard shortcut and status line indicator.
