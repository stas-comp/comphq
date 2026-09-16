// Package policy enforces the repository rules from PLAN.md P1-05: pinned
// GitHub Actions, pinned base images, safe deploy config, no raw URLs in
// templates/static, unchanged migrations, and a short Go dependency list.
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot walks up from the current directory (tools/policy, since go test
// runs with the package directory as its working directory) to find the
// repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (no go.mod in any parent directory)")
		}
		dir = parent
	}
}

var usesRe = regexp.MustCompile(`uses:\s*([^\s#]+)`)
var shaRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

func TestWorkflowActionsPinnedBySHA(t *testing.T) {
	dir := filepath.Join(repoRoot(t), ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		t.Skip("no .github/workflows yet")
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !(strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml")) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			m := usesRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			ref := m[1]
			if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "docker://") {
				continue // local and direct-image references aren't marketplace action pins
			}
			parts := strings.SplitN(ref, "@", 2)
			if len(parts) != 2 || !shaRe.MatchString(parts[1]) {
				t.Errorf("%s:%d: %q is not pinned to a 40-hex commit SHA", e.Name(), i+1, ref)
			}
		}
	}
}

func TestDockerfileFromPinnedByDigest(t *testing.T) {
	path := filepath.Join(repoRoot(t), "Dockerfile")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("Dockerfile doesn't exist yet (added in P1-06)")
	}
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
			continue
		}
		if !strings.Contains(trimmed, "@sha256:") {
			t.Errorf("Dockerfile:%d: %q is not pinned by digest", i+1, trimmed)
		}
	}
}

var imageTagRe = regexp.MustCompile(`image:\s*ghcr\.io/stas-comp/comphq:(\S+)`)
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func TestDeployYAMLRules(t *testing.T) {
	path := filepath.Join(repoRoot(t), "deploy", "truenas.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("deploy/truenas.yaml doesn't exist yet (added in P1-06)")
	}
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, "COMPHQ_TEST_MODE") {
		t.Error("deploy/truenas.yaml must never set COMPHQ_TEST_MODE")
	}

	m := imageTagRe.FindStringSubmatch(content)
	if m == nil {
		t.Error("deploy/truenas.yaml has no ghcr.io/stas-comp/comphq image reference")
	} else if !semverRe.MatchString(m[1]) {
		t.Errorf("deploy/truenas.yaml image tag %q must be an exact X.Y.Z version, never 'latest'", m[1])
	}
}

func TestVendorChecksumsMatch(t *testing.T) {
	vendorDir := filepath.Join(repoRoot(t), "web", "static", "vendor")
	data, err := os.ReadFile(filepath.Join(vendorDir, "VENDOR.md"))
	if os.IsNotExist(err) {
		t.Skip("web/static/vendor/VENDOR.md doesn't exist yet")
	}
	if err != nil {
		t.Fatal(err)
	}

	shaRe := regexp.MustCompile(`\b[0-9a-f]{64}\b`)
	// The path lives in its own backtick-quoted table cell, so a name cell
	// mentioning the same filename (e.g. "editor-3.31.3.js" as the Name
	// column, "editor/editor-3.31.3.js" as the Path column) can't be
	// picked up by mistake.
	pathRe := regexp.MustCompile("`([\\w./-]+\\.(?:js|css|woff2))`")

	checked := 0
	for i, line := range strings.Split(string(data), "\n") {
		sha := shaRe.FindString(line)
		pathMatch := pathRe.FindStringSubmatch(line)
		if sha == "" || pathMatch == nil {
			continue
		}
		rel := pathMatch[1]
		content, err := os.ReadFile(filepath.Join(vendorDir, rel))
		if err != nil {
			t.Errorf("VENDOR.md:%d: referenced file %s: %v", i+1, rel, err)
			continue
		}
		sum := sha256.Sum256(content)
		got := hex.EncodeToString(sum[:])
		if got != sha {
			t.Errorf("VENDOR.md:%d: %s checksum mismatch: VENDOR.md says %s, file hashes to %s", i+1, rel, sha, got)
		}
		checked++
	}
	if checked == 0 {
		t.Error("VENDOR.md exists but no name/checksum rows were recognised")
	}
}

var urlRe = regexp.MustCompile(`https?://`)

func TestNoRawURLsInTemplatesOrStatic(t *testing.T) {
	root := repoRoot(t)
	vendorPrefix := filepath.Join(root, "web", "static", "vendor") + string(filepath.Separator)
	// The setup page explains commands to run and may show a URL as plain
	// instructional text; it's allowed once it exists (P1-15).
	allowed := map[string]bool{
		filepath.Join(root, "web", "templates", "settings", "setup.html"): true,
	}

	for _, dir := range []string{
		filepath.Join(root, "web", "templates"),
		filepath.Join(root, "web", "static"),
	} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || strings.HasPrefix(path, vendorPrefix) || allowed[path] {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if urlRe.Match(data) {
				t.Errorf("%s contains a raw http(s):// URL; not allowed outside web/static/vendor", path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
}

func TestMigrationsUnchangedOnceCommitted(t *testing.T) {
	root := repoRoot(t)
	out, err := exec.Command("git", "-C", root, "ls-files", "migrations").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	for _, rel := range strings.Fields(string(out)) {
		prev, err := exec.Command("git", "-C", root, "show", "HEAD~1:"+rel).Output()
		if err != nil {
			// New in this commit, or there's no HEAD~1 yet: nothing to compare.
			continue
		}
		current, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(prev) != string(current) {
			t.Errorf("%s was changed after being committed; migrations are expand-only and must never be edited (SPEC §2.8)", rel)
		}
	}
}

func TestMigrationsNoForbiddenStatements(t *testing.T) {
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "migrations", "*", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"DROP", "RENAME", "ALTER COLUMN"}
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		upper := strings.ToUpper(string(data))
		for _, word := range forbidden {
			if strings.Contains(upper, word) {
				t.Errorf("%s contains forbidden %q; migrations are expand-only (SPEC §2.8)", path, word)
			}
		}
	}
}

func TestGoModAtMostFiveDirectRequirements(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	direct := 0
	inBlock := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "require ("):
			inBlock = true
		case inBlock && line == ")":
			inBlock = false
		case inBlock:
			if line != "" && !strings.Contains(line, "// indirect") {
				direct++
			}
		case strings.HasPrefix(line, "require ") && !strings.Contains(line, "("):
			rest := strings.TrimPrefix(line, "require ")
			if !strings.Contains(rest, "// indirect") {
				direct++
			}
		}
	}

	if direct > 5 {
		t.Errorf("go.mod has %d direct requirements, want at most 5 (SPEC §2.8)", direct)
	}
}
