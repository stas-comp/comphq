-- Questions (SPEC B14.6, D-110; v1.6). A new section's own tables, add-only, so a
-- rollback to 1.5.0 leaves every row untouched (1.5.0 doesn't know the section
-- exists) and upgrading again finds them all (gate 8.82).
--
-- A question is plain text (no editor). It is "waiting" while answered_at is NULL;
-- anyone can mark it answered or put it back, and the row remembers who and when.
-- Nothing is ever deleted (A3): removing a question or an answer sets removed_by
-- and removed_at, and "Removed questions" can restore a question. The length limits
-- (300 / 2,000 / 2,000 characters) are kept by the store, not by constraints.
CREATE TABLE questions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    asked_by INTEGER NOT NULL REFERENCES people(id),
    asked_at TEXT NOT NULL,
    answered_by INTEGER REFERENCES people(id),
    answered_at TEXT,
    edited_by INTEGER REFERENCES people(id),
    edited_at TEXT,
    removed_by INTEGER REFERENCES people(id),
    removed_at TEXT
);

-- The list's two groups: waiting (answered_at NULL) newest first, answered by when.
CREATE INDEX questions_state ON questions (answered_at, asked_at) WHERE removed_at IS NULL;

CREATE TABLE question_answers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id INTEGER NOT NULL REFERENCES questions(id),
    text TEXT NOT NULL,
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    edited_by INTEGER REFERENCES people(id),
    edited_at TEXT,
    removed_by INTEGER REFERENCES people(id),
    removed_at TEXT
);

CREATE INDEX question_answers_question ON question_answers (question_id);
