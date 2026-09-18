CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL,
    end_date TEXT,
    start_time TEXT,
    end_time TEXT,
    recurrence TEXT NOT NULL DEFAULT 'none',
    until_date TEXT,
    notice_days INTEGER NOT NULL DEFAULT 7,
    removed_at TEXT,
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    updated_by INTEGER NOT NULL REFERENCES people(id),
    updated_at TEXT NOT NULL
);

-- Keyed by (event_id, original_date) — SPEC B4's "exceptions are keyed
-- by the original date" — not an id of its own, since an occurrence is
-- only ever addressed by which series it belongs to and which of that
-- series' own dates it replaces.
CREATE TABLE event_exceptions (
    event_id INTEGER NOT NULL REFERENCES events(id),
    original_date TEXT NOT NULL,
    kind TEXT NOT NULL,
    new_start_date TEXT,
    new_end_date TEXT,
    new_start_time TEXT,
    new_end_time TEXT,
    updated_by INTEGER NOT NULL REFERENCES people(id),
    updated_at TEXT NOT NULL,
    PRIMARY KEY (event_id, original_date)
);
