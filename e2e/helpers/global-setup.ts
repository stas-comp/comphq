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

  execFileSync('go', ['build', '-o', bin, './cmd/comphq'], {
    cwd: root,
    stdio: 'inherit',
  });

  process.env.COMPHQ_TEST_BINARY = bin;
}
