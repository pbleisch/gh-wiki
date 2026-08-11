package wiki

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncFastForwardsLocalGitClone(t *testing.T) {
	fixture := newGitFixture(t)
	fixture.remoteCommit("Remote.md", "remote\n", "remote edit")

	manager := fixture.manager()
	snapshot, err := manager.Sync()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Behind != 0 {
		t.Fatalf("Sync() snapshot behind = %d, want 0", snapshot.Behind)
	}
	// The bytes on disk are whatever Git's checkout filter produces, which on
	// Windows means CRLF. What Sync() owes the caller is the remote's content,
	// so compare the page by line rather than by raw byte.
	got, err := os.ReadFile(filepath.Join(fixture.cache, "Remote.md"))
	if err != nil {
		t.Fatal(err)
	}
	if normalizeEOL(string(got)) != "remote\n" {
		t.Fatalf("Remote.md = %q, want remote content", got)
	}
	// A checked-out page must also be clean in Git's eyes. If it were not, the
	// next push would carry a whole-file rewrite nobody asked for.
	if status := fixture.git(fixture.cache, "status", "--porcelain"); status != "" {
		t.Fatalf("cache is dirty straight after Sync():\n%s", status)
	}
}

func TestSyncPreservesDirtyCacheWhenRemoteMoved(t *testing.T) {
	fixture := newGitFixture(t)
	localPath := filepath.Join(fixture.cache, "Local.md")
	if err := os.WriteFile(localPath, []byte("local work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := fixture.git(fixture.cache, "rev-parse", "HEAD")
	fixture.remoteCommit("Remote.md", "remote\n", "remote edit")

	_, err := fixture.manager().Sync()
	if err == nil || !strings.Contains(err.Error(), "edits were left untouched") {
		t.Fatalf("Sync() error = %v, want preserved-edits error", err)
	}
	if after := fixture.git(fixture.cache, "rev-parse", "HEAD"); after != before {
		t.Fatalf("Sync() moved HEAD from %s to %s", before, after)
	}
	got, readErr := os.ReadFile(localPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "local work\n" {
		t.Fatalf("Local.md = %q after Sync()", got)
	}
}

func TestPushCommitsAndRebasesOntoRemoteChanges(t *testing.T) {
	fixture := newGitFixture(t)
	if err := os.WriteFile(filepath.Join(fixture.cache, "Local.md"), []byte("local work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixture.remoteCommit("Remote.md", "remote work\n", "remote edit")

	manager := fixture.manager()
	if err := manager.Push("local edit"); err != nil {
		t.Fatal(err)
	}

	// Assert on the blobs in the remote itself. That is the wiki's own copy,
	// unfiltered by any checkout, and so is the same on every platform.
	for name, want := range map[string]string{
		"Local.md":  "local work\n",
		"Remote.md": "remote work\n",
	} {
		if got := fixture.remoteBlob(name); got != want {
			t.Errorf("%s on the remote = %q, want %q", name, got, want)
		}
	}
	log := fixture.git(fixture.remote, "log", "--format=%s", "-2", "main")
	if !strings.Contains(log, "local edit") || !strings.Contains(log, "remote edit") {
		t.Errorf("remote log does not contain both commits:\n%s", log)
	}
}

// TestPushPreservesRemoteLineEndings pins the behaviour that matters to a wiki:
// pushing must not rewrite the line endings of pages the user did not edit, and
// must not rewrite the untouched lines of the page they did.
//
// The cache is forced to core.autocrlf=true - Git for Windows' installer
// default, and the reason this package's tests once failed only on Windows - so
// every platform exercises the checkout filter. The fidelity comes from Git
// itself: with autocrlf=true a page whose committed blob already contains CRLF
// is exempted from conversion in both directions, and a page committed with LF
// is converted to CRLF on checkout and back to LF on commit. The tool must not
// disturb that by pinning core.autocrlf, setting core.eol, or writing a
// .gitattributes into someone else's wiki; if it ever does, this test fails.
func TestPushPreservesRemoteLineEndings(t *testing.T) {
	fixture := newGitFixtureWithCacheAutocrlf(t, "true")
	fixture.remoteCommit("Crlf.md", "alpha\r\nbeta\r\n", "page saved with CRLF")
	fixture.remoteCommit("Lf.md", "alpha\nbeta\n", "page saved with LF")
	if _, err := fixture.manager().Sync(); err != nil {
		t.Fatal(err)
	}

	// Guard the guard: if the checkout filter is not actually engaged, the rest
	// of this test proves nothing.
	onDisk, err := os.ReadFile(filepath.Join(fixture.cache, "Lf.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(onDisk, []byte("\r\n")) {
		t.Fatalf("core.autocrlf=true did not convert Lf.md on checkout: %q", onDisk)
	}

	// The user edits one line of one page in an editor that keeps the CRLF it
	// found on disk, which is what a Windows editor does.
	if err := os.WriteFile(filepath.Join(fixture.cache, "Crlf.md"), []byte("alpha\r\nbeta EDITED\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fixture.manager().Push("edit one line"); err != nil {
		t.Fatal(err)
	}

	if got := fixture.remoteBlob("Crlf.md"); got != "alpha\r\nbeta EDITED\r\n" {
		t.Errorf("edited CRLF page on the remote = %q, want only the edited line changed", got)
	}
	if got := fixture.remoteBlob("Lf.md"); got != "alpha\nbeta\n" {
		t.Errorf("untouched LF page on the remote = %q, want it unchanged", got)
	}
	if got := fixture.remoteBlob("Home.md"); got != "# Home\n" {
		t.Errorf("untouched seed page on the remote = %q, want it unchanged", got)
	}
}

// TestPushWritesCreatedPagesWithLineFeeds covers the content the tool composes
// itself rather than the content Git checked out. CreatePage writes through the
// filesystem, bypassing Git's filters, so its line endings reach the wiki as
// written even on a cache that checks out CRLF.
func TestPushWritesCreatedPagesWithLineFeeds(t *testing.T) {
	fixture := newGitFixtureWithCacheAutocrlf(t, "true")
	manager := fixture.manager()
	if _, err := manager.CreatePage("Release notes"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Push("add release notes"); err != nil {
		t.Fatal(err)
	}
	if got := fixture.remoteBlob("Release-notes.md"); got != "# Release notes\n\n" {
		t.Errorf("created page on the remote = %q, want line feeds", got)
	}
}

// normalizeEOL collapses CRLF to LF so an assertion about a page's lines does
// not become an assertion about the platform's checkout convention.
func normalizeEOL(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

type gitFixture struct {
	t      *testing.T
	remote string
	seed   string
	cache  string
}

func newGitFixture(t *testing.T) *gitFixture {
	t.Helper()
	return newGitFixtureWithCacheAutocrlf(t, "")
}

// newGitFixtureWithCacheAutocrlf builds a bare remote, a seed clone that authors
// content, and the cache the Manager operates on. The seed always pins
// core.autocrlf=false so the bytes it commits are the bytes the test asked for
// on every platform. cacheAutocrlf pins the cache's own setting; "" leaves the
// platform default in place, which is how the Windows path gets exercised in CI.
func newGitFixtureWithCacheAutocrlf(t *testing.T, cacheAutocrlf string) *gitFixture {
	t.Helper()
	root := t.TempDir()
	fixture := &gitFixture{
		t:      t,
		remote: filepath.Join(root, "wiki.git"),
		seed:   filepath.Join(root, "seed"),
		cache:  filepath.Join(root, "cache"),
	}
	fixture.run("", "init", "--bare", "--initial-branch=main", fixture.remote)
	fixture.run("", "init", "--initial-branch=main", fixture.seed)
	fixture.configure(fixture.seed)
	fixture.run(fixture.seed, "config", "core.autocrlf", "false")
	if err := os.WriteFile(filepath.Join(fixture.seed, "Home.md"), []byte("# Home\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixture.run(fixture.seed, "add", "Home.md")
	fixture.run(fixture.seed, "commit", "-m", "initial page")
	fixture.run(fixture.seed, "remote", "add", "origin", fixture.remote)
	fixture.run(fixture.seed, "push", "-u", "origin", "main")
	cloneArgs := []string{"clone", fixture.remote, fixture.cache}
	if cacheAutocrlf != "" {
		cloneArgs = append([]string{"-c", "core.autocrlf=" + cacheAutocrlf}, cloneArgs...)
	}
	fixture.run("", cloneArgs...)
	fixture.configure(fixture.cache)
	if cacheAutocrlf != "" {
		fixture.run(fixture.cache, "config", "core.autocrlf", cacheAutocrlf)
	}
	return fixture
}

func (f *gitFixture) configure(dir string) {
	f.t.Helper()
	f.run(dir, "config", "user.name", "gh-wiki tests")
	f.run(dir, "config", "user.email", "gh-wiki@example.invalid")
}

func (f *gitFixture) manager() *Manager {
	f.t.Helper()
	var output bytes.Buffer
	manager, err := NewManager(Repo{Host: "github.com", Owner: "owner", Name: "repo"}, filepath.Dir(filepath.Dir(filepath.Dir(f.cache))), &output, &output)
	if err != nil {
		f.t.Fatal(err)
	}
	manager.CachePath = f.cache
	return manager
}

func (f *gitFixture) remoteCommit(name, content, message string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.seed, name), []byte(content), 0o600); err != nil {
		f.t.Fatal(err)
	}
	f.run(f.seed, "add", name)
	f.run(f.seed, "commit", "-m", message)
	f.run(f.seed, "push", "origin", "main")
}

// remoteBlob reads a file from the bare remote's tip commit. Reading the object
// rather than a working tree keeps the assertion on the bytes the wiki actually
// stores, with no checkout filter in between.
func (f *gitFixture) remoteBlob(name string) string {
	f.t.Helper()
	command := exec.Command("git", "-C", f.remote, "cat-file", "-p", "main:"+name)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		f.t.Fatalf("read %s from the remote: %v\n%s", name, err, stderr.String())
	}
	return stdout.String()
}

func (f *gitFixture) git(dir string, args ...string) string {
	f.t.Helper()
	return strings.TrimSpace(f.run(dir, args...))
}

func (f *gitFixture) run(dir string, args ...string) string {
	f.t.Helper()
	commandArgs := args
	if dir != "" {
		commandArgs = append([]string{"-C", dir}, args...)
	}
	command := exec.Command("git", commandArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s failed: %v\n%s", strings.Join(commandArgs, " "), err, output)
	}
	return string(output)
}
