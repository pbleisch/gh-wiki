package wiki

import "testing"

func TestTitleToFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		title     string
		extension string
		want      string
	}{
		{name: "spaces", title: "Release process", want: "Release-process.md"},
		{name: "trim", title: "  Home  ", want: "Home.md"},
		{name: "literal hyphen is preserved", title: "API-v2", want: "API-v2.md"},
		{name: "multiple spaces", title: "One  Two", want: "One--Two.md"},
		{name: "unicode", title: "日本語 入門", want: "日本語-入門.md"},
		{name: "other markup", title: "Notes", extension: ".textile", want: "Notes.textile"},
		{name: "special page", title: "_Sidebar", want: "_Sidebar.md"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := TitleToFilename(test.title, test.extension)
			if err != nil {
				t.Fatalf("TitleToFilename() error = %v", err)
			}
			if got != test.want {
				t.Errorf("TitleToFilename() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestTitleToFilenameRejectsUnsafeTitles(t *testing.T) {
	t.Parallel()
	for _, title := range []string{"", "   ", `A/B`, `A\\B`, "A:B", "A*B", "A?B", `A"B`, "A<B", "A>B", "A|B", "A\nB"} {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			if _, err := TitleToFilename(title, ".md"); err == nil {
				t.Errorf("TitleToFilename(%q) unexpectedly succeeded", title)
			}
		})
	}
}

func TestFilenameToTitle(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"Release-process.md":        "Release process",
		"docs/API-v2.markdown":      "API v2",
		"One--Two.textile":          "One  Two",
		"日本語-入門.md":               "日本語 入門",
		"_Sidebar.md":               "_Sidebar",
	}
	for filename, want := range tests {
		filename, want := filename, want
		t.Run(filename, func(t *testing.T) {
			t.Parallel()
			got, err := FilenameToTitle(filename)
			if err != nil {
				t.Fatalf("FilenameToTitle() error = %v", err)
			}
			if got != want {
				t.Errorf("FilenameToTitle() = %q, want %q", got, want)
			}
		})
	}
}

func TestSpecialPage(t *testing.T) {
	t.Parallel()
	if got := SpecialPage("_Sidebar.md"); got != "sidebar" {
		t.Errorf("SpecialPage(_Sidebar.md) = %q", got)
	}
	if got := SpecialPage("_footer.textile"); got != "footer" {
		t.Errorf("SpecialPage(_footer.textile) = %q", got)
	}
	if got := SpecialPage("Sidebar.md"); got != "" {
		t.Errorf("SpecialPage(Sidebar.md) = %q", got)
	}
}
