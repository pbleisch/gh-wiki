package wiki

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// CreatePage writes an empty Markdown page for title and returns its path.
//
// The collision check covers the whole cache, not just its top level, because
// GitHub's page namespace is flat: a page already committed as
// archive/Release-notes.md occupies the title "Release notes", so creating
// Release-notes.md would produce two files GitHub cannot tell apart.
func (m *Manager) CreatePage(title string) (string, error) {
	filename, err := TitleToFilename(title, ".md")
	if err != nil {
		return "", err
	}
	pages, err := m.Pages()
	if err != nil {
		return "", err
	}
	wanted := strings.TrimSuffix(filename, filepath.Ext(filename))
	for _, page := range pages {
		base := path.Base(page.Path)
		if strings.EqualFold(strings.TrimSuffix(base, path.Ext(base)), wanted) {
			return "", fmt.Errorf("wiki page %q already exists as %s", title, page.Path)
		}
	}
	path := filepath.Join(m.CachePath, filename)
	content := ""
	if SpecialPage(filename) == "" {
		displayed, _ := FilenameToTitle(filename)
		content = "# " + displayed + "\n\n"
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create wiki page: %w", err)
	}
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write wiki page: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close wiki page: %w", err)
	}
	return path, nil
}
