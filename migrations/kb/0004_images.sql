CREATE TABLE images (
    sha256 TEXT PRIMARY KEY,
    ext TEXT NOT NULL,
    mime TEXT NOT NULL,
    bytes INTEGER NOT NULL,
    created_by INTEGER REFERENCES people(id),
    created_at TEXT NOT NULL
);
