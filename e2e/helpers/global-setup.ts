import { execFileSync } from 'node:child_process';
import path from 'node:path';

// Builds the real binary once before the run, so every worker can spawn its
// own server against a temp data dir (SPEC §2.6: "global setup builds the
// binary"). In BASE_URL mode we're pointed at an already-running server
// instead, so there's nothing to build.
export default async function globalSetup(): Promise<void> {
  if (process.env.BASE_URL) return;

  const root = path.join(__dirname, '..', '..');
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
