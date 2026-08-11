package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) CreatePage(title string) (string, error) {
	filename, err := TitleToFilename(title, ".md")
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(m.CachePath)
	if err != nil {
		return "", fmt.Errorf("list wiki cache: %w", err)
	}
	wantedStem := strings.TrimSuffix(filename, filepath.Ext(filename))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if strings.EqualFold(stem, wantedStem) {
			return "", fmt.Errorf("wiki page %q already exists as %s", title, entry.Name())
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
