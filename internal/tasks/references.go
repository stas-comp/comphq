package tasks

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Articles and COSMOS places on a job (SPEC B14.2, D-105, gates 8.10-8.21).
//
// A job can link Knowledge Base articles and hold shared-folder paths. Tasks
// never imports the Knowledge Base (SPEC B2): the article titles it needs come
// through the small ArticleFinder interface below, which cmd/comphq fills in
// with the Knowledge Base's own implementation. A nil finder means "no
// articles", so Tasks' own tests need no Knowledge Base at all.

// ArticleRef is what Tasks needs to know about an article.
type ArticleRef struct {
	ID       int64
	Title    string
	Category string
	Archived bool
}

// ArticleFinder is implemented by the Knowledge Base (through cmd/comphq).
type ArticleFinder interface {
	// FindByTitle returns published articles whose title matches q (every word
	// found, the last possibly half-typed), best match first.
	FindByTitle(ctx context.Context, q string, limit int) []ArticleRef
	// Articles returns the articles with these ids, archived ones included.
	Articles(ctx context.Context, ids []int64) map[int64]ArticleRef
}

const (
	maxJobArticles = 20
	maxJobPlaces   = 20
	maxPlaceName   = 80
	maxPlacePath   = 400
	pickerLimit    = 10
	missingArticle = "(article not found)"
	folderPathHint = "That doesn't look like a folder path. It should start with \\\\ (like \\\\NAS\\COSMOS) or a drive letter (like S:\\COSMOS)."
)

// Refusals (gates 8.15, 8.16). Each is a plain message handlers show as it is;
// nothing changes when one is returned.
var (
	ErrPlacePath       = errors.New(folderPathHint)
	ErrPlaceNameLong   = errors.New("A place's name can be at most 80 characters.")
	ErrPlaceDuplicate  = errors.New("That folder is already on this job.")
	ErrTooManyPlaces   = errors.New("A job can have at most 20 COSMOS places.")
	ErrArticleLinked   = errors.New("That article is already linked to this job.")
	ErrTooManyArticles = errors.New("A job can have at most 20 articles.")
	ErrArticleGone     = errors.New("That article isn't there any more.")
	ErrArticleArchived = errors.New("That article has been archived, so it can't be linked.")
	ErrJobRemoved      = errors.New("That job has just been removed, so nothing can be added to it.")
	ErrReferenceGone   = errors.New("That isn't on this job any more.")
)

var uncPrefix = regexp.MustCompile(`^\\\\[^\\]+\\`)
var drivePrefix = regexp.MustCompile(`^[A-Za-z]:\\`)

// NormalisePath trims a folder path and turns forward slashes into backslashes,
// then checks it: not empty, at most 400 characters, and starting with \\server\
// or a drive letter (gate 8.15). The first return is the path to keep.
func NormalisePath(raw string) (string, error) {
	p := strings.ReplaceAll(strings.TrimSpace(raw), "/", `\`)
	p = strings.TrimSpace(p)
	if p == "" || utf8.RuneCountInString(p) > maxPlacePath {
		return "", ErrPlacePath
	}
	if !uncPrefix.MatchString(p) && !drivePrefix.MatchString(p) {
		return "", ErrPlacePath
	}
	return p, nil
}

// PlaceDisplayName is the name a place is shown under: the name it was given,
// or the last folder in its path (gate 8.13).
func PlaceDisplayName(name, folderPath string) string {
	if n := strings.TrimSpace(name); n != "" {
		return n
	}
	segs := strings.FieldsFunc(folderPath, func(r rune) bool { return r == '\\' })
	if len(segs) == 0 {
		return folderPath
	}
	return segs[len(segs)-1]
}

// ArticleLink is one linked article on a job, with what the Knowledge Base says
// about it now.
type ArticleLink struct {
	LinkID    int64
	ArticleID int64
	Title     string
	Category  string
	Archived  bool
	Missing   bool
}

// Place is one COSMOS place on a job.
type Place struct {
	ID    int64
	Name  string // as given, may be empty
	Path  string
	Label string // what to show: Name, or the last folder of Path
}

// References are a job's live articles and places.
type References struct {
	Articles []ArticleLink
	Places   []Place
}

type articleDetail struct {
	ArticleID int64  `json:"article_id"`
	Title     string `json:"title"`
}

type placeDetail struct {
	PlaceID int64  `json:"place_id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
}

func (s *Store) finderArticles(ctx context.Context, ids []int64) map[int64]ArticleRef {
	if s.Articles == nil || len(ids) == 0 {
		return map[int64]ArticleRef{}
	}
	if m := s.Articles.Articles(ctx, ids); m != nil {
		return m
	}
	return map[int64]ArticleRef{}
}

// SearchArticles is the "Link an article" picker (gate 8.11): up to ten
// published articles matching q, leaving out those already linked to taskID.
func (s *Store) SearchArticles(ctx context.Context, q string, taskID int64) ([]ArticleRef, error) {
	if s.Articles == nil {
		return []ArticleRef{}, nil
	}
	linked := map[int64]bool{}
	if taskID != 0 {
		rows, err := s.DB.QueryContext(ctx, `SELECT article_id FROM task_articles WHERE task_id = ? AND removed_at IS NULL`, taskID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			linked[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	// Ask for more than the limit, so dropping the linked ones still leaves ten.
	found := s.Articles.FindByTitle(ctx, q, pickerLimit+maxJobArticles)
	out := make([]ArticleRef, 0, pickerLimit)
	for _, a := range found {
		if a.Archived || linked[a.ID] {
			continue
		}
		out = append(out, a)
		if len(out) == pickerLimit {
			break
		}
	}
	return out, nil
}

// checkLive confirms the job exists and isn't removed, inside tx.
func checkLive(ctx context.Context, tx *sql.Tx, taskID int64) error {
	var removed sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT removed_at FROM tasks WHERE id = ?`, taskID).Scan(&removed); err != nil {
		if err == sql.ErrNoRows {
			return ErrTaskNotFound
		}
		return err
	}
	if removed.Valid {
		return ErrJobRemoved
	}
	return nil
}

// AddArticle links a published article to a job (gates 8.11, 8.16, 8.17). Every
// refusal is checked inside the write transaction, so two people racing for the
// last place cannot both get it. Linking is not an edit of the article.
func (s *Store) AddArticle(ctx context.Context, taskID, articleID, actorID int64, now time.Time) error {
	// The finder reads the Knowledge Base through its own handle, so it is asked
	// before the transaction opens (the app has a single database connection).
	ref, ok := s.finderArticles(ctx, []int64{articleID})[articleID]
	if !ok {
		return ErrArticleGone
	}
	if ref.Archived {
		return ErrArticleArchived
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := checkLive(ctx, tx, taskID); err != nil {
		return err
	}
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_articles WHERE task_id = ? AND article_id = ? AND removed_at IS NULL`,
		taskID, articleID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrArticleLinked
	}
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_articles WHERE task_id = ? AND removed_at IS NULL`, taskID).Scan(&n); err != nil {
		return err
	}
	if n >= maxJobArticles {
		return ErrTooManyArticles
	}
	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO task_articles (task_id, article_id, created_by, created_at) VALUES (?, ?, ?, ?)`,
		taskID, articleID, actorID, nowStr); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "article_linked", articleDetail{ArticleID: articleID, Title: ref.Title}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveArticle takes an article off a job (gate 8.17). It never deletes:
// removed_at and removed_by are set (A3). History keeps the title as it is now.
func (s *Store) RemoveArticle(ctx context.Context, taskID, linkID, actorID int64, now time.Time) error {
	var articleID int64
	err := s.DB.QueryRowContext(ctx, `SELECT article_id FROM task_articles WHERE id = ? AND task_id = ?`, linkID, taskID).Scan(&articleID)
	if err == sql.ErrNoRows {
		return ErrReferenceGone
	}
	if err != nil {
		return err
	}
	title := s.finderArticles(ctx, []int64{articleID})[articleID].Title
	if title == "" {
		title = missingArticle
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	nowStr := now.UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx,
		`UPDATE task_articles SET removed_by = ?, removed_at = ? WHERE id = ? AND task_id = ? AND removed_at IS NULL`,
		actorID, nowStr, linkID, taskID)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return nil // somebody else got there first
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "article_unlinked", articleDetail{ArticleID: articleID, Title: title}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// AddPlace adds a COSMOS place (a shared-folder path) to a job (gates 8.13,
// 8.15, 8.16, 8.17).
func (s *Store) AddPlace(ctx context.Context, taskID int64, name, folderPath string, actorID int64, now time.Time) error {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxPlaceName {
		return ErrPlaceNameLong
	}
	p, err := NormalisePath(folderPath)
	if err != nil {
		return err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := addPlaceTx(ctx, tx, taskID, name, p, actorID, now, true); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// addPlaceTx is the checks and the insert, shared with the Add task window,
// which saves a new job's places in the same transaction as the job (D-106).
// path is already normalised. record says whether History gets a line: a place
// added to an existing job does, one saved with a new job doesn't, because the
// job's own "created" line says it all.
func addPlaceTx(ctx context.Context, tx *sql.Tx, taskID int64, name, normPath string, actorID int64, now time.Time, record bool) error {
	if err := checkLive(ctx, tx, taskID); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT path, position FROM task_places WHERE task_id = ? AND removed_at IS NULL`, taskID)
	if err != nil {
		return err
	}
	count, maxPos := 0, 0
	dup := false
	for rows.Next() {
		var existing string
		var pos int
		if err := rows.Scan(&existing, &pos); err != nil {
			rows.Close()
			return err
		}
		count++
		if pos > maxPos {
			maxPos = pos
		}
		if strings.EqualFold(existing, normPath) {
			dup = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if dup {
		return ErrPlaceDuplicate
	}
	if count >= maxJobPlaces {
		return ErrTooManyPlaces
	}
	nowStr := now.UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx,
		`INSERT INTO task_places (task_id, name, path, position, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		taskID, name, normPath, maxPos+1, actorID, nowStr)
	if err != nil {
		return err
	}
	placeID, _ := res.LastInsertId()
	if !record {
		return nil
	}
	return recordActivity(ctx, tx, taskID, actorID, "place_added", placeDetail{PlaceID: placeID, Name: PlaceDisplayName(name, normPath), Path: normPath}, nowStr)
}

// RemovePlace takes a place off a job (gate 8.17). It never deletes.
func (s *Store) RemovePlace(ctx context.Context, taskID, placeID, actorID int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var name, folder string
	var removed sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT name, path, removed_at FROM task_places WHERE id = ? AND task_id = ?`, placeID, taskID).Scan(&name, &folder, &removed)
	if err == sql.ErrNoRows {
		return ErrReferenceGone
	}
	if err != nil {
		return err
	}
	if removed.Valid {
		return nil // somebody else got there first
	}
	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `UPDATE task_places SET removed_by = ?, removed_at = ? WHERE id = ?`, actorID, nowStr, placeID); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "place_removed", placeDetail{PlaceID: placeID, Name: PlaceDisplayName(name, folder), Path: folder}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// ListReferences reads a job's live articles and places. Titles, categories and
// the Archived mark come from the Knowledge Base through the finder, so a
// renamed or archived article shows as it is now (gate 8.12).
func (s *Store) ListReferences(ctx context.Context, taskID int64) (References, error) {
	var refs References
	type link struct{ id, article int64 }
	var links []link
	rows, err := s.DB.QueryContext(ctx, `SELECT id, article_id FROM task_articles WHERE task_id = ? AND removed_at IS NULL ORDER BY id`, taskID)
	if err != nil {
		return refs, err
	}
	for rows.Next() {
		var l link
		if err := rows.Scan(&l.id, &l.article); err != nil {
			rows.Close()
			return refs, err
		}
		links = append(links, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return refs, err
	}
	ids := make([]int64, len(links))
	for i, l := range links {
		ids[i] = l.article
	}
	known := s.finderArticles(ctx, ids)
	for _, l := range links {
		a, ok := known[l.article]
		v := ArticleLink{LinkID: l.id, ArticleID: l.article, Title: a.Title, Category: a.Category, Archived: a.Archived}
		if !ok {
			v.Title, v.Missing = missingArticle, true
		}
		refs.Articles = append(refs.Articles, v)
	}

	prows, err := s.DB.QueryContext(ctx, `SELECT id, name, path FROM task_places WHERE task_id = ? AND removed_at IS NULL ORDER BY position, id`, taskID)
	if err != nil {
		return refs, err
	}
	defer prows.Close()
	for prows.Next() {
		var p Place
		if err := prows.Scan(&p.ID, &p.Name, &p.Path); err != nil {
			return refs, err
		}
		p.Label = PlaceDisplayName(p.Name, p.Path)
		refs.Places = append(refs.Places, p)
	}
	return refs, prows.Err()
}

// ArticleJob is one job that links an article, for the article's own page
// (gate 8.18).
type ArticleJob struct {
	ID         int64
	Title      string
	StageLabel string
	Finished   bool
}

// JobsUsingArticle lists the jobs that link an article, leaving out removed
// jobs and removed links. Jobs still to do come first, in column order, then
// finished ones; within each, by title.
func (s *Store) JobsUsingArticle(ctx context.Context, articleID int64) ([]ArticleJob, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT DISTINCT t.id, t.title, t.stage
		FROM task_articles a
		JOIN tasks t ON t.id = a.task_id
		WHERE a.article_id = ? AND a.removed_at IS NULL AND t.removed_at IS NULL
		ORDER BY CASE t.stage WHEN 'doing' THEN 0 WHEN 'todo' THEN 1 WHEN 'idea' THEN 2 ELSE 3 END, t.title COLLATE NOCASE, t.id`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ArticleJob
	for rows.Next() {
		var j ArticleJob
		var stage string
		if err := rows.Scan(&j.ID, &j.Title, &stage); err != nil {
			return nil, err
		}
		j.StageLabel, j.Finished = stageLabels[stage], stage == StageDone
		out = append(out, j)
	}
	return out, rows.Err()
}

// cleanNewSteps tidies the steps typed with a new job: blank ones are dropped
// (a blank line in the one-per-line box), and the limits are the single-step
// route's (gate 8.32): 200 characters each, 50 in all.
func cleanNewSteps(raw []string) ([]string, error) {
	var out []string
	for _, r := range raw {
		if strings.TrimSpace(r) == "" {
			continue
		}
		text, err := cleanStepText(r)
		if err != nil {
			return nil, err
		}
		out = append(out, text)
	}
	if len(out) > MaxSteps {
		return nil, ErrTooManySteps
	}
	return out, nil
}

// cleanNewPlaces normalises the places typed with a new job; an entirely empty
// pair (no name and no path) is dropped. A bad path or a long name is refused
// here, and a duplicate or a 21st in the transaction that saves them.
func cleanNewPlaces(raw []PlaceInput) ([]PlaceInput, error) {
	var out []PlaceInput
	for _, r := range raw {
		name := strings.TrimSpace(r.Name)
		if name == "" && strings.TrimSpace(r.Path) == "" {
			continue
		}
		if utf8.RuneCountInString(name) > maxPlaceName {
			return nil, ErrPlaceNameLong
		}
		p, err := NormalisePath(r.Path)
		if err != nil {
			return nil, err
		}
		out = append(out, PlaceInput{Name: name, Path: p})
	}
	return out, nil
}

// checkNewArticles checks the articles chosen for a new job against the same
// rules as AddArticle (gate 8.32): published, not twice, at most 20.
func (s *Store) checkNewArticles(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if seen[id] {
			return ErrArticleLinked
		}
		seen[id] = true
	}
	if len(ids) > maxJobArticles {
		return ErrTooManyArticles
	}
	known := s.finderArticles(ctx, ids)
	for _, id := range ids {
		ref, ok := known[id]
		if !ok {
			return ErrArticleGone
		}
		if ref.Archived {
			return ErrArticleArchived
		}
	}
	return nil
}
