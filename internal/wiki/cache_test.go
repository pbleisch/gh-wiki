package wiki

import (
	"path/filepath"
	"testing"
)

func TestCachePathKeysByRepository(t *testing.T) {
	t.Parallel()
	root := filepath.Join("tmp", "cache")
	got, err := CachePath(root, "GitHub.COM", "OctoCat", "Hello-World")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "github.com", "octocat", "hello-world")
	if got != want {
		t.Errorf("CachePath() = %q, want %q", got, want)
	}

	other, err := CachePath(root, "github.example", "OctoCat", "Hello-World")
	if err != nil {
		t.Fatal(err)
	}
	if other == got {
		t.Errorf("different hosts produced the same path %q", got)
	}
}

func TestCachePathEscapesUnsafeSegments(t *testing.T) {
	t.Parallel()
	got, err := CachePath("cache", "github.example:8443", `owner/name\team`, "repo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("cache", "github.example%3A8443", "owner%2Fname%5Cteam", "repo")
	if got != want {
		t.Errorf("CachePath() = %q, want %q", got, want)
	}
}

func TestCachePathRejectsTraversalSegments(t *testing.T) {
	t.Parallel()
	for _, segment := range []string{"", ".", "..", "   "} {
		if _, err := CachePath("cache", segment, "owner", "repo"); err == nil {
			t.Errorf("CachePath() accepted host %q", segment)
		}
	}
}

func TestCacheRootOverride(t *testing.T) {
	root := filepath.Join(t.TempDir(), "custom")
	t.Setenv(cacheDirEnv, root)
	got, err := CacheRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("CacheRoot() = %q, want %q", got, want)
	}
}
