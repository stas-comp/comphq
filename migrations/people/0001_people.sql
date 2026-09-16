CREATE TABLE people (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL
);

-- Names are unique among active people only (SPEC B3): a removed name can
-- be reused. Case-insensitive, since "Sam" and "sam" are the same person.
CREATE UNIQUE INDEX people_active_name_unique ON people (LOWER(name)) WHERE active = 1;
