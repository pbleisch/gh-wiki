# gh-wiki

A [GitHub CLI](https://cli.github.com) extension for editing a repository's
GitHub wiki as ordinary local files. It keeps one managed Git clone per wiki,
opens that clone in your editor, and reconciles remote changes before pushing.

```console
$ gh wiki list
Home
Release process
Troubleshooting

$ gh wiki new "Cache layout"
Created /Users/you/Library/Caches/gh-wiki/github.com/you/project/Cache-layout.md

$ gh wiki push -m "Document the cache layout"
Pushed wiki changes to you/project.
```

**Why not just clone the wiki yourself?** You can — a GitHub wiki is only a Git
repository at `OWNER/REPO.wiki.git`. This does the parts that are tedious or
easy to get wrong: it finds and clones the right wiki without you naming it,
keeps one cache per repository so you are never unsure which copy you edited,
converts page titles to the filenames GitHub expects, and — the reason it
exists — never resolves a race with the web editor by throwing away your work.

## Install

```sh
gh extension install pbleisch/gh-wiki
```

Releases are precompiled for the platforms GitHub CLI supports, so you do not
need Go installed. To build from source instead:

```sh
go build -o gh-wiki . && gh extension install .
```

Requires `git` on your `PATH` and an authenticated `gh` (`gh auth status`).

## Commands

Commands act on the repository in the current directory. Pass
`--repo OWNER/REPO` or `--repo HOST/OWNER/REPO` to select another one, including
on GitHub Enterprise Server.

```text
gh wiki list                 List the wiki's pages
gh wiki edit                 Sync and open the whole wiki in an editor
gh wiki edit --code          Open the wiki in VS Code
gh wiki new "Release notes"  Create and edit Release-notes.md
gh wiki push -m "Add notes"  Commit, reconcile, and push local edits
gh wiki sync                 Fetch and safely update the cached wiki
gh wiki status               Show edits and local/remote divergence
gh wiki path                 Print the managed clone's path
```

Run `gh wiki --help` or `gh wiki COMMAND --help` for every option.

Progress messages go to standard error and data goes to standard output, so
`list`, `path`, and `status` compose with other tools:

```sh
gh wiki list --files | xargs wc -l
cd "$(gh wiki path)"
```

### Editing

`gh wiki edit` uses the first editor it finds from:

1. `--editor COMMAND`
2. `$VISUAL`
3. `$EDITOR`
4. `code --wait`, when VS Code's `code` command is available
5. `vi`

Use `--code` to force `code --wait`. Editor values may include arguments, for
example `--editor "code --reuse-window --wait"`.

### Page titles and filenames

GitHub stores a page titled `Release process` as `Release-process.md` and
derives the displayed title by turning the hyphens back into spaces. `gh wiki
new` applies the same mapping, and `gh wiki list` reverses it.

Two consequences are worth knowing, because they are GitHub's, not this tool's:

- **A literal hyphen is indistinguishable from a space.** `API-v2.md` displays
  as `API v2` on GitHub and here. Nothing can recover the difference.
- **GitHub identifies a page by its filename alone.** A wiki repository may
  contain directories, and GitHub does render pages inside them, but it ignores
  the directory: `archive/Old-notes.md` is listed and served as `Old notes`, and
  `/wiki/archive/Old-notes` does not resolve. So `Notes.md`, `notes.md`, and
  `archive/Notes.md` are all the same page to GitHub, and only one of them is
  reachable. `gh wiki list` warns when a wiki contains such a set, and
  `gh wiki new` refuses to create one.

For the same reason, `gh wiki new` rejects a `/` in a title rather than
guessing. GitHub documents `\ / : * ? " < > |` as characters not to use in a
wiki title, its own editor silently strips a slash out of one, and a directory
is not a page hierarchy — so there is no faithful thing to do with it.

### Syncing and pushing

`sync` fetches remote changes and fast-forwards only when that cannot hide or
overwrite local work. If the cache has uncommitted edits or divergent commits,
it explains what to do and leaves every local file untouched.

`push` stages and commits all wiki changes, fetches again, and rebases your wiki
commits onto the current remote branch before pushing. If someone edited the
same page on github.com in the meantime, your commits are replayed on top of
theirs. On a conflict, Git stops in the managed clone with all work preserved:
resolve the listed files, run the printed `git -C ... rebase --continue`, then
run `gh wiki push` again. If the remote moves during the push itself, it
reconciles and retries once, then gives up rather than force-pushing. Omit `-m`
to write the commit message in Git's editor.

## Cache

Wikis are cloned into the operating system's user cache directory:

- macOS: `~/Library/Caches/gh-wiki`
- Linux: usually `~/.cache/gh-wiki`
- Windows: the directory returned by the Windows user-cache API

Clones are keyed by host, owner, and repository, so identically named
repositories never collide. Set `GH_WIKI_CACHE_DIR` to override the root.

To clear one wiki, delete the directory `gh wiki path` prints; to clear all of
them, delete the cache root above. Unpushed edits live in these directories, so
check `gh wiki status` first. Nothing is deleted automatically, and a failed
network call never removes an existing cache.

## Expected errors

- **The wiki is disabled** — enable **Wikis** in the repository settings.
- **The wiki has no pages yet** — GitHub does not create the
  `OWNER/REPO.wiki.git` repository until its first page exists. Create that page
  on github.com, then retry.
- **Authentication fails** — credentials come from GitHub CLI. Check
  `gh auth status` and repair with `gh auth login`.
- **No repository found** — run inside a GitHub repository or pass `--repo`.

## Design and research notes

A GitHub wiki is a separate Git repository at
`https://HOST/OWNER/REPO.wiki.git`. This extension uses
[`cli/go-gh`](https://github.com/cli/go-gh) for repository resolution, API
access, and authentication, and lets Git perform the clone, fetch, rebase, and
push. Credentials therefore stay in GitHub CLI rather than in another token
store; see [SECURITY.md](SECURITY.md) for exactly how they reach Git.

The command set targets what the web editor is bad at: working across several
pages at once, using a real editor, reviewing changes before publishing, and
landing one coherent commit. It deliberately omits rendering, search, attachment
management, page deletion, and a second history UI — your editor and Git already
do those, and duplicating them would make the tool harder to understand rather
than more capable. `new` exists because the title-to-filename conversion is easy
to get wrong by hand. `_Sidebar` and `_Footer` stay ordinary editable files, and
are recognized as GitHub's layout files so `list` hides them by default, as
GitHub's own page index does.

GitHub CLI discovers an extension from a `gh-NAME` repository containing a
same-named executable and invokes it as `gh NAME`. `gh extension install`
prefers release artifacts for binary extensions, so this project ships
precompiled binaries built by
[`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile).

Primary references:

- [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
- [`gh extension` manual](https://cli.github.com/manual/gh_extension)
- [GitHub wiki documentation](https://docs.github.com/en/communities/documenting-your-project-with-wikis)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Bug reports and small fixes are welcome;
please open an issue before a large change.

## License

[MIT](LICENSE) © Paul Bleisch
