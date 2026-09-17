import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';

// Builds the real binary once before the run, so every worker can spawn its
// own server against a temp data dir (SPEC §2.6: "global setup builds the
// binary"). In BASE_URL mode we're pointed at an already-running server
// instead, so there's nothing to build there — except the unzip helper
// below, which e2e/settings/export.spec.ts needs regardless of mode.
export default async function globalSetup(): Promise<void> {
  const root = path.join(__dirname, '..', '..');

  // e2e/settings/export.spec.ts (SPEC gate 1.34) needs to inspect the
  // downloaded export as real files under a file:// URL, so it extracts
  // the zip with this small Go helper rather than an npm dependency.
  // Needed in BASE_URL mode too (PLAN.md P1-38's offline-container run,
  // which skips every other build below) — but skipped there if the
  // binary already exists, rather than run unconditionally: the
  // container-test.sh sub-steps that run this file over BASE_URL from
  // inside the "runner" Playwright container (mcr.microsoft.com/playwright)
  // have no Go toolchain at all, unlike the earlier host-side sub-steps
  // (e.g. seeding, right after the container comes up) that share the
  // same bind-mounted repo and build it there first.
  const unzipBin = path.join(
    root,
    'bin',
    process.platform === 'win32' ? 'comphq-unzip-e2e.exe' : 'comphq-unzip-e2e',
  );
  if (!fs.existsSync(unzipBin)) {
    execFileSync('go', ['build', '-o', unzipBin, './e2e/unzip'], {
      cwd: root,
      stdio: 'inherit',
    });
  }
  process.env.COMPHQ_UNZIP_BINARY = unzipBin;

  if (process.env.BASE_URL) return;

  const bin = path.join(
    root,
    'bin',
    process.platform === 'win32' ? 'comphq-e2e.exe' : 'comphq-e2e',
  );

  // -X main.version pins a known value so tests (e.g. the About page) can
  // assert on it (PLAN.md P1-15: "test build uses -X main.version=0.0.0-test").
  execFileSync('go', ['build', '-ldflags', '-X main.version=0.0.0-test', '-o', bin, './cmd/comphq'], {
    cwd: root,
    stdio: 'inherit',
  });

  process.env.COMPHQ_TEST_BINARY = bin;

  // The speed project's test-library seeder (PLAN.md P1-27): built once
  // here like the app binary itself, rather than via `go run` per worker,
  // so a slow `go build` never counts against a speed budget.
  const seedBin = path.join(
    root,
    'bin',
    process.platform === 'win32' ? 'comphq-seed-e2e.exe' : 'comphq-seed-e2e',
  );
  execFileSync('go', ['build', '-o', seedBin, './e2e/seed'], {
    cwd: root,
    stdio: 'inherit',
  });
  process.env.COMPHQ_SEED_BINARY = seedBin;
}
