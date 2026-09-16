//go:build integration

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// buildBinary compiles the comphq binary once for this test. go test runs
// with the working directory set to this package's directory, so "." is
// this main package.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "comphq")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestHealthzAndHealthcheckSubcommand(t *testing.T) {
	bin := buildBinary(t)
	addr := fmt.Sprintf("127.0.0.1:%d", freePort(t))

	server := exec.Command(bin)
	server.Env = append(os.Environ(),
		"COMPHQ_DATA_DIR="+t.TempDir(),
		"COMPHQ_ADDR="+addr,
	)
	var stderr strings.Builder
	server.Stderr = &stderr
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	}()

	url := "http://" + addr + "/healthz"
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	var lastStatus int
	healthy := false
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
			continue
		}
		lastStatus = resp.StatusCode
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			healthy = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !healthy {
		t.Fatalf("/healthz never returned 200 (last status %d, last error %v); server stderr:\n%s", lastStatus, lastErr, stderr.String())
	}

	check := exec.Command(bin, "healthcheck")
	check.Env = append(os.Environ(), "COMPHQ_ADDR="+addr)
	if out, err := check.CombinedOutput(); err != nil {
		t.Fatalf("healthcheck against a live server should exit 0: %v\n%s", err, out)
	}

	_ = server.Process.Kill()
	_ = server.Wait()

	checkDown := exec.Command(bin, "healthcheck")
	checkDown.Env = append(os.Environ(), "COMPHQ_ADDR="+addr)
	if err := checkDown.Run(); err == nil {
		t.Fatal("healthcheck against a stopped server should exit non-zero")
	}
}
