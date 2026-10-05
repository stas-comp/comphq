-- Linked jobs (SPEC B13.6, D-96; v1.4). Add-only: a table nothing in 1.3.0
-- reads, so a rollback leaves every row untouched and upgrading again finds
-- them all (gate 7.49).
--
-- kind 'before' means task_id must be finished before other_task_id; kind
-- 'related' is stored once, with task_id < other_task_id, and shown on both
-- jobs. A link is never deleted: removing it sets removed_at and removed_by
-- (A3). At most one live link joins any two jobs, and a job has at most 20,
-- but those are rules the store keeps inside a transaction, not constraints.
CREATE TABLE task_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    other_task_id INTEGER NOT NULL REFERENCES tasks(id),
    kind TEXT NOT NULL CHECK (kind IN ('before', 'related')),
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    removed_by INTEGER REFERENCES people(id),
    removed_at TEXT
);

CREATE INDEX task_links_task ON task_links (task_id);
CREATE INDEX task_links_other ON task_links (other_task_id);
