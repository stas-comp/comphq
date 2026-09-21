-- Steps inside a job (SPEC B3 task_checklist_items, B10; v1.2).
--
-- Add-only, and safe to roll back across: a v1.1.0 app never selects from
-- this table, so going back leaves every row untouched and upgrading again
-- finds them all (gate 5.15).
--
-- Ticked is done_at IS NOT NULL, which carries who and when in the same
-- row. Removed is removed_at IS NOT NULL, the same soft delete the rest of
-- the app uses (SPEC A3). position is unique per task among non-removed
-- rows, but that's an invariant the store maintains, not a constraint,
-- the same way tasks.position is.
CREATE TABLE task_checklist_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    text TEXT NOT NULL,
    position INTEGER NOT NULL,
    done_by INTEGER REFERENCES people(id),
    done_at TEXT,
    removed_at TEXT,
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    updated_by INTEGER NOT NULL REFERENCES people(id),
    updated_at TEXT NOT NULL
);

CREATE INDEX task_checklist_items_task ON task_checklist_items (task_id, position);
