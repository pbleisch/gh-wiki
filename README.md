# gh-wiki

`gh-wiki` is a GitHub CLI extension for editing a repository's GitHub wiki as ordinary local files. It keeps one managed Git clone per wiki, opens that clone in your preferred editor, and safely reconciles remote changes before pushing.

## Install

Install a released build with GitHub CLI:

```sh
gh extension install paul/gh-wiki
```

For local development:

```sh
go build -o gh-wiki .
gh extension install .
```

The repository and executable are both named `gh-wiki`, following GitHub CLI's extension convention. Releases are precompiled for supported platforms, so users do not need Go installed.

## Commands

Commands use the repository in the current directory. Pass `--repo OWNER/REPO` or `--repo HOST/OWNER/REPO` to select another repository.

```text
gh wiki sync                 Fetch and safely update the cached wiki
gh wiki edit                 Sync and open the entire wiki in an editor
gh wiki edit --code          Open the wiki in VS Code
gh wiki new "Release notes"  Create and edit Release-notes.md
gh wiki push -m "Add notes"  Commit, reconcile, and push local edits
gh wiki status               Show edits and local/remote divergence
gh wiki path                 Print the managed clone's path
```

Run `gh wiki --help` or `gh wiki COMMAND --help` for all options.

### Editing

`gh wiki edit` chooses the first configured editor from:

1. `--editor COMMAND`
2. `$VISUAL`
3. `$EDITOR`
4. `code --wait`, when VS Code's `code` command is available
5. `vi`

Use `--code` for an explicit `code --wait`. Editor values may include arguments, for example `--editor "code --reuse-window --wait"`.

### Syncing and pushing

`sync` fetches remote changes. It fast-forwards only when doing so cannot hide or overwrite local work. If the cache has edits or divergent commits, it explains what to do and leaves every local file untouched.

`push` stages and commits all wiki changes, fetches again, and rebases local wiki commits onto the current remote branch before pushing. If a conflict occurs, Git stops in the managed clone with all work preserved. Resolve the listed files, run the displayed `git -C ... rebase --continue` command, then run `gh wiki push` again. A commit message is requested through Git when `-m` is omitted.

## Cache

The cache uses the operating system's standard user cache directory:

- macOS: `~/Library/Caches/gh-wiki`
- Linux: usually `~/.cache/gh-wiki`
- Windows: the directory returned by the Windows user-cache API

Clones are keyed by host, owner, and repository, so identically named repositories do not collide. Set `GH_WIKI_CACHE_DIR` to override the root. `gh wiki path` prints the exact path for a repository.

To remove one cached wiki, remove the directory printed by `gh wiki path`. To remove every cached wiki, remove the `gh-wiki` cache root. Unpushed edits live in these directories, so check `gh wiki status` before deleting them.

## Expected errors

- If wikis are disabled, enable **Wikis** in the repository settings.
- GitHub does not create the separate `OWNER/REPO.wiki.git` repository until its first page exists. Create that first page on GitHub, then retry.
- Authentication comes from GitHub CLI. Check it with `gh auth status` and repair it with `gh auth login`.
- Run inside a GitHub repository or select one with `--repo`.
- Network errors do not remove an existing cache or its edits.

## Design and research notes

GitHub wikis are separate Git repositories at `https://HOST/OWNER/REPO.wiki.git`. The extension uses `github.com/cli/go-gh/v2` for repository resolution, API access, and authentication, while Git performs clone, fetch, rebase, and push. This keeps credentials in GitHub CLI instead of introducing another token store.

The command set focuses on the common shortcomings of the web editor: working across several pages, using a full editor, reviewing changes locally, and publishing a coherent commit. It intentionally does not add rendering, search, attachment management, page deletion, or a second history UI. Editors and existing Git tools already cover those needs, while duplicating them would make the extension harder to understand. `new` is included because GitHub's conversion between page titles and filenames is easy to get wrong by hand. `_Sidebar` and `_Footer` remain ordinary editable files but are recognized as GitHub's special layout files.

GitHub CLI discovers an extension from a `gh-NAME` repository containing a same-named executable and invokes it as `gh NAME`. `gh extension install` prefers release artifacts for binary extensions and falls back to cloning script extensions when no release exists. This project is a Go binary extension and uses GitHub's `cli/gh-extension-precompile` release action to produce the artifact names GitHub CLI expects.

Useful primary references:

- [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
- [`gh extension` manual](https://cli.github.com/manual/gh_extension)
- [`cli/go-gh`](https://github.com/cli/go-gh)
- [`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile)
- [GitHub wiki documentation](https://docs.github.com/en/communities/documenting-your-project-with-wikis)

## Development

```sh
go test ./...
go vet ./...
go build ./...
```
