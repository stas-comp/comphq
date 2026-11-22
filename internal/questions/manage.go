package questions

import (
	"context"
	"database/sql"
	"html/template"
	"sort"
	"strconv"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/app/wordmatch"
)

// Editing, removing, restoring and searching (SPEC B14.6, gates 8.75, 8.76, 8.78).

// EditQuestion changes the words of a question (gate 8.75). Anyone can; it then
// shows "edited by Sam, Sat 21 Nov". The limits are those of asking, and a
// refusal changes nothing. The answered state is untouched.
func (s *Store) EditQuestion(ctx context.Context, id int64, text, details string, by int64, now time.Time) error {
	text, details = cleanText(text), cleanText(details)
	if text == "" {
		return ErrEmptyQuestion
	}
	if chars(text) > MaxQuestionChars {
		return ErrQuestionTooLong
	}
	if chars(details) > MaxDetailsChars {
		return ErrDetailsTooLong
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := live(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE questions SET text = ?, details = ?, edited_by = ?, edited_at = ? WHERE id = ?`, text, details, by, stamp(now), id); err != nil {
		return err
	}
	if err := bump(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// EditAnswer changes the words of an answer (gate 8.75). The answer stays its
// writer's: it shows "edited by Jane, Sat 21 Nov" beside their name.
func (s *Store) EditAnswer(ctx context.Context, questionID, answerID int64, text string, by int64, now time.Time) error {
	text = cleanText(text)
	if text == "" {
		return ErrEmptyAnswer
	}
	if chars(text) > MaxAnswerChars {
		return ErrAnswerTooLong
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := live(ctx, tx, questionID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE question_answers SET text = ?, edited_by = ?, edited_at = ? WHERE id = ? AND question_id = ? AND removed_at IS NULL`,
		text, by, stamp(now), answerID, questionID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAnswerNotFound
	}
	if err := bump(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveAnswer takes an answer off its question (gate 8.76) and returns its words,
// for the Undo message. It never deletes: removed_by and removed_at are set (A3).
// Removing one that is already removed is not an error.
func (s *Store) RemoveAnswer(ctx context.Context, questionID, answerID, by int64, now time.Time) (string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var text string
	var removed sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT text, removed_at FROM question_answers WHERE id = ? AND question_id = ?`, answerID, questionID).Scan(&text, &removed)
	if err == sql.ErrNoRows {
		return "", ErrAnswerNotFound
	}
	if err != nil {
		return "", err
	}
	if removed.Valid {
		return text, nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE question_answers SET removed_by = ?, removed_at = ? WHERE id = ?`, by, stamp(now), answerID); err != nil {
		return "", err
	}
	if err := bump(ctx, tx); err != nil {
		return "", err
	}
	return text, tx.Commit()
}

// RestoreAnswer brings a removed answer back (Undo, gate 8.76), in its place by
// when it was written. Restoring one that isn't removed is not an error.
func (s *Store) RestoreAnswer(ctx context.Context, questionID, answerID, by int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE question_answers SET removed_by = NULL, removed_at = NULL WHERE id = ? AND question_id = ? AND removed_at IS NOT NULL`, answerID, questionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var one int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM question_answers WHERE id = ? AND question_id = ?`, answerID, questionID).Scan(&one); err == sql.ErrNoRows {
			return ErrAnswerNotFound
		}
		return tx.Commit()
	}
	if err := bump(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveQuestion moves a question (and with it its answers, which stay with it) to
// Removed questions (gate 8.76). Nothing is deleted. Removing one that is already
// removed is not an error.
func (s *Store) RemoveQuestion(ctx context.Context, id, by int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var removed sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT removed_at FROM questions WHERE id = ?`, id).Scan(&removed)
	if err == sql.ErrNoRows {
		return ErrQuestionNotFound
	}
	if err != nil {
		return err
	}
	if removed.Valid {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE questions SET removed_by = ?, removed_at = ? WHERE id = ?`, by, stamp(now), id); err != nil {
		return err
	}
	if err := bump(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RestoreQuestion brings a removed question back, with its answers and its state
// (gate 8.76). Restoring one that isn't removed is not an error.
func (s *Store) RestoreQuestion(ctx context.Context, id, by int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var removed sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT removed_at FROM questions WHERE id = ?`, id).Scan(&removed)
	if err == sql.ErrNoRows {
		return ErrQuestionNotFound
	}
	if err != nil {
		return err
	}
	if !removed.Valid {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE questions SET removed_by = NULL, removed_at = NULL WHERE id = ?`, id); err != nil {
		return err
	}
	if err := bump(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RemovedRow is one line of Removed questions.
type RemovedRow struct {
	Row
	RemovedBy    string
	RemovedLabel string
}

// ListRemoved is every removed question, most recently removed first.
func (s *Store) ListRemoved(ctx context.Context, today time.Time) ([]RemovedRow, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT q.id, q.text, p.name, q.asked_at, rb.name, q.removed_at,
		       (SELECT COUNT(*) FROM question_answers a WHERE a.question_id = q.id AND a.removed_at IS NULL)
		FROM questions q
		JOIN people p ON p.id = q.asked_by
		JOIN people rb ON rb.id = q.removed_by
		WHERE q.removed_at IS NOT NULL
		ORDER BY q.removed_at DESC, q.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RemovedRow
	for rows.Next() {
		var r RemovedRow
		var askedAt, removedAt string
		if err := rows.Scan(&r.ID, &r.Text, &r.AskedBy, &askedAt, &r.RemovedBy, &removedAt, &r.AnswerCount); err != nil {
			return nil, err
		}
		r.AskedLabel = label(askedAt, today)
		r.RemovedLabel = label(removedAt, today)
		out = append(out, r)
	}
	return out, rows.Err()
}

// SearchLimit is how many results a search returns (as job search, gate 7.21).
const SearchLimit = 20

// Hit is one search result: the question with the matched words marked (Title and
// Passage are HTML-escaped, with <mark>), and enough to say who asked and when.
type Hit struct {
	ID          int64
	URL         string
	Text        string
	Title       template.HTML
	Passage     template.HTML
	AskedBy     string
	AskedLabel  string
	AnswerCount int
	Answered    bool

	inText  bool
	askedAt string
}

// Search finds questions for the words typed (gate 8.78, D-94): every word has to
// appear in the question, its details or its live answers, case and accents
// folded, the last word possibly half-typed. Waiting questions come first, then
// those with the words in the question itself before those with them only in the
// details or an answer, newest first. Removed questions are never found. The match
// is done in Go over one pass (as job search is), which a 1,000-question library
// answers well inside a second.
func (s *Store) Search(ctx context.Context, query string, today time.Time) ([]Hit, error) {
	words := wordmatch.Words(query)
	if len(words) == 0 {
		return []Hit{}, nil
	}
	lowered := make([][]rune, len(words))
	for i, w := range words {
		lowered[i] = wordmatch.Lower(w)
	}

	answerTexts := map[int64][]string{}
	arows, err := s.DB.QueryContext(ctx, `
		SELECT a.question_id, a.text FROM question_answers a
		JOIN questions q ON q.id = a.question_id
		WHERE a.removed_at IS NULL AND q.removed_at IS NULL
		ORDER BY a.question_id, a.created_at, a.id`)
	if err != nil {
		return nil, err
	}
	for arows.Next() {
		var qid int64
		var text string
		if err := arows.Scan(&qid, &text); err != nil {
			arows.Close()
			return nil, err
		}
		answerTexts[qid] = append(answerTexts[qid], text)
	}
	arows.Close()
	if err := arows.Err(); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT q.id, q.text, q.details, p.name, q.asked_at, q.answered_at IS NOT NULL
		FROM questions q JOIN people p ON p.id = q.asked_by
		WHERE q.removed_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []Hit
	for rows.Next() {
		var h Hit
		var details, askedAt string
		if err := rows.Scan(&h.ID, &h.Text, &details, &h.AskedBy, &askedAt, &h.Answered); err != nil {
			return nil, err
		}
		textLow, detailsLow := wordmatch.Lower(h.Text), wordmatch.Lower(details)
		answersLow := make([][]rune, len(answerTexts[h.ID]))
		for i, a := range answerTexts[h.ID] {
			answersLow[i] = wordmatch.Lower(a)
		}
		all, inText := true, true
		for _, w := range lowered {
			t := wordmatch.Index(textLow, w) >= 0
			found := t || wordmatch.Index(detailsLow, w) >= 0
			for _, a := range answersLow {
				if found {
					break
				}
				found = wordmatch.Index(a, w) >= 0
			}
			if !found {
				all = false
				break
			}
			if !t {
				inText = false
			}
		}
		if !all {
			continue
		}
		h.inText, h.askedAt = inText, askedAt
		h.URL = "/questions/" + strconv.FormatInt(h.ID, 10)
		h.Title = wordmatch.Highlight(h.Text, lowered)
		// The passage is from the details if the words are there, else from the first
		// answer that has one.
		h.Passage = wordmatch.Passage(details, lowered)
		if h.Passage == "" {
			for _, a := range answerTexts[h.ID] {
				if p := wordmatch.Passage(a, lowered); p != "" {
					h.Passage = p
					break
				}
			}
		}
		h.AnswerCount = len(answerTexts[h.ID])
		h.AskedLabel = label(askedAt, today)
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.Answered != b.Answered {
			return !a.Answered
		}
		if a.inText != b.inText {
			return a.inText
		}
		if a.askedAt != b.askedAt {
			return a.askedAt > b.askedAt
		}
		return a.ID > b.ID
	})
	if len(hits) > SearchLimit {
		hits = hits[:SearchLimit]
	}
	return hits, nil
}

// QuestionsCSV is the export's questions.csv (gate 8.81): every question, removed
// ones included, with who asked, answered, edited and removed it, and when.
type QuestionsCSV struct{ DB *sql.DB }

func exportStamp(s sql.NullString) string {
	if !s.Valid || s.String == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, s.String); err == nil {
		return format.DateTime(t)
	}
	return s.String
}

// ExportRows returns a header row in plain English, then one row per question in
// the order they were asked.
func (c QuestionsCSV) ExportRows(ctx context.Context) ([][]string, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT q.text, q.details, p.name, q.asked_at, COALESCE(ab.name, ''), q.answered_at, COALESCE(eb.name, ''), q.edited_at, COALESCE(rb.name, ''), q.removed_at
		FROM questions q
		JOIN people p ON p.id = q.asked_by
		LEFT JOIN people ab ON ab.id = q.answered_by
		LEFT JOIN people eb ON eb.id = q.edited_by
		LEFT JOIN people rb ON rb.id = q.removed_by
		ORDER BY q.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][]string{{"Question", "Details", "Asked by", "Asked", "Answered by", "Answered", "Edited by", "Edited", "Removed by", "Removed"}}
	for rows.Next() {
		var text, details, asker, askedAt, answerer, editor, remover string
		var answeredAt, editedAt, removedAt sql.NullString
		if err := rows.Scan(&text, &details, &asker, &askedAt, &answerer, &answeredAt, &editor, &editedAt, &remover, &removedAt); err != nil {
			return nil, err
		}
		out = append(out, []string{text, details, asker, exportStamp(sql.NullString{String: askedAt, Valid: true}), answerer, exportStamp(answeredAt), editor, exportStamp(editedAt), remover, exportStamp(removedAt)})
	}
	return out, rows.Err()
}

// AnswersCSV is the export's answers.csv (gate 8.81): every answer, removed ones
// included, under the question it belongs to.
type AnswersCSV struct{ DB *sql.DB }

// ExportRows returns a header row in plain English, then one row per answer in the
// order they were written.
func (c AnswersCSV) ExportRows(ctx context.Context) ([][]string, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT q.text, a.text, p.name, a.created_at, COALESCE(eb.name, ''), a.edited_at, COALESCE(rb.name, ''), a.removed_at
		FROM question_answers a
		JOIN questions q ON q.id = a.question_id
		JOIN people p ON p.id = a.created_by
		LEFT JOIN people eb ON eb.id = a.edited_by
		LEFT JOIN people rb ON rb.id = a.removed_by
		ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][]string{{"Question", "Answer", "Answered by", "Answered", "Edited by", "Edited", "Removed by", "Removed"}}
	for rows.Next() {
		var question, text, writer, createdAt, editor, remover string
		var editedAt, removedAt sql.NullString
		if err := rows.Scan(&question, &text, &writer, &createdAt, &editor, &editedAt, &remover, &removedAt); err != nil {
			return nil, err
		}
		out = append(out, []string{question, text, writer, exportStamp(sql.NullString{String: createdAt, Valid: true}), editor, exportStamp(editedAt), remover, exportStamp(removedAt)})
	}
	return out, rows.Err()
}

// WaitingOldest is up to limit waiting questions, oldest first, with how many are
// waiting in all: the Briefing's "Questions waiting for an answer" (gate 8.77).
func (s *Store) WaitingOldest(ctx context.Context, limit int, today time.Time) ([]Row, int, error) {
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM questions WHERE removed_at IS NULL AND answered_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, rowSelect+`
		WHERE q.removed_at IS NULL AND q.answered_at IS NULL
		ORDER BY q.asked_at ASC, q.id ASC LIMIT ?`, limit)
	if err != nil {
		return nil, 0, err
	}
	out, err := scanRows(rows, today)
	return out, total, err
}

// AnswerRemoved says whether an answer of a question has been removed, for the Undo
// message after removing one.
func (s *Store) AnswerRemoved(ctx context.Context, questionID, answerID int64) (bool, error) {
	var removed sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT removed_at FROM question_answers WHERE id = ? AND question_id = ?`, answerID, questionID).Scan(&removed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return removed.Valid, err
}
