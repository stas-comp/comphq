CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT 'M',
    stage TEXT NOT NULL DEFAULT 'idea',
    position INTEGER NOT NULL,
    due_date TEXT,
    done_at TEXT,
    removed_at TEXT,
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    updated_by INTEGER NOT NULL REFERENCES people(id),
    updated_at TEXT NOT NULL
);

-- Full shape from SPEC B3, created now (not a minimal stub) the same way
-- kb_categories/kb_articles were at P1-17: position is unique per stage
-- among visible (non-removed) tasks, but that's an invariant the move/
-- create code maintains, not a DB constraint (kb_categories.sort_order
-- follows the same pattern).
CREATE TABLE task_assignees (
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    person_id INTEGER NOT NULL REFERENCES people(id),
    PRIMARY KEY (task_id, person_id)
);

CREATE TABLE task_activity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    person_id INTEGER NOT NULL REFERENCES people(id),
    action TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    at TEXT NOT NULL
);
