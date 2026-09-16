CREATE TABLE kb_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Full shape from SPEC B3, created now (not just a minimal stub) so the
-- category delete-refusal check (SPEC gate 1.12) and the KB home page's
-- "recently updated" query (gate 1.13) are both real queries against a
-- real table from the start. P1-19 starts writing rows into it; P1-18's
-- sanitiser/block-id work and P1-19's kb_article_versions table build on
-- top of this via their own expand-only migrations.
CREATE TABLE kb_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL REFERENCES kb_categories(id),
    title TEXT NOT NULL,
    body_html TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'published',
    version_no INTEGER NOT NULL DEFAULT 1,
    created_by INTEGER REFERENCES people(id),
    created_at TEXT NOT NULL,
    updated_by INTEGER REFERENCES people(id),
    updated_at TEXT NOT NULL
);
