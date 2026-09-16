// Package comphq embeds the files shared by every section: HTML templates
// and SQL migrations (SPEC B1 — "All templates, CSS, JS, fonts and
// migrations embedded with go:embed"). The embed directives must live at
// the repository root because go:embed patterns can't reach outside the
// directory of the file that declares them.
package comphq

import "embed"

//go:embed all:web/templates
var Templates embed.FS

//go:embed all:migrations
var Migrations embed.FS
