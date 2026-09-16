// Command comphq is the Comp HQ server binary. It also answers the
// "healthcheck" subcommand used by the container's HEALTHCHECK.
package main

import (
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
	"github.com/stas-comp/comphq/internal/db"
)

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

	migrations, err := db.LoadMigrations(comphq.Migrations, "app")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if err := db.RunMigrations(sqlDB, version, migrations); err != nil {
		// Refuse to start on a failed migration (SPEC §2.5), with a clear
		// log line explaining why.
		log.Fatalf("refusing to start: migration failed: %v", err)
	}

	srv, err := app.NewServer(sqlDB, version, cfg.TestMode)
	if err != nil {
		return fmt.Errorf("build server: %w", err)
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
