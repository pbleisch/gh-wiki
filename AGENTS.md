# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

- Run `go test ./...`, `go vet ./...`, and `go build ./...` before delivery. Git-backed integration tests use only temporary local repositories.
- `internal/wiki/manager.go` owns cache lifecycle and safe Git reconciliation. Never resolve divergence by discarding edits or force-pushing.
- `.github/workflows/release.yml` is the authoritative precompiled GitHub CLI extension release path.
- See `README.md` for the command contract, cache layout, and researched extension conventions.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
