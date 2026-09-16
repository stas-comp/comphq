CREATE TABLE kb_article_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL REFERENCES kb_articles(id),
    version_no INTEGER NOT NULL,
    title TEXT NOT NULL,
    category_id INTEGER NOT NULL REFERENCES kb_categories(id),
    body_html TEXT NOT NULL,
    action TEXT NOT NULL,
    restored_from_version INTEGER,
    edited_by INTEGER REFERENCES people(id),
    edited_at TEXT NOT NULL
);
