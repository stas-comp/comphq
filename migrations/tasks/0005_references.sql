-- Articles and COSMOS places on a job (SPEC B14.2, D-105; v1.6). Add-only:
-- two tables nothing in 1.5.0 reads, so a rollback leaves every row untouched
-- and upgrading again finds them all (gate 8.21).
--
-- task_articles.article_id has no foreign key: the Knowledge Base is another
-- section and neither imports the other (SPEC B2), so the article's title and
-- category are read through an interface wired in cmd/comphq. A link is never
-- deleted: removing it sets removed_at and removed_by (A3).
--
-- task_places keeps a folder path exactly as the store normalised it
-- (trimmed, forward slashes turned into backslashes). "At most 20 of each per
-- job", "no duplicates" and "a path must start with \\ or a drive letter" are
-- rules the store keeps inside a transaction, not constraints.
CREATE TABLE task_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    article_id INTEGER NOT NULL,
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    removed_by INTEGER REFERENCES people(id),
    removed_at TEXT
);

CREATE INDEX task_articles_task ON task_articles (task_id);
CREATE INDEX task_articles_article ON task_articles (article_id);

CREATE TABLE task_places (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    name TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL,
    position INTEGER NOT NULL,
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    removed_by INTEGER REFERENCES people(id),
    removed_at TEXT
);

CREATE INDEX task_places_task ON task_places (task_id);
