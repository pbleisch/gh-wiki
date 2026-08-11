package wiki

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeWiki lays out a wiki working tree and returns its root.
func writeWiki(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, file := range files {
		full := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("body\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func titlesOf(pages []Page) []string {
	titles := make([]string, 0, len(pages))
	for _, page := range pages {
		titles = append(titles, page.Title)
	}
	return titles
}

func TestPagesInListsMarkupFilesByTitle(t *testing.T) {
	t.Parallel()
	root := writeWiki(t,
		"Home.md",
		"Release-process.md",
		"Design-notes.textile",
		"AsciiDoc-tests.asciidoc",
	)

	pages, err := pagesIn(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"AsciiDoc tests", "Design notes", "Home", "Release process"}
	if got := titlesOf(pages); !reflect.DeepEqual(got, want) {
		t.Errorf("pagesIn() titles = %q, want %q", got, want)
	}
}

func TestPagesInSkipsNonPageFiles(t *testing.T) {
	t.Parallel()
	root := writeWiki(t,
		"Home.md",
		"images/diagram.png",
		"handbook.pdf",
		"Makefile",
		".git/config",
		".git/refs/heads/main",
	)

	pages, err := pagesIn(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := titlesOf(pages); !reflect.DeepEqual(got, []string{"Home"}) {
		t.Errorf("pagesIn() titles = %q, want [Home]", got)
	}
}

// GitHub renders pages committed inside a directory, but titles them by
// filename alone, so the directory must not appear in the title.
func TestPagesInFlattensNestedPages(t *testing.T) {
	t.Parallel()
	root := writeWiki(t,
		"Home.md",
		"subfolder/Sub-page.md",
		"subfolder/subsubfolder/Deep-page.md",
	)

	pages, err := pagesIn(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Deep page", "Home", "Sub page"}
	if got := titlesOf(pages); !reflect.DeepEqual(got, want) {
		t.Fatalf("pagesIn() titles = %q, want %q", got, want)
	}
	byTitle := map[string]string{}
	for _, page := range pages {
		byTitle[page.Title] = page.Path
	}
	if got := byTitle["Deep page"]; got != "subfolder/subsubfolder/Deep-page.md" {
		t.Errorf("nested page path = %q, want slash-separated repository path", got)
	}
}

func TestPagesInMarksSpecialPages(t *testing.T) {
	t.Parallel()
	root := writeWiki(t, "Home.md", "_Sidebar.md", "_Footer.md")

	pages, err := pagesIn(root)
	if err != nil {
		t.Fatal(err)
	}
	special := map[string]string{}
	for _, page := range pages {
		special[page.Title] = page.Special
	}
	want := map[string]string{"Home": "", "_Sidebar": "sidebar", "_Footer": "footer"}
	if !reflect.DeepEqual(special, want) {
		t.Errorf("pagesIn() special = %v, want %v", special, want)
	}
}

func TestPagesInOnMissingDirectory(t *testing.T) {
	t.Parallel()
	if _, err := pagesIn(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("pagesIn() on a missing directory unexpectedly succeeded")
	}
}

func TestCollisionsFindsPagesGitHubCannotTellApart(t *testing.T) {
	t.Parallel()
	root := writeWiki(t,
		"Home.md",
		"Notes.md",
		"archive/Notes.md",
		"Design.md",
		"Design.textile",
		"Unique-page.md",
	)

	pages, err := pagesIn(root)
	if err != nil {
		t.Fatal(err)
	}
	var got [][]string
	for _, group := range Collisions(pages) {
		paths := make([]string, 0, len(group))
		for _, page := range group {
			paths = append(paths, page.Path)
		}
		got = append(got, paths)
	}
	want := [][]string{
		{"Design.md", "Design.textile"},
		{"Notes.md", "archive/Notes.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collisions() = %v, want %v", got, want)
	}
}

func TestCollisionsIgnoresCaseAndReportsNothingWhenUnique(t *testing.T) {
	t.Parallel()
	root := writeWiki(t, "Notes.md", "sub/notes.md")
	pages, err := pagesIn(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := Collisions(pages); len(got) != 1 || len(got[0]) != 2 {
		t.Errorf("Collisions() = %v, want one group of two case-insensitive matches", got)
	}

	unique := writeWiki(t, "Home.md", "sub/Other.md")
	uniquePages, err := pagesIn(unique)
	if err != nil {
		t.Fatal(err)
	}
	if got := Collisions(uniquePages); got != nil {
		t.Errorf("Collisions() = %v, want none", got)
	}
}
