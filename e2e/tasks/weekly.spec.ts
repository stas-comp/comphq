import { expect, test } from '@playwright/test';
import { newDataDir, startServerAt } from '../helpers/server-at';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 7.30-7.37 end to end (B13.5, D-95). Each test has a server of its
// own with a fixed "today", and lets Saturday pass by starting the app again on
// the same data with a later date: the reset happens the first time Comp HQ is
// used on or after the Saturday (gate 7.35).
// 2026-10-07 is a Wednesday; its Saturday is 10 Oct, then 17 and 24.

test.describe.configure({ timeout: 90_000 });

async function makeWeekly(page: Page, base: string, title: string, meName: string, opts: { due?: string } = {}) {
  await page.goto(base + '/tasks/new');
  await ready(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', 'todo');
  await page.selectOption('#new-task-people', { label: meName });
  if (opts.due) await page.fill('#new-task-due-date', opts.due);
  await page.check('#new-task-repeat');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

const boardCard = (page: Page, title: string) => page.locator('.task-card', { hasText: title });

async function jobHref(page: Page, base: string, title: string): Promise<string> {
  await page.goto(base + '/tasks/board?q=' + encodeURIComponent(title));
  await ready(page);
  return (await boardCard(page, title).locator('.task-card-title a').first().getAttribute('href'))!;
}

test('@fresh gates 7.30, 7.31: the tick-box is in the form, sets This Saturday, and a WEEKLY stamp shows on every screen', async ({ page }) => {
  const dataDir = newDataDir('weekly-a');
  const server = await startServerAt(dataDir, '2026-10-07');
  try {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const title = uniqueName('Clean the kitchen');

    await openNewTask(page);
    await expect(page.getByLabel('Repeats every week (back in To do each Saturday)')).not.toBeChecked();
    await makeWeekly(page, server.baseURL, title, me);

    // The job's own page: ticked, and the empty due date became This Saturday.
    const href = await jobHref(page, server.baseURL, title);
    await page.goto(server.baseURL + href);
    await ready(page);
    await expect(page.locator('#details-repeat')).toBeChecked();
    await expect(page.locator('#details-due-date')).toHaveValue('10/10/2026');

    // WEEKLY on the Board, My jobs, Team and the Briefing.
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    await expect(boardCard(page, title).locator('.stamp.weekly')).toHaveText('WEEKLY');
    await page.goto(server.baseURL + '/tasks');
    await ready(page);
    await expect(page.locator('.myjobs-upnext-list .team-task', { hasText: title }).locator('.stamp.weekly')).toHaveText('WEEKLY');
    await page.goto(server.baseURL + '/tasks/team');
    await ready(page);
    await expect(page.locator('.team-lane.me .team-task', { hasText: title }).locator('.stamp.weekly')).toHaveText('WEEKLY');
    await page.goto(server.baseURL + '/briefing');
    await ready(page);
    await expect(page.locator('.briefing-card', { hasText: title }).locator('.stamp.weekly')).toHaveText('WEEKLY');

    // An ordinary job has none.
    const plain = uniqueName('Ordinary job');
    await openNewTask(page);
    await page.fill('#new-task-title', plain);
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
    await expect(boardCard(page, plain).locator('.stamp.weekly')).toHaveCount(0);
  } finally {
    await server.stop();
  }
});

test('@fresh gates 7.31, 7.32, 7.36: finished, it is back in To do the next Saturday, due that day, steps unticked, in the Briefing, with History', async ({ page }) => {
  const dataDir = newDataDir('weekly-b');
  let server = await startServerAt(dataDir, '2026-10-07');
  try {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const title = uniqueName('Wash the cups');
    await makeWeekly(page, server.baseURL, title, me, { due: '10/10/2026' });
    const href = await jobHref(page, server.baseURL, title);

    // A step, ticked; then the job finished from My jobs.
    await page.goto(server.baseURL + href);
    await ready(page);
    await page.getByPlaceholder('Add a step').fill('Dry them');
    await page.keyboard.press('Enter');
    await page.getByRole('checkbox', { name: 'Dry them', exact: true }).check();
    await expect(page.locator('#steps .steps-count')).toHaveText('All 1 done');
    await page.goto(server.baseURL + '/tasks');
    await ready(page);
    await page.locator('.myjobs-upnext-list .team-task', { hasText: title }).getByRole('button', { name: 'Done ✓' }).click();
    await ready(page);
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: title })).toHaveCount(1);

    // Friday 9 Oct: the app is used, and nothing happens yet.
    await server.stop();
    server = await startServerAt(dataDir, '2026-10-09');
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: title })).toHaveCount(1);

    // Saturday 10 Oct.
    await server.stop();
    server = await startServerAt(dataDir, '2026-10-10');
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    const card = page.locator('.task-column[data-stage="todo"] .task-card', { hasText: title });
    await expect(card).toHaveCount(1);
    await expect(card.locator('.task-card-due')).toContainText('Sat 10 Oct');
    // At the bottom of To do, among my own jobs at least.
    const todo = await page.locator('.task-column[data-stage="todo"] .task-card-title').allInnerTexts();
    expect(todo[todo.length - 1]).toContain(title);

    // Its step is still there, unticked, and the job is in that Saturday's Briefing, under Must be done today.
    await page.goto(server.baseURL + href);
    await ready(page);
    await expect(page.getByRole('checkbox', { name: 'Dry them', exact: true })).not.toBeChecked();
    await page.goto(server.baseURL + '/briefing');
    await ready(page);
    await expect(page.locator('.briefing-section', { hasText: 'Must be done today' }).locator('.briefing-card', { hasText: title })).toHaveCount(1);

    // History: "Comp HQ put this back…", and Comp HQ is not a person anywhere.
    await page.goto(server.baseURL + href);
    await ready(page);
    await expect(page.locator('.task-activity-list')).toContainText('Comp HQ put this back in To do for Sat 10 Oct (weekly)');
    await page.goto(server.baseURL + '/tasks/new');
    await expect(page.locator('#new-task-people option', { hasText: 'Comp HQ' })).toHaveCount(0);
    await page.goto(server.baseURL + '/settings/people');
    await expect(page.locator('main')).not.toContainText('Comp HQ put');

    // Finished again, and two Saturdays pass with the app switched off: one reset, due the latest.
    await page.goto(server.baseURL + '/tasks');
    await ready(page);
    await page.locator('.myjobs-upnext-list .team-task', { hasText: title }).getByRole('button', { name: 'Done ✓' }).click();
    await ready(page);
    await server.stop();
    server = await startServerAt(dataDir, '2026-10-24');
    await page.goto(server.baseURL + href);
    await ready(page);
    await expect(page.locator('#details-due-date')).toHaveValue('24/10/2026');
    const resets = await page.locator('.task-activity-list li', { hasText: 'Comp HQ put this back' }).allInnerTexts();
    expect(resets.length).toBe(2); // 10 Oct, and 24 Oct (17 Oct was missed, and counts once)
    await expect(page.locator('.task-activity-list')).not.toContainText('Sat 17 Oct');
  } finally {
    await server.stop();
  }
});

test('@fresh gates 7.33, 7.34, 7.37: unfinished stays and reads OVERDUE; a weekly job never leaves Done; unticking makes it ordinary', async ({ page }) => {
  const dataDir = newDataDir('weekly-c');
  let server = await startServerAt(dataDir, '2026-10-07');
  try {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const stays = uniqueName('Left unfinished');
    const finished = uniqueName('Finished and kept');
    const ordinary = uniqueName('No longer weekly');
    for (const t of [stays, finished, ordinary]) await makeWeekly(page, server.baseURL, t, me, { due: '10/10/2026' });
    const ordinaryHref = await jobHref(page, server.baseURL, ordinary);

    for (const t of [finished, ordinary]) {
      await page.goto(server.baseURL + '/tasks');
      await ready(page);
      await page.locator('.myjobs-upnext-list .team-task', { hasText: t }).getByRole('button', { name: 'Done ✓' }).click();
      await ready(page);
    }
    // Untick the box on the finished one (7.37).
    await page.goto(server.baseURL + ordinaryHref);
    await ready(page);
    await page.uncheck('#details-repeat');
    await page.click('.task-details-form button[type="submit"]');
    await ready(page);

    // Two weeks on (17 Oct).
    await server.stop();
    server = await startServerAt(dataDir, '2026-10-17');
    await page.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent('Left unfinished'));
    await ready(page);
    // The unfinished one is where it was, with its old date, and OVERDUE: one card, never doubled.
    const unfinished = page.locator('.task-card', { hasText: stays });
    await expect(unfinished).toHaveCount(1);
    await expect(unfinished.locator('.stamp-overdue')).toHaveText('OVERDUE');
    await expect(unfinished.locator('.task-card-due')).toContainText('Sat 10 Oct');

    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    // The one that was finished is back in To do; the one made ordinary stays in Done.
    await expect(page.locator('.task-column[data-stage="todo"] .task-card', { hasText: finished })).toHaveCount(1);
    await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: ordinary })).toHaveCount(1);

    // A weekly job stays in the Done column however long it sits there, and never goes to Finished tasks.
    await page.goto(server.baseURL + '/tasks');
    await ready(page);
    await page.locator('.myjobs-upnext-list .team-task', { hasText: finished }).getByRole('button', { name: 'Done ✓' }).click();
    await ready(page);
    await server.stop();
    server = await startServerAt(dataDir, '2026-10-21'); // before the next Saturday; still Done
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: finished })).toHaveCount(1);
    await page.goto(server.baseURL + '/tasks/finished');
    await expect(page.locator('main')).not.toContainText(finished);
  } finally {
    await server.stop();
  }
});
