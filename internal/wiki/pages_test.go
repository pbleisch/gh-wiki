package wiki

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func managerAt(t *testing.T, cache string) *Manager {
	t.Helper()
	manager, err := NewManager(Repo{Host: "github.com", Owner: "owner", Name: "repo"}, t.TempDir(), io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	manager.CachePath = cache
	return manager
}

func TestCreatePageWritesTitledMarkdown(t *testing.T) {
	t.Parallel()
	cache := writeWiki(t, "Home.md")
	manager := managerAt(t, cache)

	path, err := manager.CreatePage("Release process")
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.Base(path); got != "Release-process.md" {
		t.Errorf("CreatePage() wrote %q, want Release-process.md", got)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "# Release process\n\n" {
		t.Errorf("CreatePage() body = %q", body)
	}
}

func TestCreatePageLeavesSpecialPagesEmpty(t *testing.T) {
	t.Parallel()
	manager := managerAt(t, writeWiki(t, "Home.md"))

	path, err := manager.CreatePage("_Sidebar")
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Errorf("CreatePage(_Sidebar) body = %q, want empty", body)
	}
}

// GitHub's page namespace is flat, so an existing page in a subdirectory or
// under a different markup extension already occupies the title.
func TestCreatePageRejectsPagesGitHubCannotTellApart(t *testing.T) {
	t.Parallel()
	cache := writeWiki(t, "Home.md", "archive/Release-notes.md", "Design.textile")
	manager := managerAt(t, cache)

	for _, test := range []struct {
		title    string
		conflict string
	}{
		{title: "Release notes", conflict: "archive/Release-notes.md"},
		{title: "release NOTES", conflict: "archive/Release-notes.md"},
		{title: "Design", conflict: "Design.textile"},
		{title: "Home", conflict: "Home.md"},
	} {
		t.Run(test.title, func(t *testing.T) {
			_, err := manager.CreatePage(test.title)
			if err == nil {
				t.Fatalf("CreatePage(%q) unexpectedly succeeded", test.title)
			}
			if !strings.Contains(err.Error(), test.conflict) {
				t.Errorf("CreatePage(%q) error = %v, want it to name %s", test.title, err, test.conflict)
			}
		})
	}
}

func TestCreatePageRejectsSlashTitles(t *testing.T) {
	t.Parallel()
	manager := managerAt(t, writeWiki(t, "Home.md"))

	_, err := manager.CreatePage("Tutorials/Getting started")
	if err == nil {
		t.Fatal("CreatePage() with a slash unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "'/'") {
		t.Errorf("CreatePage() error = %v, want it to explain the slash", err)
	}
	entries, readErr := os.ReadDir(manager.CachePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 1 {
		t.Errorf("CreatePage() changed the cache: %v", entries)
	}
}
