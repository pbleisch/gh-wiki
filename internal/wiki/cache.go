package wiki

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const cacheDirEnv = "GH_WIKI_CACHE_DIR"

// CacheRoot follows the operating system cache convention through
// os.UserCacheDir. GH_WIKI_CACHE_DIR is an explicit override.
func CacheRoot() (string, error) {
	if root := os.Getenv(cacheDirEnv); root != "" {
		return filepath.Abs(root)
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find user cache directory: %w", err)
	}
	return filepath.Join(root, "gh-wiki"), nil
}

// CachePath keys wiki clones by host, owner, and repository so repositories
// with the same name never collide.
func CachePath(root, host, owner, repo string) (string, error) {
	parts := []string{host, owner, repo}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid empty repository cache key")
		}
		part = strings.ToLower(part)
		part = url.PathEscape(part)
		part = strings.ReplaceAll(part, ":", "%3A")
		part = strings.ReplaceAll(part, `\`, "%5C")
		parts[i] = part
	}
	return filepath.Join(root, parts[0], parts[1], parts[2]), nil
}
