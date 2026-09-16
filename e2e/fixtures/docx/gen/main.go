// Command gen writes the .docx fixtures used by internal/kb/docx and the
// P1-11+ E2E tests. Run via `npm run fixtures:docx` from the repository
// root; every output is committed and must be byte-identical on rerun.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "fixtures:docx:", err)
		os.Exit(1)
	}
}

func readFixtureImage(name string) []byte {
	data, err := os.ReadFile(filepath.Join("e2e", "fixtures", "images", name))
	must(err)
	return data
}

func main() {
	outDir := filepath.Join("e2e", "fixtures", "docx")
	badDir := filepath.Join(outDir, "bad")
	must(os.MkdirAll(outDir, 0o755))
	must(os.MkdirAll(badDir, 0o755))

	must(writeSampleDocx(filepath.Join(outDir, "sample.docx")))
	must(writeGoogleDocsDocx(filepath.Join(outDir, "googledocs.docx")))
	must(writeBig20PagesDocx(filepath.Join(outDir, "big-20pages.docx")))

	must(writeBadFixtures(badDir))

	fmt.Println("wrote docx fixtures to", outDir)
}
