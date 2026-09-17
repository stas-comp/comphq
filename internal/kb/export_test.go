package kb

import (
	"context"
	"strings"
	"testing"
)

func TestListForExportGroupsByCategorySortOrderAndSeparatesArchived(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}
	ctx := context.Background()

	// Created in reverse of the intended display order, so the test
	// actually exercises sort_order rather than insertion order.
	printers, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create(Printers): %v", err)
	}
	it, err := catStore.Create("IT")
	if err != nil {
		t.Fatalf("Create(IT): %v", err)
	}
	if err := catStore.MoveUp(it.ID); err != nil {
		t.Fatalf("MoveUp(IT): %v", err)
	}

	if _, err := catStore.Create("Empty"); err != nil {
		t.Fatalf("Create(Empty): %v", err)
	}

	if _, err := store.Publish(ctx, ArticleInput{CategoryID: it.ID, Title: "VPN access", BodyHTML: "<p>Connect here.</p>"}, personID); err != nil {
		t.Fatalf("Publish(VPN access): %v", err)
	}
	if _, err := store.Publish(ctx, ArticleInput{CategoryID: printers.ID, Title: "Changing the toner", BodyHTML: "<p>Pull it out.</p>"}, personID); err != nil {
		t.Fatalf("Publish(Changing the toner): %v", err)
	}
	toArchive, err := store.Publish(ctx, ArticleInput{CategoryID: printers.ID, Title: "Old policy", BodyHTML: "<p>Retired.</p>"}, personID)
	if err != nil {
		t.Fatalf("Publish(Old policy): %v", err)
	}
	if err := store.Archive(ctx, toArchive.ID, personID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	categories, archived, err := store.ListForExport(ctx)
	if err != nil {
		t.Fatalf("ListForExport: %v", err)
	}

	if len(categories) != 2 {
		t.Fatalf("len(categories) = %d, want 2 (Empty has no published articles)", len(categories))
	}
	if categories[0].Name != "IT" || categories[1].Name != "Printers" {
		t.Fatalf("category order = [%s, %s], want [IT, Printers] (IT was moved up)", categories[0].Name, categories[1].Name)
	}
	if len(categories[0].Articles) != 1 || categories[0].Articles[0].Title != "VPN access" {
		t.Errorf("IT articles = %+v, want just VPN access", categories[0].Articles)
	}
	if len(categories[1].Articles) != 1 || categories[1].Articles[0].Title != "Changing the toner" {
		t.Errorf("Printers articles = %+v, want just Changing the toner (Old policy is archived)", categories[1].Articles)
	}

	if len(archived) != 1 || archived[0].Title != "Old policy" {
		t.Fatalf("archived = %+v, want just Old policy", archived)
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "Setting up your laptop", "Setting up your laptop"},
		{"forbidden characters replaced", `Q&A: "top" tips <for> you|us? * really\slash/`, "Q&A top tips for you us really slash"},
		{"repeated whitespace collapsed", "Two   spaces\tand a tab", "Two spaces and a tab"},
		{"trailing dot trimmed", "Version 1.0.", "Version 1.0"},
		{"trailing space trimmed", "Trailing space   ", "Trailing space"},
		{"reserved name suffixed", "CON", "CON_"},
		{"reserved name case-insensitive", "com1", "com1_"},
		{"not reserved as a substring", "CONTOSO", "CONTOSO"},
		{"empty becomes Untitled", "", "Untitled"},
		{"only forbidden characters becomes Untitled", `***`, "Untitled"},
		{"only dots and spaces becomes Untitled", " . . ", "Untitled"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sanitizeFilename(c.in); got != c.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSanitizeFilenameLongNameTruncated(t *testing.T) {
	long := strings.Repeat("a", 150)
	got := sanitizeFilename(long)
	if len([]rune(got)) != maxFilenameLength {
		t.Fatalf("len(sanitizeFilename(150 a's)) = %d, want %d", len([]rune(got)), maxFilenameLength)
	}
}

func TestSanitizeFilenameTruncationDoesNotLeaveTrailingDotOrSpace(t *testing.T) {
	// The 100-rune cutoff lands exactly on a space (index 99), which
	// truncation alone would leave as a trailing space.
	long := strings.Repeat("a", 99) + " " + strings.Repeat("b", 20)
	got := sanitizeFilename(long)
	if strings.HasSuffix(got, " ") || strings.HasSuffix(got, ".") {
		t.Fatalf("sanitizeFilename(...) = %q, ends with a trailing space or dot", got)
	}
	if got != strings.Repeat("a", 99) {
		t.Fatalf("sanitizeFilename(...) = %q, want %q", got, strings.Repeat("a", 99))
	}
}

func TestFilenameAllocatorDeduplicates(t *testing.T) {
	a := newFilenameAllocator()
	if got := a.next("Setup"); got != "Setup" {
		t.Errorf("first allocation = %q, want %q", got, "Setup")
	}
	if got := a.next("Setup"); got != "Setup (2)" {
		t.Errorf("second allocation = %q, want %q", got, "Setup (2)")
	}
	if got := a.next("Setup"); got != "Setup (3)" {
		t.Errorf("third allocation = %q, want %q", got, "Setup (3)")
	}
	// Case-insensitive, matching Windows' own filesystem.
	if got := a.next("SETUP"); got != "SETUP (4)" {
		t.Errorf("case-insensitive collision = %q, want %q", got, "SETUP (4)")
	}
	if got := a.next("Different"); got != "Different" {
		t.Errorf("unrelated name = %q, want %q", got, "Different")
	}
}

func TestBuildExportGroupsByCategoryAndOmitsEmptyOnes(t *testing.T) {
	categories := []ExportCategory{
		{Name: "Empty category"},
		{Name: "IT", Articles: []ExportArticle{
			{Title: "Setting up your laptop", BodyHTML: "<p>Step one.</p>"},
			{Title: "VPN access", BodyHTML: "<p>Connect here.</p>"},
		}},
	}
	archived := []ExportArticle{
		{Title: "Old policy", BodyHTML: "<p>No longer used.</p>"},
	}

	files, imageSrcs := BuildExport(categories, archived)

	if len(imageSrcs) != 0 {
		t.Fatalf("imageSrcs = %v, want none (no articles reference an image)", imageSrcs)
	}

	paths := map[string]string{}
	for _, f := range files {
		paths[f.Path] = string(f.Content)
	}

	if _, ok := paths["articles/IT/Setting up your laptop.html"]; !ok {
		t.Errorf("missing articles/IT/Setting up your laptop.html; got paths %v", pathList(files))
	}
	if _, ok := paths["articles/IT/VPN access.html"]; !ok {
		t.Errorf("missing articles/IT/VPN access.html; got paths %v", pathList(files))
	}
	if _, ok := paths["articles/_Archived/Old policy.html"]; !ok {
		t.Errorf("missing articles/_Archived/Old policy.html; got paths %v", pathList(files))
	}
	for path := range paths {
		if strings.HasPrefix(path, "articles/Empty category/") {
			t.Errorf("empty category should be omitted entirely, but found %q", path)
		}
	}

	index, ok := paths["index.html"]
	if !ok {
		t.Fatal("index.html missing")
	}
	if !strings.Contains(index, `href="articles/IT/Setting up your laptop.html"`) {
		t.Error("index.html doesn't link to the IT article")
	}
	if !strings.Contains(index, "Archived") {
		t.Error("index.html doesn't mark the Archived section")
	}
	if strings.Contains(index, "Empty category") {
		t.Error("index.html mentions the empty category, which has nothing to list")
	}
}

func TestBuildExportDuplicateTitlesInSameCategoryGetSuffixes(t *testing.T) {
	categories := []ExportCategory{
		{Name: "IT", Articles: []ExportArticle{
			{Title: "Setup", BodyHTML: "<p>First.</p>"},
			{Title: "Setup", BodyHTML: "<p>Second.</p>"},
		}},
	}

	files, _ := BuildExport(categories, nil)
	var paths []string
	for _, f := range files {
		if f.Path != "index.html" {
			paths = append(paths, f.Path)
		}
	}

	want := []string{"articles/IT/Setup.html", "articles/IT/Setup (2).html"}
	for _, w := range want {
		found := false
		for _, p := range paths {
			if p == w {
				found = true
			}
		}
		if !found {
			t.Errorf("missing expected path %q; got %v", w, paths)
		}
	}
}

func TestBuildExportRewritesImageSrcsRelativeToArticleAndCollectsThem(t *testing.T) {
	categories := []ExportCategory{
		{Name: "IT", Articles: []ExportArticle{
			{Title: "With a picture", BodyHTML: `<p>See:</p><img src="/images/abc123.png" alt="a screenshot">`},
		}},
	}

	files, imageSrcs := BuildExport(categories, nil)

	if len(imageSrcs) != 1 || imageSrcs[0] != "/images/abc123.png" {
		t.Fatalf("imageSrcs = %v, want [/images/abc123.png]", imageSrcs)
	}

	var articleHTML string
	for _, f := range files {
		if f.Path == "articles/IT/With a picture.html" {
			articleHTML = string(f.Content)
		}
	}
	if articleHTML == "" {
		t.Fatal("article file not found")
	}
	if !strings.Contains(articleHTML, `src="../../images/abc123.png"`) {
		t.Errorf("article HTML doesn't reference the image relatively: %s", articleHTML)
	}
}

func pathList(files []ExportFile) []string {
	var out []string
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}
