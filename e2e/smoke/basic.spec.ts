import { test } from '../helpers/fixtures';
import { ready } from '../helpers/ready';

// The smoke suite runs against a real, already-running server (inside a
// container, over BASE_URL — see e2e/helpers/fixtures.ts) for the offline
// and upgrade/rollback harnesses (P1-12). Real content-creation seeding
// starts once the People section exists (P1-13) and grows with every
// later release; for now these placeholders prove the app is reachable
// and healthy end to end, which is all "seed" and "verify" can mean
// before there's any data to seed.
test('@seed the test library', async ({ page, server }) => {
  await page.goto(server.baseURL + '/');
  await ready(page);
});

test('@verify the seeded data is present', async ({ page, server }) => {
  await page.goto(server.baseURL + '/');
  await ready(page);
});

// Run against the *current* release's smoke suite while pointed at data
// seeded by the *previous* release, to prove an upgrade didn't lose or
// break anything old.
test('@verify-prev reads data seeded by the previous release', async ({ page, server }) => {
  await page.goto(server.baseURL + '/');
  await ready(page);
});
