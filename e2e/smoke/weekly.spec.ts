import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Weekly jobs across an upgrade and a rollback (SPEC gate 7.39, B13.5, D-95):
// two weekly jobs are made under the new build, then the container is rolled
// back to the previous release, which has never heard of weekly jobs: it must
// show them as ordinary jobs, and one is finished there (as 1.3.0 would, and
// would tuck away after 14 days). Upgraded again, both must still be weekly.
// (The reset itself needs a Saturday to pass, which a container run cannot wait
// for; gate 7.32's tests drive it with a fixed date. What this proves is that
// the marking survives.)
//
// Tagged @weekly-seed / @weekly-rolledback / @weekly-verify-upgraded, none a
// substring of another, so container-test.sh's grep-filtered phases pick the
// right one. Titles are fixed because a later phase has to find the jobs; the
// person is not. Only plain endpoints are used by the rolled-back step, so the
// same code runs against the previous release.
const A = 'Smoke Weekly Job A';
const B = 'Smoke Weekly Job B';

async function hrefOf(page: import('@playwright/test').Page, baseURL: string, title: string): Promise<string> {
  await page.goto(baseURL + '/tasks/board?q=' + encodeURIComponent(title));
  await ready(page);
  const links = page.locator('.task-card', { hasText: title }).locator('.task-card-title a');
  expect(await links.count(), `${title} should be on the Board exactly once`).toBe(1);
  return (await links.first().getAttribute('href'))!;
}

test('@weekly-seed two weekly jobs made under the new build', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  for (const title of [A, B]) {
    await page.goto(server.baseURL + '/tasks/new');
    await ready(page);
    await page.fill('#new-task-title', title);
    await page.selectOption('#new-task-stage', 'todo');
    await page.selectOption('#new-task-people', { label: me });
    await page.check('#new-task-repeat');
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
  }
  for (const title of [A, B]) {
    await expect(page.locator('.task-card', { hasText: title }).locator('.stamp.weekly')).toHaveText('WEEKLY');
  }
});

// Run against the PREVIOUS release's app, which has no weekly jobs.
test('@weekly-rolledback the previous release shows them as ordinary jobs, and one can be finished', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const hrefA = await hrefOf(page, server.baseURL, A);
  const hrefB = await hrefOf(page, server.baseURL, B);
  await expect(page.locator('.stamp.weekly')).toHaveCount(0);

  // A opens and can be edited like any job.
  await page.goto(server.baseURL + hrefA);
  await ready(page);
  await expect(page.locator('.task-details-form')).toBeVisible();

  // B is finished, as anyone on that release could.
  const id = /\/tasks\/(\d+)/.exec(hrefB)![1];
  const res = await page.request.post(`${server.baseURL}/tasks/${id}/move`, {
    form: { stage: 'done', to_bottom: '1' },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBe(true);
});

test('@weekly-verify-upgraded both jobs are still weekly after the upgrade', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  for (const title of [A, B]) {
    const href = await hrefOf(page, server.baseURL, title);
    await expect(page.locator('.task-card', { hasText: title }).locator('.stamp.weekly')).toHaveText('WEEKLY');
    await page.goto(server.baseURL + href);
    await ready(page);
    await expect(page.locator('#details-repeat')).toBeChecked();
  }
});
