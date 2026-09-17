import { test as base, expect } from '@playwright/test';
import { type ChildProcess, execFile, execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { freePort, waitForHealthy } from '../helpers/fixtures';

export type SeededServer = { baseURL: string; dataDir: string };

// The speed project runs with a single worker (playwright.config.ts), so
// this worker-scoped fixture seeds the test library exactly once (PLAN.md
// P1-27, D-24) rather than once per test: the seeder writes directly into
// comphq.db and exits before the real comphq binary — built once in global
// setup, like COMPHQ_TEST_BINARY — ever opens it, so the two processes
// never touch the same SQLite file at the same time.
export const test = base.extend<{}, { server: SeededServer }>({
  // eslint-disable-next-line no-empty-pattern
  server: [async ({}, use) => {
    const bin = process.env.COMPHQ_TEST_BINARY;
    const seedBin = process.env.COMPHQ_SEED_BINARY;
    if (!bin || !seedBin) {
      throw new Error('COMPHQ_TEST_BINARY / COMPHQ_SEED_BINARY are not set; global setup should have built them');
    }

    const dataDir = fs.mkdtempSync(path.join(os.tmpdir(), 'comphq-speed-'));

    execFileSync(seedBin, ['-data', dataDir, '-articles', '500', '-words', '800', '-categories', '15', '-images=true'], {
      stdio: 'inherit',
    });

    const port = await freePort();
    const addr = `127.0.0.1:${port}`;
    const child: ChildProcess = execFile(bin, [], {
      env: { ...process.env, COMPHQ_DATA_DIR: dataDir, COMPHQ_ADDR: addr, COMPHQ_TEST_MODE: '1' },
    });
    child.stderr?.on('data', (chunk) => process.stderr.write(`[comphq speed] ${chunk}`));

    const baseURL = `http://${addr}`;
    await waitForHealthy(baseURL);

    await use({ baseURL, dataDir });

    child.kill();
  }, { scope: 'worker' }],
});

export { expect };
