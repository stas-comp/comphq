import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

async function setLastBackupDaysAgo(server: { baseURL: string }, page: import('@playwright/test').Page, daysAgo: number) {
  const res = await page.request.post(server.baseURL + '/__test/backups/set-last-backup-at', {
    form: { days_ago: String(daysAgo) },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBeTruthy();
}

// SPEC gate 1.35: fresh install (test mode seeds a just-now backup at
// startup, SPEC B7's own test plan) shows no banner anywhere.
test('no banner shows with a fresh last backup', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/');
  await ready(page);
  await expect(page.locator('.backup-banner')).toHaveCount(0);
});

// SPEC gate 1.35: "if the last backup is more than 2 days old, a warning
// shows at the top of every page." Reuses the /__test/routes list so
// this covers every real route without hard-coding it twice (mirrors
// e2e/frame/gates.spec.ts's gate 1.07 test).
test('a stale backup shows the warning banner on every registered page', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/');
  await setLastBackupDaysAgo(server, page, 3);

  const res = await page.request.get(server.baseURL + '/__test/routes');
  const routes: string[] = await res.json();
  expect(routes.length).toBeGreaterThan(0);

  for (const route of routes) {
    await page.goto(server.baseURL + route);
    await ready(page);
    await expect(page.locator('.backup-banner')).toHaveCount(1);
    await expect(page.locator('.backup-banner')).toContainText('more than 2 days old');
  }

  // Setting a fresh value again makes the banner go away.
  await setLastBackupDaysAgo(server, page, 0);
  await page.goto(server.baseURL + '/');
  await ready(page);
  await expect(page.locator('.backup-banner')).toHaveCount(0);
});

test('Settings > Backups shows the last backup time', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/settings/backups');
  await ready(page);
  await expect(page.locator('h1')).toHaveText('Backups');
  await expect(page.locator('#last-backup-at')).not.toHaveText('No backup has been made yet.');
  await expect(page.locator('.message')).toHaveCount(0);
});
