package wiki

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

var forbiddenTitleCharacters = strings.NewReplacer(
	"\\", "", "/", "", ":", "", "*", "", "?", "", `"`, "", "<", "", ">", "", "|", "",
)

// TitleToFilename applies GitHub wiki's canonical title mapping. GitHub turns
// spaces into hyphens when a page is created and derives the rendered title by
// turning filename hyphens back into spaces. Literal hyphens are therefore
// inherently indistinguishable from spaces.
//
// Titles are rejected rather than silently rewritten. GitHub documents
// \ / : * ? " < > | as characters not to use in a wiki title, and its own web
// editor strips a slash out of a typed title instead of nesting the page. A
// slash cannot mean "subpage" here either: directories may exist in a wiki
// repository, but GitHub identifies a page by its filename alone and ignores
// the directory entirely, so a slash in a title has no faithful mapping.
func TitleToFilename(title, extension string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("page title cannot be empty")
	}
	if strings.Contains(title, "/") {
		return "", fmt.Errorf("page title %q cannot contain '/': GitHub's own wiki editor strips it, and a directory in the wiki repository is not a page hierarchy because GitHub titles every page by its filename alone", title)
	}
	if forbiddenTitleCharacters.Replace(title) != title {
		return "", fmt.Errorf(`page title %q contains a character GitHub does not allow in a wiki title: \ / : * ? " < > |`, title)
	}
	for _, r := range title {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("page title cannot contain control characters")
		}
	}
	if extension == "" {
		extension = ".md"
	}
	if !strings.HasPrefix(extension, ".") || strings.ContainsAny(extension, `/\`) {
		return "", fmt.Errorf("invalid page extension %q", extension)
	}

	return strings.ReplaceAll(title, " ", "-") + extension, nil
}

// FilenameToTitle returns the title GitHub displays for a wiki page filename.
// Any directory is dropped, matching GitHub: a page committed as
// archive/Old-notes.md is listed and served as "Old notes".
func FilenameToTitle(filename string) (string, error) {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	if ext == "" {
		return "", fmt.Errorf("wiki page filename %q has no markup extension", filename)
	}
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return "", fmt.Errorf("wiki page filename %q has no title", filename)
	}
	return strings.ReplaceAll(name, "-", " "), nil
}

// SpecialPage identifies GitHub's non-page wiki files.
func SpecialPage(filename string) string {
	name := strings.ToLower(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	switch name {
	case "_sidebar":
		return "sidebar"
	case "_footer":
		return "footer"
	default:
		return ""
	}
}
