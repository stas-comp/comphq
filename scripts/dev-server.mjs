#!/usr/bin/env node
// Local manual dev server for browser-based spot checks during
// development. Not used by any test or CI path — those spawn the
// binary themselves with their own env (e2e/helpers/fixtures.ts).
// `node scripts/dev-server.mjs mc` runs it as MC HQ on :8091 (D-103).
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const brand = process.argv[2] ?? '';
const dataDir = fs.mkdtempSync(path.join(os.tmpdir(), 'comphq-dev-'));

const result = spawnSync('go', ['run', './cmd/comphq'], {
  stdio: 'inherit',
  env: {
    ...process.env,
    COMPHQ_DATA_DIR: dataDir,
    COMPHQ_ADDR: brand === 'mc' ? ':8091' : ':8090',
    COMPHQ_BRAND: brand,
    COMPHQ_TEST_MODE: '1',
  },
});
process.exit(result.status ?? 1);
