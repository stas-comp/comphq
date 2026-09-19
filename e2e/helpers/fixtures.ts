import { test as base, expect } from '@playwright/test';
import { type ChildProcess, execFile } from 'node:child_process';
import fs from 'node:fs';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import { retireTestPeople } from './people';

export type Server = { baseURL: string; dataDir: string; stubImageHost: string };

export async function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.listen(0, '127.0.0.1', () => {
      const address = srv.address();
      if (address && typeof address === 'object') {
        const { port } = address;
        srv.close(() => resolve(port));
      } else {
        srv.close(() => reject(new Error('could not find a free port')));
      }
    });
    srv.on('error', reject);
  });
}

export async function waitForHealthy(baseURL: string, timeoutMs = 15_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseURL}/healthz`);
      if (res.ok) return;
    } catch (err) {
      lastError = err;
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(`server at ${baseURL} never became healthy: ${String(lastError)}`);
}

// Each worker gets its own server: its own temp /data folder and a free
// port, so workers never share state (SPEC §2.6).
export const test = base.extend<{ retirePeople: void }, { server: Server }>({
  // After every test, hand back the people it signed in as (see people.ts).
  retirePeople: [
    // eslint-disable-next-line no-empty-pattern
    async ({}, use) => {
      await use();
      await retireTestPeople();
    },
    { auto: true },
  ],

  // eslint-disable-next-line no-empty-pattern
  server: [async ({}, use, workerInfo) => {
    const baseURLFromEnv = process.env.BASE_URL;
    if (baseURLFromEnv) {
      // dataDir and stubImageHost are empty here: BASE_URL points at an
      // already-running, separately-hosted server (e.g. a container), so
      // this process can't see its filesystem, and the fixed
      // COMPHQ_TEST_ALLOW_FETCH_HOST allowlist below wasn't set up for
      // it. Tests needing either are tagged @fresh and excluded from this
      // mode (see below).
      await use({ baseURL: baseURLFromEnv, dataDir: '', stubImageHost: '' });
      return;
    }

    const bin = process.env.COMPHQ_TEST_BINARY;
    if (!bin) {
      throw new Error('COMPHQ_TEST_BINARY is not set; global setup should have built it');
    }

    const dataDir = fs.mkdtempSync(path.join(os.tmpdir(), `comphq-e2e-${workerInfo.workerIndex}-`));
    const port = await freePort();
    const addr = `127.0.0.1:${port}`;

    // A single fixed loopback address, reserved per worker, that
    // internal/kb/images.FetchAndStore is allowed to reach in test mode
    // (SPEC P1-23's SSRF guard blocks every other loopback/private
    // address even under COMPHQ_TEST_MODE=1). Tests bind their own stub
    // image server to exactly this host:port when they need one; since a
    // worker runs its tests one at a time, only one test ever holds it at
    // once.
    const stubPort = await freePort();
    const stubImageHost = `127.0.0.1:${stubPort}`;

    const child: ChildProcess = execFile(bin, [], {
      env: {
        ...process.env,
        COMPHQ_DATA_DIR: dataDir,
        COMPHQ_ADDR: addr,
        COMPHQ_TEST_MODE: '1',
        COMPHQ_TEST_ALLOW_FETCH_HOST: stubImageHost,
      },
    });
    child.stderr?.on('data', (chunk) => process.stderr.write(`[comphq worker ${workerInfo.workerIndex}] ${chunk}`));

    const baseURL = `http://${addr}`;
    await waitForHealthy(baseURL);

    await use({ baseURL, dataDir, stubImageHost });

    child.kill();
  }, { scope: 'worker' }],
});

export { expect };
