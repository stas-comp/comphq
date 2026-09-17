package kb

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	kbhtml "github.com/stas-comp/comphq/internal/kb/html"
)

// ExportArticle is one article ready to render into the "Export
// everything" zip (SPEC B5, gate 1.34).
type ExportArticle struct {
	Title    string
	BodyHTML string
}

// ExportCategory groups a category's published articles for export.
type ExportCategory struct {
	Name     string
	Articles []ExportArticle
}

// ListForExport returns every published article grouped by category (in
// the categories' own sort order), plus every archived article
// regardless of its original category — SPEC B5 puts archived articles
// in one flat `articles/_Archived/` folder rather than nested under
// their category. Categories with no published articles are omitted.
func (s *ArticleStore) ListForExport(ctx context.Context) (categories []ExportCategory, archived []ExportArticle, err error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.name, a.title, a.body_html
		FROM kb_articles a
		JOIN kb_categories c ON c.id = a.category_id
		WHERE a.status = 'published'
		ORDER BY c.sort_order, a.title
	`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	indexByName := map[string]int{}
	for rows.Next() {
		var catName, title, body string
		if err := rows.Scan(&catName, &title, &body); err != nil {
			return nil, nil, err
		}
		idx, ok := indexByName[catName]
		if !ok {
			idx = len(categories)
			categories = append(categories, ExportCategory{Name: catName})
			indexByName[catName] = idx
		}
		categories[idx].Articles = append(categories[idx].Articles, ExportArticle{Title: title, BodyHTML: body})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	archivedRows, err := s.DB.QueryContext(ctx, `SELECT title, body_html FROM kb_articles WHERE status = 'archived' ORDER BY title`)
	if err != nil {
		return nil, nil, err
	}
	defer archivedRows.Close()

	for archivedRows.Next() {
		var title, body string
		if err := archivedRows.Scan(&title, &body); err != nil {
			return nil, nil, err
		}
		archived = append(archived, ExportArticle{Title: title, BodyHTML: body})
	}
	if err := archivedRows.Err(); err != nil {
		return nil, nil, err
	}

	return categories, archived, nil
}

// ExportFile is one HTML file for the export zip: its path (forward
// slashes, relative to the zip root) and rendered content.
type ExportFile struct {
	Path    string
	Content []byte
}

// BuildExport renders index.html and one self-contained page per article
// (SPEC B5/gate 1.34: "unpacked on a computer with no network, index.html
// opens a contents page... every article opens with its formatting and
// images"). It returns every HTML file to add to the zip, plus every
// original /images/... src referenced by at least one article — this
// package knows nothing about the image store or the filesystem, so the
// caller uses that list to copy the right files into the zip's images/
// folder.
func BuildExport(categories []ExportCategory, archived []ExportArticle) (files []ExportFile, imageSrcs []string) {
	dirNames := newFilenameAllocator()
	seenImages := map[string]bool{}
	addImages := func(srcs []string) {
		for _, src := range srcs {
			if !seenImages[src] {
				seenImages[src] = true
				imageSrcs = append(imageSrcs, src)
			}
		}
	}

	var sections []indexSection

	for _, cat := range categories {
		if len(cat.Articles) == 0 {
			continue
		}
		dir := "articles/" + dirNames.next(sanitizeFilename(cat.Name))
		section := indexSection{Name: cat.Name}
		fileNames := newFilenameAllocator()
		for _, a := range cat.Articles {
			path := dir + "/" + fileNames.next(sanitizeFilename(a.Title)) + ".html"
			page, srcs := renderArticlePage(a.Title, a.BodyHTML)
			files = append(files, ExportFile{Path: path, Content: []byte(page)})
			addImages(srcs)
			section.Links = append(section.Links, indexLink{Title: a.Title, Path: path})
		}
		sections = append(sections, section)
	}

	if len(archived) > 0 {
		dir := "articles/" + dirNames.next("_Archived")
		section := indexSection{Name: "Archived", Archived: true}
		fileNames := newFilenameAllocator()
		for _, a := range archived {
			path := dir + "/" + fileNames.next(sanitizeFilename(a.Title)) + ".html"
			page, srcs := renderArticlePage(a.Title, a.BodyHTML)
			files = append(files, ExportFile{Path: path, Content: []byte(page)})
			addImages(srcs)
			section.Links = append(section.Links, indexLink{Title: a.Title, Path: path})
		}
		sections = append(sections, section)
	}

	files = append(files, ExportFile{Path: "index.html", Content: []byte(renderIndexPage(sections))})
	return files, imageSrcs
}

type indexSection struct {
	Name     string
	Archived bool
	Links    []indexLink
}

type indexLink struct {
	Title string
	Path  string
}

// exportCSS is inlined into every generated page (SPEC B5: "self-
// contained... with inline CSS"), so the export reads well with no
// network and no dependency on the app's own stylesheets. It targets
// bare tags, since Sanitize's allowlist never keeps a class or id
// attribute.
const exportCSS = `body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; color: #1b1f24; max-width: 40rem; margin: 2rem auto; padding: 0 1rem; line-height: 1.5; }
h1 { font-size: 1.75rem; }
h2 { font-size: 1.375rem; margin-top: 1.5em; }
h3 { font-size: 1.125rem; margin-top: 1.5em; }
a { color: #c2410c; }
img { max-width: 100%; height: auto; }
table { border-collapse: collapse; width: 100%; margin: 1em 0; }
td, th { border: 1px solid #ccc; padding: 0.4em 0.6em; text-align: left; vertical-align: top; }
th { background: #f3f3f3; }
div[data-missing-kind] { border: 2px dashed #a4222c; color: #a4222c; padding: 0.75rem; text-align: center; margin: 1em 0; }
blockquote { margin: 1em 0; padding-left: 1em; border-left: 3px solid #ccc; color: #444; }`

const exportPageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>%s</title>
<style>%s</style>
</head>
<body>
%s
</body>
</html>
`

// renderArticlePage wraps bodyHTML into a self-contained page, rewriting
// its image srcs from the live /images/<hash>.<ext> route to a path
// relative to the article file two directories below the zip root
// (articles/<Category-or-_Archived>/<Title>.html -> ../../images/...).
func renderArticlePage(title, bodyHTML string) (page string, imageSrcs []string) {
	rewritten, srcs := kbhtml.RewriteImageSrcs(bodyHTML, func(src string) string {
		return "../../" + strings.TrimPrefix(src, "/")
	})
	escapedTitle := html.EscapeString(title)
	body := "<h1>" + escapedTitle + "</h1>\n" + rewritten
	return fmt.Sprintf(exportPageTemplate, escapedTitle, exportCSS, body), srcs
}

func renderIndexPage(sections []indexSection) string {
	var body strings.Builder
	body.WriteString("<h1>Comp HQ export</h1>\n")
	if len(sections) == 0 {
		body.WriteString("<p>There are no articles yet.</p>\n")
	}
	for _, s := range sections {
		heading := html.EscapeString(s.Name)
		if s.Archived {
			heading += " (archived)"
		}
		fmt.Fprintf(&body, "<h2>%s</h2>\n<ul>\n", heading)
		for _, link := range s.Links {
			fmt.Fprintf(&body, "<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(link.Path), html.EscapeString(link.Title))
		}
		body.WriteString("</ul>\n")
	}
	return fmt.Sprintf(exportPageTemplate, "Comp HQ export", exportCSS, body.String())
}

var filenameForbiddenChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

const maxFilenameLength = 100

// sanitizeFilename turns name into a name safe to use as a Windows file
// or directory name (SPEC B5: "filenames are made safe for Windows"):
// forbidden characters replaced, repeated whitespace collapsed, trailing
// dots/spaces trimmed (both illegal at the end of a Windows name),
// reserved device names suffixed, and length capped. Never empty.
func sanitizeFilename(name string) string {
	cleaned := filenameForbiddenChars.ReplaceAllString(name, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	cleaned = strings.TrimRight(cleaned, " .")
	if cleaned == "" {
		cleaned = "Untitled"
	}

	if runes := []rune(cleaned); len(runes) > maxFilenameLength {
		cleaned = strings.TrimRight(string(runes[:maxFilenameLength]), " .")
		if cleaned == "" {
			cleaned = "Untitled"
		}
	}

	if windowsReservedNames[strings.ToUpper(cleaned)] {
		cleaned += "_"
	}
	return cleaned
}

// filenameAllocator hands out unique names within one directory (SPEC
// B5: "duplicate suffixes"), matching Windows' own case-insensitive
// filesystem when deciding whether two names collide.
type filenameAllocator struct {
	used map[string]int
}

func newFilenameAllocator() *filenameAllocator {
	return &filenameAllocator{used: map[string]int{}}
}

func (a *filenameAllocator) next(name string) string {
	key := strings.ToLower(name)
	count := a.used[key]
	a.used[key] = count + 1
	if count == 0 {
		return name
	}
	return fmt.Sprintf("%s (%d)", name, count+1)
}
