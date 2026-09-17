import { defineConfig, devices } from '@playwright/test';

// BASE_URL mode points at an already-running, already-populated server
// (used by the container offline/upgrade runs): global setup skips building
// a binary, no worker starts its own server, and @fresh tests (which need a
// private, empty database) are excluded.
const BASE_URL = process.env.BASE_URL;

export default defineConfig({
  testDir: 'e2e',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: [['list']],
  globalSetup: require.resolve('./e2e/helpers/global-setup.ts'),
  grepInvert: BASE_URL ? /@fresh/ : undefined,
  use: {
    trace: 'retain-on-failure',
    viewport: { width: 1366, height: 768 },
  },
  projects: [
    {
      name: 'e2e',
      testDir: 'e2e',
      testIgnore: ['a11y/**', 'speed/**', 'screens/**', 'smoke/**'],
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'a11y',
      testDir: 'e2e/a11y',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'speed',
      testDir: 'e2e/speed',
      // The test library is seeded once per worker (D-24); a single
      // worker keeps that a true "once" instead of once per parallel
      // shard, and keeps timing measurements free of CPU contention
      // from other workers' tests running at the same time.
      workers: 1,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'screens',
      testDir: 'e2e/screens',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'smoke',
      testDir: 'e2e/smoke',
      // @seed, @verify and @verify-prev depend on running against the
      // same server and data (in BASE_URL mode, container-test.sh's own
      // sequencing already guarantees this; locally, without BASE_URL, a
      // single worker keeps every test in this project on the one
      // worker-spawned server instead of scattering them across several
      // independent, empty ones).
      workers: 1,
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
