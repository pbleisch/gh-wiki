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
	got, err := os.ReadFile(filepath.Join(fixture.cache, "Remote.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "remote\n" {
		t.Fatalf("Remote.md = %q, want remote content", got)
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

	verification := filepath.Join(t.TempDir(), "verification")
	fixture.run("", "clone", fixture.remote, verification)
	for name, want := range map[string]string{
		"Local.md":  "local work\n",
		"Remote.md": "remote work\n",
	} {
		got, err := os.ReadFile(filepath.Join(verification, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	log := fixture.git(verification, "log", "--format=%s", "-2")
	if !strings.Contains(log, "local edit") || !strings.Contains(log, "remote edit") {
		t.Errorf("remote log does not contain both commits:\n%s", log)
	}
}

type gitFixture struct {
	t      *testing.T
	remote string
	seed   string
	cache  string
}

func newGitFixture(t *testing.T) *gitFixture {
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
	fixture.run(fixture.seed, "config", "user.name", "gh-wiki tests")
	fixture.run(fixture.seed, "config", "user.email", "gh-wiki@example.invalid")
	if err := os.WriteFile(filepath.Join(fixture.seed, "Home.md"), []byte("# Home\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixture.run(fixture.seed, "add", "Home.md")
	fixture.run(fixture.seed, "commit", "-m", "initial page")
	fixture.run(fixture.seed, "remote", "add", "origin", fixture.remote)
	fixture.run(fixture.seed, "push", "-u", "origin", "main")
	fixture.run("", "clone", fixture.remote, fixture.cache)
	fixture.run(fixture.cache, "config", "user.name", "gh-wiki tests")
	fixture.run(fixture.cache, "config", "user.email", "gh-wiki@example.invalid")
	return fixture
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
