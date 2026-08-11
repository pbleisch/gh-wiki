# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

- Run `go build ./...`, `go vet ./...`, `go test ./...`, and `gofmt -l .` before delivery. `.github/workflows/ci.yml` enforces all four plus `go mod tidy` on Linux, macOS, and Windows. Git-backed tests use only temporary local repositories and never the network.
- `internal/wiki/manager.go` owns cache lifecycle and safe Git reconciliation, and `internal/wiki/state.go` enumerates which automatic action each divergence state permits. Never resolve divergence by discarding edits or force-pushing.
- **GitHub identifies a wiki page by its filename alone.** Directories in a wiki repository are rendered but ignored for naming, so `archive/Notes.md`, `notes.md`, and `Notes.md` are one page and only one is reachable. This drives `internal/wiki/list.go`, the slash rejection in `internal/wiki/titles.go`, and the flat collision check in `internal/wiki/pages.go`. Verify any further claim about GitHub's behavior against a real wiki before coding it, and record the evidence in a comment.
- Manager narration goes to `Err` and data to `Out`, so `list`, `path`, and `status` stay usable in a pipeline. Keep it that way when adding output.
- `.github/workflows/release.yml` is the authoritative release path: push a `vX.Y.Z` tag and `cli/gh-extension-precompile` builds the binaries `gh extension install` expects. It does **not** inject a version via ldflags; `cmd/version.go` recovers the version from Go's embedded build info instead, which yields the tag on a clean tagged checkout.
- The module path `github.com/pbleisch/gh-wiki` must match the repository. The Go module proxy records `module@version` immutably, so a wrong path cannot be cleanly corrected once a tag ships. At `v2.0.0` the path itself must gain a `/v2` suffix.
- See `README.md` for the command contract, cache layout, and researched extension conventions, and `CONTRIBUTING.md` for what is deliberately out of scope.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
