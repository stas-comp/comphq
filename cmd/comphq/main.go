// Command comphq is the Comp HQ server binary. It also answers the
// "healthcheck" subcommand used by the container's HEALTHCHECK.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "time/tzdata"

	"github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/briefing"
	"github.com/stas-comp/comphq/internal/calendar"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/kb"
	"github.com/stas-comp/comphq/internal/settings"
	"github.com/stas-comp/comphq/internal/tasks"
)

// taskDeadlines adapts *tasks.Store to calendar.DueTasksSource (SPEC
// gate 2.19), converting between each section's own DueTask shape —
// this is the one place allowed to know about both sections (SPEC B2:
// "sections never import each other's internals"; neither
// internal/tasks nor internal/calendar imports the other).
type taskDeadlines struct{ store *tasks.Store }

func (a taskDeadlines) DueTasks(ctx context.Context, from, to time.Time) ([]calendar.DueTask, error) {
	rows, err := a.store.DueTasks(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]calendar.DueTask, len(rows))
	for i, r := range rows {
		out[i] = calendar.DueTask{ID: r.ID, Title: r.Title, DueDate: r.DueDate}
	}
	return out, nil
}

// version is set by ldflags at release build time (SPEC P1-03 behaviour).
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := app.LoadConfig()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	sqlDB, err := db.Open(filepath.Join(cfg.DataDir, "comphq.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer sqlDB.Close()

	srv, err := app.NewServer(sqlDB, version, cfg.DataDir, cfg.TestMode)
	if err != nil {
		return fmt.Errorf("build server: %w", err)
	}

	// Sidebar order (SPEC A4): Briefing, Knowledge Base, Tasks, Calendar,
	// Settings.
	srv.Registry().Add(briefing.Section(srv))
	srv.Registry().Add(kb.Section(srv))
	srv.Registry().Add(tasks.Section(srv))
	srv.Registry().Add(calendar.Section(srv, taskDeadlines{store: &tasks.Store{DB: srv.DB}}))
	srv.Registry().Add(settings.Section(srv, &tasks.Store{DB: srv.DB}, &calendar.Store{DB: srv.DB}))

	sectionMigrations := make(map[string][]db.Migration)
	pending := false
	for _, section := range srv.Registry().MigrationNames() {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			return fmt.Errorf("load %s migrations: %w", section, err)
		}
		sectionMigrations[section] = migrations
		sectionPending, err := db.HasPendingMigrations(sqlDB, migrations)
		if err != nil {
			return fmt.Errorf("check %s migrations: %w", section, err)
		}
		if sectionPending {
			pending = true
		}
	}

	existingData, err := db.HasExistingData(sqlDB)
	if err != nil {
		return fmt.Errorf("check for existing data: %w", err)
	}
	if pending && existingData {
		// SPEC B6: migrations run "after a VACUUM INTO pre-update backup
		// (only when migrations are pending)" — one backup covering every
		// section, not one per section. A fresh install has nothing yet
		// worth protecting, so it skips straight to the first migration
		// run, which is what creates app_meta in the first place.
		if err := db.PreUpdateBackup(sqlDB, cfg.DataDir, version, time.Now()); err != nil {
			log.Fatalf("refusing to start: pre-update backup failed: %v", err)
		}
	}

	for _, section := range srv.Registry().MigrationNames() {
		if err := db.RunMigrations(sqlDB, version, sectionMigrations[section]); err != nil {
			// Refuse to start on a failed migration (SPEC §2.5), with a
			// clear log line explaining why.
			log.Fatalf("refusing to start: migration failed: %v", err)
		}
	}

	if cfg.TestMode {
		// Hundreds of existing tests render pages without caring about
		// backups; a fresh test server's app_meta has no last_backup_at
		// row, which would make every one of them see the stale banner.
		// Seed a fresh timestamp instead of running the real scheduler, so
		// only tests that explicitly set an old timestamp see it (SPEC
		// gate 1.35's own test plan: "test-mode helper").
		if err := db.SetLastBackupAtForTest(sqlDB, time.Now()); err != nil {
			return fmt.Errorf("seed backup status for test mode: %w", err)
		}
	} else {
		go db.RunBackupScheduler(sqlDB, cfg.DataDir, db.RealClock{})
	}

	httpServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: srv.Routes(),
	}
	log.Printf("comphq %s listening on %s", version, cfg.Addr)
	return httpServer.ListenAndServe()
}

// runHealthcheck implements the "comphq healthcheck" subcommand: a GET to
// /healthz on the configured address with a 3s timeout, exiting 0 or 1.
func runHealthcheck() int {
	addr := app.LoadConfig().Addr
	url := fmt.Sprintf("http://%s/healthz", healthcheckHost(addr))

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck failed:", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck failed: status", resp.StatusCode)
		return 1
	}
	return 0
}

// healthcheckHost turns a listen address (which may have no host, e.g.
// ":8080") into a dialable 127.0.0.1 address.
func healthcheckHost(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "127.0.0.1:8080"
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}
