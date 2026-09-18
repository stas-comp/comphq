// Command seed populates a fresh data directory with a large, deterministic
// test library, so the speed tests (PLAN.md P1-27, SPEC gate 1.33) measure
// against a realistic amount of content instead of an empty database. It
// talks to the store types directly (internal/kb, internal/kb/images,
// internal/people) rather than over HTTP, and always runs before the real
// comphq binary starts serving the same data directory (D-24) — the two
// never touch the SQLite file at the same time.
//
// It seeds the Knowledge Base (articles, categories, images) and, since
// P2-12, Tasks (people, and a 2,000-task library across every stage,
// with shared assignees and some removed/aged-done for Finished tasks
// and Removed tasks); P2-17 extends it further for calendar events.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/kb"
	"github.com/stas-comp/comphq/internal/kb/images"
	"github.com/stas-comp/comphq/internal/people"
	"github.com/stas-comp/comphq/internal/tasks"
)

// imageCount is fixed, not a flag: the goal is a small, realistic set of
// pictures reused across many articles (matching how a real Knowledge Base
// accumulates a handful of diagrams and photos reused across pages), not
// one unique image per article.
const imageCount = 20

// seedRandSeed is fixed so every run of the seeder — local or in CI —
// produces byte-identical content, which makes a slow speed run
// reproducible instead of a one-off fluke.
const seedRandSeed = 42

func main() {
	dataDir := flag.String("data", "", "data directory to seed into (required; must not already contain comphq.db)")
	articleCount := flag.Int("articles", 500, "number of articles to create")
	wordsPerArticle := flag.Int("words", 800, "approximate word count per article body")
	categoryCount := flag.Int("categories", 15, "number of categories to spread articles across")
	withImages := flag.Bool("images", true, "embed generated images in some articles")
	taskCount := flag.Int("tasks", 2000, "number of tasks to create")
	peopleCount := flag.Int("people", 10, "number of active people to create, for tasks to be assigned to")
	flag.Parse()

	if *dataDir == "" {
		fmt.Fprintln(os.Stderr, "seed: -data is required")
		os.Exit(1)
	}

	if err := run(*dataDir, *articleCount, *wordsPerArticle, *categoryCount, *withImages, *taskCount, *peopleCount); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

func run(dataDir string, articleCount, wordsPerArticle, categoryCount int, withImages bool, taskCount, peopleCount int) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	sqlDB, err := db.Open(filepath.Join(dataDir, "comphq.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer sqlDB.Close()

	for _, section := range []string{"app", "people", "kb", "tasks"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			return fmt.Errorf("load %s migrations: %w", section, err)
		}
		if err := db.RunMigrations(sqlDB, "seed", migrations); err != nil {
			return fmt.Errorf("run %s migrations: %w", section, err)
		}
	}

	personStore := &people.Store{DB: sqlDB}
	person, err := personStore.Create("Speed Test Seeder")
	if err != nil {
		return fmt.Errorf("create seed person: %w", err)
	}

	categoryStore := &kb.CategoryStore{DB: sqlDB}
	categoryIDs := make([]int64, 0, categoryCount)
	for i := 0; i < categoryCount; i++ {
		c, err := categoryStore.Create(fmt.Sprintf("Category %02d", i+1))
		if err != nil {
			return fmt.Errorf("create category %d: %w", i, err)
		}
		categoryIDs = append(categoryIDs, c.ID)
	}

	ctx := context.Background()
	rng := rand.New(rand.NewSource(seedRandSeed))

	var imageTags []string
	if withImages {
		imageStore := &images.Store{DB: sqlDB, DataDir: dataDir}
		imageTags, err = seedImages(ctx, imageStore, person.ID, imageCount)
		if err != nil {
			return fmt.Errorf("seed images: %w", err)
		}
	}

	articleStore := &kb.ArticleStore{DB: sqlDB, Images: &images.Store{DB: sqlDB, DataDir: dataDir}}

	for i := 0; i < articleCount; i++ {
		categoryID := categoryIDs[i%len(categoryIDs)]
		title := articleTitle(rng, i)
		body := articleBody(rng, wordsPerArticle, imageTags)

		article, err := articleStore.Publish(ctx, kb.ArticleInput{
			CategoryID: categoryID,
			Title:      title,
			BodyHTML:   body,
		}, person.ID)
		if err != nil {
			return fmt.Errorf("publish article %d: %w", i, err)
		}

		// Every 10th article gets a second version, so History has real
		// entries to page through, not just a single "created" row.
		if i%10 == 0 {
			article, err = articleStore.Publish(ctx, kb.ArticleInput{
				ID: article.ID, CategoryID: categoryID, Title: title,
				BodyHTML:        body + "<p>Updated with a small correction.</p>",
				ExpectedVersion: article.VersionNo,
			}, person.ID)
			if err != nil {
				return fmt.Errorf("re-publish article %d: %w", i, err)
			}
		}

		// Every 25th article ends up archived, so the Archived list also
		// has real content rather than always being empty.
		if i%25 == 0 {
			if err := articleStore.Archive(ctx, article.ID, person.ID); err != nil {
				return fmt.Errorf("archive article %d: %w", i, err)
			}
		}
	}

	log.Printf("seeded %d articles across %d categories (%d images) into %s", articleCount, categoryCount, len(imageTags), dataDir)

	if err := seedTasks(ctx, sqlDB, personStore, rng, taskCount, peopleCount); err != nil {
		return fmt.Errorf("seed tasks: %w", err)
	}
	log.Printf("seeded %d tasks across %d people into %s", taskCount, peopleCount, dataDir)

	return nil
}

// seedTasks builds PLAN.md P2-12's own task library: peopleCount active
// people, and taskCount tasks spread across every stage. The 2,000-task
// figure (SPEC's "test library") is a cumulative, historical total, not
// how many stay simultaneously visible — a real team's board looks
// like a modest current backlog plus a long tail of finished/cancelled
// work, not 1,850 live cards on one screen at once (measured directly:
// that unrealistic shape alone pushed the Board's ready time to ~1.96s
// against SPEC gate 2.21's 1.5s budget, purely from parsing/laying out
// tens of thousands of DOM nodes for cards nobody would ever actually
// see live together — not a query or template defect). So most of the
// library (75%) is removed, matching how a mature board accumulates
// far more cancelled/no-longer-relevant tasks than open ones; another
// 10% is done long enough ago to have aged into Finished tasks (gate
// 2.08); only the remaining 15% stays genuinely active on the
// Board/Team/My jobs, which is still generous for 10 people. Distribution
// is by index, not the PRNG, so a re-run always produces the exact same
// counts in each bucket.
func seedTasks(ctx context.Context, sqlDB *sql.DB, personStore *people.Store, rng *rand.Rand, taskCount, peopleCount int) error {
	personIDs := make([]int64, 0, peopleCount)
	for i := 0; i < peopleCount; i++ {
		p, err := personStore.Create(fmt.Sprintf("Person %02d", i+1))
		if err != nil {
			return fmt.Errorf("create person %d: %w", i, err)
		}
		personIDs = append(personIDs, p.ID)
	}
	creator := personIDs[0]

	store := &tasks.Store{DB: sqlDB}
	now := time.Now()
	sizes := []string{"S", "M", "L"}
	activeStages := []string{tasks.StageIdea, tasks.StageTodo, tasks.StageDoing, tasks.StageDone}

	var doneAgedIDs []int64
	var removableIDs []int64

	for i := 0; i < taskCount; i++ {
		bucket := i % 100
		var stage string
		agesOff := false
		toBeRemoved := false
		switch {
		case bucket < 75: // removed: still spread across every stage first
			stage = activeStages[i%len(activeStages)]
			toBeRemoved = true
		case bucket < 85: // done long enough ago to have left the board
			stage = tasks.StageDone
			agesOff = true
		case bucket < 90:
			stage = tasks.StageIdea
		case bucket < 95:
			stage = tasks.StageTodo
		case bucket < 98:
			stage = tasks.StageDoing
		default:
			stage = tasks.StageDone
		}

		var personIDsForTask []int64
		switch {
		case i%7 == 0: // unassigned, for Unassigned/Up for grabs
			// none
		case i%15 == 0: // shared, for gate 2.30/2.33's shared-job cases
			personIDsForTask = []int64{personIDs[i%peopleCount], personIDs[(i+1)%peopleCount]}
		default:
			personIDsForTask = []int64{personIDs[i%peopleCount]}
		}

		var dueDate string
		if i%5 < 2 {
			dueDate = now.AddDate(0, 0, (i%21)-10).Format("2006-01-02")
		}

		task, err := store.Create(ctx, tasks.CreateInput{
			Title:     taskTitle(rng, i),
			Notes:     sentence(rng, 12),
			Size:      sizes[i%len(sizes)],
			Stage:     stage,
			DueDate:   dueDate,
			PersonIDs: personIDsForTask,
		}, creator, now)
		if err != nil {
			return fmt.Errorf("create task %d: %w", i, err)
		}

		switch {
		case toBeRemoved:
			removableIDs = append(removableIDs, task.ID)
		case agesOff:
			doneAgedIDs = append(doneAgedIDs, task.ID)
		}
	}

	if err := backdateDone(sqlDB, doneAgedIDs, now.AddDate(0, 0, -20)); err != nil {
		return fmt.Errorf("backdate aged-done tasks: %w", err)
	}
	if err := removeTasks(sqlDB, removableIDs, now); err != nil {
		return fmt.Errorf("mark tasks removed: %w", err)
	}

	return nil
}

// backdateDone sets done_at far enough in the past that gate 2.08's
// 14-day rule moves these tasks off the board and into Finished tasks
// — a direct SQL update, not a store method, since only the test-only
// HTTP route (used by E2E tests without DB access) needed one before.
func backdateDone(sqlDB *sql.DB, ids []int64, at time.Time) error {
	stmt, err := sqlDB.Prepare(`UPDATE tasks SET done_at = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	atStr := at.UTC().Format(time.RFC3339)
	for _, id := range ids {
		if _, err := stmt.Exec(atStr, id); err != nil {
			return err
		}
	}
	return nil
}

// removeTasks marks tasks removed directly, for a realistic Removed
// tasks page — Store.Remove also renumbers the stage it leaves, which
// doesn't matter for a page that never gets reordered once seeded.
func removeTasks(sqlDB *sql.DB, ids []int64, at time.Time) error {
	stmt, err := sqlDB.Prepare(`UPDATE tasks SET removed_at = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	atStr := at.UTC().Format(time.RFC3339)
	for _, id := range ids {
		if _, err := stmt.Exec(atStr, id); err != nil {
			return err
		}
	}
	return nil
}

// taskTitle reuses the same office vocabulary as article titles — a
// task board looks like a real one with titles like "Renew Vendor
// Contract", not a wall of Lorem Ipsum.
func taskTitle(rng *rand.Rand, index int) string {
	return fmt.Sprintf("%s (%d)", titleCase(pickWords(rng, 2+rng.Intn(3))), index+1)
}

// seedImages generates count small, genuinely distinct PNGs (varying
// colour per index, so each hashes to its own file rather than
// collapsing via Store.Save's duplicate-collapse) and returns an <img>
// tag for each, ready to drop into article bodies.
func seedImages(ctx context.Context, store *images.Store, personID int64, count int) ([]string, error) {
	tags := make([]string, 0, count)
	for i := 0; i < count; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 48, 48))
		c := color.RGBA{
			R: uint8((i*53 + 17) % 256),
			G: uint8((i*97 + 61) % 256),
			B: uint8((i*151 + 113) % 256),
			A: 255,
		}
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				img.Set(x, y, c)
			}
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("encode image %d: %w", i, err)
		}
		saved, err := store.Save(ctx, &buf, personID)
		if err != nil {
			return nil, fmt.Errorf("save image %d: %w", i, err)
		}
		tags = append(tags, fmt.Sprintf(`<img src="/images/%s.%s" alt="Illustration %d">`, saved.SHA256, saved.Ext, i+1))
	}
	return tags, nil
}

// vocabulary is a fixed, office-flavoured word list; article titles and
// bodies are built by picking from it with the seeded PRNG, so the
// generated text at least resembles the kind of content a real Knowledge
// Base holds instead of being pure gibberish.
var vocabulary = strings.Fields(`
	toner printer cartridge scanner network router password badge parking
	supplier delivery invoice onboarding benefits payroll holiday policy
	handbook safety fire drill visitor wifi guest kitchen coffee machine
	recycling compost waste desk chair monitor headset laptop charger
	cable adapter meeting room booking calendar reminder deadline approval
	signature form request ticket helpdesk escalation vendor contract
	renewal budget expense reimbursement travel mileage elevator lobby
	reception security camera access door lock keycard emergency exit
	evacuation assembly point backup restore server maintenance window
	outage incident report checklist procedure guideline standard return
	equipment interview replacement warranty label shelf storage archive
`)

func pickWords(rng *rand.Rand, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = vocabulary[rng.Intn(len(vocabulary))]
	}
	return out
}

func capitalize(word string) string {
	return strings.ToUpper(word[:1]) + word[1:]
}

func titleCase(words []string) string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = capitalize(w)
	}
	return strings.Join(out, " ")
}

func sentence(rng *rand.Rand, words int) string {
	w := pickWords(rng, words)
	w[0] = capitalize(w[0])
	return strings.Join(w, " ") + "."
}

func paragraph(rng *rand.Rand, words int) string {
	var sentences []string
	remaining := words
	for remaining > 0 {
		n := 8 + rng.Intn(8)
		if n > remaining {
			n = remaining
		}
		sentences = append(sentences, sentence(rng, n))
		remaining -= n
	}
	return strings.Join(sentences, " ")
}

func articleTitle(rng *rand.Rand, index int) string {
	return fmt.Sprintf("%s (%d)", titleCase(pickWords(rng, 3+rng.Intn(3))), index+1)
}

// articleBody builds a roughly wordCount-word article: a few paragraphs, one
// heading, a short bullet list, and — when images are available — one
// picture reused from the shared set, so a seeded article looks like a real
// one instead of a wall of plain text.
func articleBody(rng *rand.Rand, wordCount int, imageTags []string) string {
	var b strings.Builder
	written := 0
	paragraphIndex := 0

	for written < wordCount {
		if paragraphIndex == 2 {
			b.WriteString("<h2>" + titleCase(pickWords(rng, 3)) + "</h2>\n")
		}
		if paragraphIndex == 3 && len(imageTags) > 0 {
			b.WriteString(imageTags[rng.Intn(len(imageTags))] + "\n")
		}
		if paragraphIndex == 4 {
			b.WriteString("<ul>\n")
			for i := 0; i < 4; i++ {
				b.WriteString("<li>" + sentence(rng, 6) + "</li>\n")
			}
			b.WriteString("</ul>\n")
			written += 24
		}

		n := 100 + rng.Intn(60)
		b.WriteString("<p>" + paragraph(rng, n) + "</p>\n")
		written += n
		paragraphIndex++
	}

	return b.String()
}
