// Package questions is the Questions section (SPEC A8a, B14.6, D-110): anyone asks,
// anyone answers, anyone marks a question answered. There are no roles (A3), no
// notifications, and no editor: a question and an answer are plain text. It depends
// only on people, db and app (SPEC B2).
package questions

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stas-comp/comphq/internal/app/format"
)

// The limits (gate 8.71, 8.73), kept here so no route can get round them.
const (
	MaxQuestionChars = 300
	MaxDetailsChars  = 2000
	MaxAnswerChars   = 2000
	// AnsweredShown is how many answered questions the list shows before "Show all"
	// (gate 8.72).
	AnsweredShown = 50
)

// Refusals, in the words the pages show; nothing changes when one is returned.
var (
	ErrEmptyQuestion    = errors.New("Type your question first.")
	ErrQuestionTooLong  = errors.New("A question can be up to 300 characters.")
	ErrDetailsTooLong   = errors.New("The details can be up to 2,000 characters.")
	ErrEmptyAnswer      = errors.New("Type your answer first.")
	ErrAnswerTooLong    = errors.New("An answer can be up to 2,000 characters.")
	ErrQuestionNotFound = errors.New("That question isn't there any more.")
	ErrAnswerNotFound   = errors.New("That answer isn't there any more.")
)

// Store reads and writes questions and their answers.
type Store struct {
	DB *sql.DB
}

const questionsVersionKey = "questions_version"

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// bump moves the section's change counter, inside the write's transaction, so it
// only advances if the change committed (the refresh design of SPEC B4).
func bump(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO app_meta (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, questionsVersionKey)
	return err
}

// Version is the questions change counter, 0 before anything was ever written.
func (s *Store) Version(ctx context.Context) (int64, error) {
	var v sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, questionsVersionKey).Scan(&v)
	if err == sql.ErrNoRows || !v.Valid {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var n int64
	for _, c := range v.String {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

// cleanText trims a typed text, makes its line breaks plain "\n" and counts it in
// characters, not bytes. The text itself is kept exactly as typed otherwise: it is
// escaped when it is drawn, never altered here.
func cleanText(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	return strings.TrimSpace(raw)
}

func chars(s string) int { return utf8.RuneCountInString(s) }

// Answer is one answer, as the question's page shows it.
type Answer struct {
	ID          int64
	Text        string
	By          string
	CreatedAt   time.Time
	Label       string // "Sat 21 Nov"
	Edited      bool
	EditedBy    string
	EditedLabel string
}

// Question is a question with its live answers, as its page shows it.
type Question struct {
	ID            int64
	Text          string
	Details       string
	AskedBy       string
	AskedLabel    string
	Answered      bool
	AnsweredBy    string
	AnsweredLabel string
	Edited        bool
	EditedBy      string
	EditedLabel   string
	Removed       bool
	Answers       []Answer
}

// AnsweredText is the line under an answered question: "Answered · marked by Jane,
// Sat 21 Nov" (gate 8.74).
func (q Question) AnsweredText() string {
	return "Answered · marked by " + q.AnsweredBy + ", " + q.AnsweredLabel
}

// Row is one line of the list: the question, who asked and when, how many answers.
type Row struct {
	ID            int64
	Text          string
	AskedBy       string
	AskedLabel    string
	AnswerCount   int
	AnsweredBy    string
	AnsweredLabel string
}

// ListResult is the Questions page: Waiting for an answer (newest first), then
// Answered (most recently answered first, 50 unless Show all).
type ListResult struct {
	Waiting       []Row
	Answered      []Row
	AnsweredTotal int
	MoreAnswered  bool
}

func label(ts string, today time.Time) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return format.DateShort(t.In(today.Location()), today)
}

// Ask adds a question with who asked and when (gate 8.71). The text is required
// and up to 300 characters; the details are optional, up to 2,000, and keep their
// line breaks. Nothing is saved when either is refused.
func (s *Store) Ask(ctx context.Context, text, details string, askedBy int64, now time.Time) (int64, error) {
	text, details = cleanText(text), cleanText(details)
	if text == "" {
		return 0, ErrEmptyQuestion
	}
	if chars(text) > MaxQuestionChars {
		return 0, ErrQuestionTooLong
	}
	if chars(details) > MaxDetailsChars {
		return 0, ErrDetailsTooLong
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO questions (text, details, asked_by, asked_at) VALUES (?, ?, ?, ?)`, text, details, askedBy, stamp(now))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := bump(ctx, tx); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// Get returns a question with its live answers, oldest first (gate 8.73). A removed
// question is returned too, marked Removed, so its page can offer to restore it.
func (s *Store) Get(ctx context.Context, id int64, today time.Time) (Question, error) {
	var q Question
	var askedAt string
	var answeredAt, editedAt, removedAt, answeredBy, editedBy sql.NullString
	err := s.DB.QueryRowContext(ctx, `
		SELECT q.id, q.text, q.details, p.name, q.asked_at, ab.name, q.answered_at, eb.name, q.edited_at, q.removed_at
		FROM questions q
		JOIN people p ON p.id = q.asked_by
		LEFT JOIN people ab ON ab.id = q.answered_by
		LEFT JOIN people eb ON eb.id = q.edited_by
		WHERE q.id = ?`, id).Scan(&q.ID, &q.Text, &q.Details, &q.AskedBy, &askedAt, &answeredBy, &answeredAt, &editedBy, &editedAt, &removedAt)
	if err == sql.ErrNoRows {
		return Question{}, ErrQuestionNotFound
	}
	if err != nil {
		return Question{}, err
	}
	q.AskedLabel = label(askedAt, today)
	if answeredAt.Valid {
		q.Answered, q.AnsweredBy, q.AnsweredLabel = true, answeredBy.String, label(answeredAt.String, today)
	}
	if editedAt.Valid {
		q.Edited, q.EditedBy, q.EditedLabel = true, editedBy.String, label(editedAt.String, today)
	}
	q.Removed = removedAt.Valid

	rows, err := s.DB.QueryContext(ctx, `
		SELECT a.id, a.text, p.name, a.created_at, eb.name, a.edited_at
		FROM question_answers a
		JOIN people p ON p.id = a.created_by
		LEFT JOIN people eb ON eb.id = a.edited_by
		WHERE a.question_id = ? AND a.removed_at IS NULL
		ORDER BY a.created_at, a.id`, id)
	if err != nil {
		return Question{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Answer
		var created string
		var eBy, eAt sql.NullString
		if err := rows.Scan(&a.ID, &a.Text, &a.By, &created, &eBy, &eAt); err != nil {
			return Question{}, err
		}
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			a.CreatedAt = t
		}
		a.Label = label(created, today)
		if eAt.Valid {
			a.Edited, a.EditedBy, a.EditedLabel = true, eBy.String, label(eAt.String, today)
		}
		q.Answers = append(q.Answers, a)
	}
	return q, rows.Err()
}

const rowSelect = `
	SELECT q.id, q.text, p.name, q.asked_at, COALESCE(ab.name, ''), q.answered_at,
	       (SELECT COUNT(*) FROM question_answers a WHERE a.question_id = q.id AND a.removed_at IS NULL)
	FROM questions q
	JOIN people p ON p.id = q.asked_by
	LEFT JOIN people ab ON ab.id = q.answered_by`

func scanRows(rows *sql.Rows, today time.Time) ([]Row, error) {
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		var askedAt string
		var answeredAt sql.NullString
		if err := rows.Scan(&r.ID, &r.Text, &r.AskedBy, &askedAt, &r.AnsweredBy, &answeredAt, &r.AnswerCount); err != nil {
			return nil, err
		}
		r.AskedLabel = label(askedAt, today)
		if answeredAt.Valid {
			r.AnsweredLabel = label(answeredAt.String, today)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// List is the Questions page (gate 8.72): every waiting question, newest first, and
// the answered ones, most recently answered first, the 50 newest unless showAll.
func (s *Store) List(ctx context.Context, today time.Time, showAll bool) (ListResult, error) {
	var res ListResult
	rows, err := s.DB.QueryContext(ctx, rowSelect+`
		WHERE q.removed_at IS NULL AND q.answered_at IS NULL
		ORDER BY q.asked_at DESC, q.id DESC`)
	if err != nil {
		return res, err
	}
	if res.Waiting, err = scanRows(rows, today); err != nil {
		return res, err
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM questions WHERE removed_at IS NULL AND answered_at IS NOT NULL`).Scan(&res.AnsweredTotal); err != nil {
		return res, err
	}
	query := rowSelect + `
		WHERE q.removed_at IS NULL AND q.answered_at IS NOT NULL
		ORDER BY q.answered_at DESC, q.id DESC`
	if !showAll {
		query += " LIMIT 50"
	}
	rows, err = s.DB.QueryContext(ctx, query)
	if err != nil {
		return res, err
	}
	if res.Answered, err = scanRows(rows, today); err != nil {
		return res, err
	}
	res.MoreAnswered = !showAll && res.AnsweredTotal > len(res.Answered)
	return res, nil
}

// live checks the question exists and isn't removed, inside tx.
func live(ctx context.Context, tx *sql.Tx, id int64) error {
	var removed sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT removed_at FROM questions WHERE id = ?`, id).Scan(&removed)
	if err == sql.ErrNoRows || (err == nil && removed.Valid) {
		return ErrQuestionNotFound
	}
	return err
}

// AddAnswer adds an answer: anyone, any number of times (gate 8.73). It doesn't mark
// the question answered on its own (gate 8.74).
func (s *Store) AddAnswer(ctx context.Context, questionID int64, text string, by int64, now time.Time) (int64, error) {
	text = cleanText(text)
	if text == "" {
		return 0, ErrEmptyAnswer
	}
	if chars(text) > MaxAnswerChars {
		return 0, ErrAnswerTooLong
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := live(ctx, tx, questionID); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO question_answers (question_id, text, created_by, created_at) VALUES (?, ?, ?, ?)`, questionID, text, by, stamp(now))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := bump(ctx, tx); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// SetAnswered marks a question answered, remembering who and when (gate 8.74).
// Marking one that is already answered changes nothing: the first person's name
// and time stay.
func (s *Store) SetAnswered(ctx context.Context, id, by int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := live(ctx, tx, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE questions SET answered_by = ?, answered_at = ? WHERE id = ? AND answered_at IS NULL`, by, stamp(now), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if err := bump(ctx, tx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ClearAnswered puts a question back among the waiting (gate 8.74).
func (s *Store) ClearAnswered(ctx context.Context, id, by int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := live(ctx, tx, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE questions SET answered_by = NULL, answered_at = NULL WHERE id = ? AND answered_at IS NOT NULL`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if err := bump(ctx, tx); err != nil {
			return err
		}
	}
	return tx.Commit()
}
