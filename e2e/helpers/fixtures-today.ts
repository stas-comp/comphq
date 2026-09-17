import { execFile, type ChildProcess } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { test as base, expect } from '@playwright/test';
import { freePort, waitForHealthy, type Server } from './fixtures';

// A dedicated server per test (not per worker, unlike helpers/fixtures.ts's
// own `server`), with a fixed COMPHQ_TEST_TODAY (SPEC B4) — for tests
// asserting date-dependent behaviour (an overdue task, the Saturday
// Briefing) where different tests need different "today" values and so
// can't share one worker-wide server the way most tests do. Set the date
// with `test.use({ today: 'YYYY-MM-DD' })` in a describe block; every
// test using this fixture must be tagged @fresh in its title, the same
// as any test needing its own private server — BASE_URL mode has no way
// to control a shared container's idea of "today".
type TodayFixtures = { today: string; server: Server };

export const test = base.extend<TodayFixtures>({
  today: ['2026-09-19', { option: true }],

  // eslint-disable-next-line no-empty-pattern
  server: async ({ today }, use, testInfo) => {
    if (process.env.BASE_URL) {
      throw new Error('fixtures-today\'s server fixture needs its own process; it cannot run in BASE_URL mode');
    }

    const bin = process.env.COMPHQ_TEST_BINARY;
    if (!bin) {
      throw new Error('COMPHQ_TEST_BINARY is not set; global setup should have built it');
    }

    const dataDir = fs.mkdtempSync(path.join(os.tmpdir(), `comphq-e2e-today-${testInfo.workerIndex}-`));
    const port = await freePort();
    const addr = `127.0.0.1:${port}`;

    const child: ChildProcess = execFile(bin, [], {
      env: {
        ...process.env,
        COMPHQ_DATA_DIR: dataDir,
        COMPHQ_ADDR: addr,
        COMPHQ_TEST_MODE: '1',
        COMPHQ_TEST_TODAY: today,
      },
    });
    child.stderr?.on('data', (chunk) => process.stderr.write(`[comphq today=${today}] ${chunk}`));

    const baseURL = `http://${addr}`;
    await waitForHealthy(baseURL);

    await use({ baseURL, dataDir, stubImageHost: '' });

    child.kill();
  },
});

export { expect };
