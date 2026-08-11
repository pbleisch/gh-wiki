package wiki

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// markupExtensions are the extensions GitHub's wiki renderer treats as pages.
// GitHub wikis are served by gollum, which picks a converter from the file
// extension; anything outside this set - images, PDFs, and other attachments
// committed to the wiki repository - is not a page.
var markupExtensions = map[string]bool{
	".markdown":  true,
	".md":        true,
	".mdown":     true,
	".mkd":       true,
	".mkdn":      true,
	".textile":   true,
	".rdoc":      true,
	".org":       true,
	".creole":    true,
	".rest":      true,
	".rst":       true,
	".asciidoc":  true,
	".adoc":      true,
	".mediawiki": true,
	".wiki":      true,
	".pod":       true,
	".bib":       true,
	".txt":       true,
}

// Page is a single wiki page as GitHub identifies it.
type Page struct {
	// Title is the title GitHub displays for the page.
	Title string
	// Path is the page's location in the wiki repository, relative to the
	// cache root and always slash-separated.
	Path string
	// Special is "sidebar" or "footer" for GitHub's layout files, and empty
	// for an ordinary page.
	Special string
}

// Pages lists every page in the cached wiki, sorted by title.
func (m *Manager) Pages() ([]Page, error) {
	return pagesIn(m.CachePath)
}

// pagesIn walks a wiki working tree. Subdirectories are included because a
// wiki repository may contain them, but GitHub flattens them: it identifies a
// page by its filename alone, so a nested page is titled and served under its
// own name with no directory prefix. See Collisions for the consequence.
func pagesIn(root string) ([]Page, error) {
	var pages []Page
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !markupExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			return nil
		}
		title, err := FilenameToTitle(entry.Name())
		if err != nil {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		pages = append(pages, Page{
			Title:   title,
			Path:    filepath.ToSlash(relative),
			Special: SpecialPage(entry.Name()),
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("list wiki pages: %w", walkErr)
	}
	sort.Slice(pages, func(i, j int) bool {
		left, right := strings.ToLower(pages[i].Title), strings.ToLower(pages[j].Title)
		if left != right {
			return left < right
		}
		return pages[i].Path < pages[j].Path
	})
	return pages, nil
}

// Collisions groups pages GitHub cannot tell apart. Because a page is
// identified by its filename alone, ignoring both its directory and its
// markup extension, files such as Notes.md, notes.md, and archive/Notes.md all
// name the same wiki page and only one of them is reachable on GitHub. Each
// returned group holds two or more pages, ordered by path.
func Collisions(pages []Page) [][]Page {
	order := make([]string, 0, len(pages))
	grouped := make(map[string][]Page, len(pages))
	for _, page := range pages {
		key := strings.ToLower(page.Title)
		if _, seen := grouped[key]; !seen {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], page)
	}
	var collisions [][]Page
	for _, key := range order {
		group := grouped[key]
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(i, j int) bool { return group[i].Path < group[j].Path })
		collisions = append(collisions, group)
	}
	return collisions
}
