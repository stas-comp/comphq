import { execFile, type ChildProcess } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { freePort, waitForHealthy } from './fixtures';

// A server on a data folder of the test's own, with a fixed "today"
// (COMPHQ_TEST_TODAY, SPEC B4) — and one that can be stopped and started again
// on the same folder with a different date, which is how a test lets a
// Saturday pass: the weekly reset (gate 7.32) happens when the app next runs on
// or after it. A person's cookie holds across restarts (cookies ignore the
// port), so the same browser context keeps being the same person.
export type RunningServer = { baseURL: string; dataDir: string; stop: () => Promise<void> };

export function newDataDir(label: string): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), `comphq-e2e-${label}-`));
}

export async function startServerAt(dataDir: string, today: string): Promise<RunningServer> {
  if (process.env.BASE_URL) {
    throw new Error('startServerAt needs its own process; it cannot run in BASE_URL mode');
  }
  const bin = process.env.COMPHQ_TEST_BINARY;
  if (!bin) throw new Error('COMPHQ_TEST_BINARY is not set; global setup should have built it');

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
  const exited = new Promise<void>((resolve) => child.once('exit', () => resolve()));

  const baseURL = `http://${addr}`;
  await waitForHealthy(baseURL);
  return {
    baseURL,
    dataDir,
    stop: async () => {
      child.kill();
      await exited;
    },
  };
}
