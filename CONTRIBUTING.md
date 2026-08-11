# Contributing

Thanks for looking. This is a small, deliberately narrow tool maintained by one
person, so the most useful contributions are small ones: a bug you can
reproduce, a rough edge in the help text, a platform where something behaves
differently than the README claims.

Please be decent to whoever else turns up here. Reports and reviews get read;
rudeness gets the thread locked.

## Reporting a bug

Open an issue and include:

- the command you ran and what you expected instead,
- the output of `gh wiki status` for the repository involved, which reports the
  cache path and how far it has diverged from the remote,
- `gh wiki --version`, `gh --version`, `git --version`, and your OS.

If a suspected vulnerability is involved, do not open an issue. Follow
[SECURITY.md](SECURITY.md) instead.

## Before opening a pull request

For anything larger than a fix, open an issue first. The scope of this tool is
intentionally small — it manages a local clone of a wiki and nothing else. The
"Design and research notes" section of the [README](README.md) lists what has
been left out on purpose (rendering, search, page deletion, a second history
UI); a pull request adding one of those is likely to be declined however good
the code is, and it is kinder to say so before you write it.

## Working on the code

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

All four must be clean; `gofmt -l .` must print nothing. Then install your build
as the real extension to try it:

```sh
go build -o gh-wiki . && gh extension install .
```

Notes that save time:

- The tests build real Git repositories in temporary directories. They never
  touch the network, your GitHub account, or your own wiki caches. Keep it that
  way — set `GH_WIKI_CACHE_DIR` if you need a cache in a test.
- Nothing may resolve a divergence between the cache and the remote by
  discarding edits or force-pushing. `internal/wiki/state.go` enumerates the
  states and which automatic action each one permits; if a change needs a new
  action, it belongs there with a test.
- Claims about how GitHub itself behaves — page naming, what a wiki URL
  resolves to — should be verified against a real wiki and recorded in a comment
  next to the code, not assumed.
- Commands print data on standard output and everything else on standard error,
  so `gh wiki list`, `path`, and `status` stay usable in a pipeline.

## Releasing

Maintainer only: push a `vX.Y.Z` tag. `.github/workflows/release.yml` builds the
precompiled binaries that `gh extension install` expects and attaches them to
the release.
