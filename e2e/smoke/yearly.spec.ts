import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// Yearly jobs across an upgrade and a rollback (SPEC gate 8.53, B14.4, D-108): two
// waiting yearly jobs are made under the new build (due about 18 months from now, so
// they are waiting whatever day this runs); the container is rolled back to the
// previous release, which has never heard of yearly jobs and shows them as ordinary
// jobs in To do with their future due dates; one of them is saved there from its edit
// form, as anyone on that release could; upgraded again, both are yearly again and
// both are waiting again.
//
// Tagged @yearly-seed / @yearly-rolledback / @yearly-verify-upgraded (none a
// substring of another) for container-test.sh's phases. Titles are fixed because a
// later phase has to find them; the person is not. Only plain endpoints are used by
// the rolled-back step, so the same code runs against the previous release.
const A = 'Smoke Yearly Job A';
const B = 'Smoke Yearly Job B';

function dueInDays(days: number): string {
  const d = new Date(Date.now() + days * 86_400_000);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()}`;
}

async function boardHref(page: Page, baseURL: string, title: string): Promise<string | null> {
  await page.goto(baseURL + '/tasks/board?q=' + encodeURIComponent(title));
  await ready(page);
  const links = page.locator('.task-card', { hasText: title }).locator('.task-card-title a');
  const n = await links.count();
  expect(n, `${title} should be on the Board at most once`).toBeLessThanOrEqual(1);
  return n === 1 ? await links.first().getAttribute('href') : null;
}

async function yearlyPageHas(page: Page, baseURL: string, title: string): Promise<boolean> {
  await page.goto(baseURL + '/tasks/yearly');
  await ready(page);
  return (await page.locator('.yearly-job', { hasText: title }).count()) === 1;
}

test('@yearly-seed two waiting yearly jobs made under the new build', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  for (const title of [A, B]) {
    const res = await page.request.post(server.baseURL + '/tasks', {
      form: { title, stage: 'todo', due_date: dueInDays(540), repeat: 'yearly', notice_weeks: '6' },
      headers: { origin: server.baseURL },
      maxRedirects: 0,
    });
    expect(res.status(), title).toBeLessThan(400);
  }
  for (const title of [A, B]) {
    expect(await boardHref(page, server.baseURL, title), `${title} is waiting, so not on the Board`).toBeNull();
    expect(await yearlyPageHas(page, server.baseURL, title)).toBe(true);
  }
});

// "1.5.0" >= "1.6.0": compares dotted release numbers part by part.
function atLeast(version: string, min: string): boolean {
  const a = version.split('.').map(Number);
  const b = min.split('.').map(Number);
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const d = (a[i] ?? 0) - (b[i] ?? 0);
    if (d !== 0) return d > 0;
  }
  return true;
}

// Run against the PREVIOUS release's app. container-test.sh passes its version as
// COMPHQ_PREV_VERSION. A release before 1.6.0 has never heard of yearly jobs: it
// shows them as ordinary jobs, waiting ones in To do with their future dates, and
// a job there can still be opened and saved. A later release still hides them.
test('@yearly-rolledback the previous release shows them as ordinary jobs, and one is saved there', async ({ page, server }) => {
  const prev = process.env.COMPHQ_PREV_VERSION ?? '';
  const prevKnowsYearly = prev === '' || atLeast(prev, '1.6.0');
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const hrefA = await boardHref(page, server.baseURL, A);
  const hrefB = await boardHref(page, server.baseURL, B);
  if (prevKnowsYearly) {
    expect(hrefA).toBeNull();
    expect(hrefB).toBeNull();
    return;
  }
  expect(hrefA, `${A} shows as an ordinary job in To do`).not.toBeNull();
  expect(hrefB, `${B} shows as an ordinary job in To do`).not.toBeNull();
  await expect(page.locator('.task-column[data-stage="todo"] .task-card', { hasText: B })).toHaveCount(1);

  // B is opened and saved from its edit form, as anyone on that release could.
  await page.goto(server.baseURL + hrefB!);
  await ready(page);
  await expect(page.locator('.task-details-form')).toBeVisible();
  const idB = /\/tasks\/(\d+)/.exec(hrefB!)![1];
  const saved = await page.request.post(`${server.baseURL}/tasks/${idB}`, {
    form: { title: B, notes: 'Edited while rolled back', size: 'M', stage: 'todo', due_date: dueInDays(540) },
    headers: { origin: server.baseURL },
    maxRedirects: 0,
  });
  expect(saved.status()).toBeLessThan(400);
});

test('@yearly-verify-upgraded both jobs are yearly again and waiting again after the upgrade', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  for (const title of [A, B]) {
    expect(await boardHref(page, server.baseURL, title), `${title} is waiting again, so not on the Board`).toBeNull();
    expect(await yearlyPageHas(page, server.baseURL, title), `${title} is on the Yearly jobs page`).toBe(true);
  }
});
