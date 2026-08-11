# Security policy

## What this tool touches

`gh-wiki` is a GitHub CLI extension that runs on your machine with your
privileges. Two parts of what it does are security-relevant, and both are
deliberate:

- **It uses your GitHub CLI credentials.** It does not store a token of its own
  and never writes one to disk. When it runs a Git command that needs the
  network, it sets `GIT_ASKPASS` to its own executable plus a
  `GH_WIKI_ASKPASS_HOST` environment variable, and Git calls it back to ask for
  the username and password. In that callback the process prints the token that
  `gh` holds for the host, for that host only, on standard output. Anything that
  can already set environment variables and run programs as you could equally
  run `gh auth token`, so this does not widen your exposure, but it does mean
  the token passes through this program.
- **It clones repositories to your disk.** The wiki of every repository you
  touch is cloned into your user cache directory (see the README for the exact
  paths). Wikis of private repositories therefore end up in plain files on
  local disk. The cache directories this tool creates are made with `0700` and
  new pages with `0600`, but Git creates the clone's own files with your normal
  umask. Cached wikis are not encrypted and are not removed automatically.

The extension makes network requests to the GitHub host you target and nowhere
else. It collects no telemetry.

## Reporting a vulnerability

Report privately through GitHub, using **Security → Report a vulnerability** on
this repository, which opens a private security advisory. Please do not open a
public issue for a suspected vulnerability.

A useful report covers:

- what an attacker gains, and what access they need to start,
- the commands or repository state that reproduce it,
- the `gh-wiki` version (`gh wiki --version`), your OS, and your Git version.

Things worth reporting include the token leaking anywhere other than the Git
callback described above, this tool executing something an attacker controls
(a repository name, page title, branch name, or editor value reaching a shell),
a wiki page or cache path escaping the cache directory, and any path where
`sync` or `push` destroys local edits rather than stopping.

## What to expect

This is a small tool maintained by one person in their spare time. Reports are
read and taken seriously, but no response time is promised and there is no bug
bounty. Fixes ship in a normal tagged release; the advisory will credit you
unless you ask otherwise.

Only the most recent release is supported. There are no backports to earlier
tags.
