import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { openNewTask, openTaskPage } from '../helpers/tasks';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, stage = 'idea'): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

async function taskIDFromDetailsLink(page: Page, title: string): Promise<string> {
  const href = await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').getAttribute('href');
  const match = href?.match(/\/tasks\/(\d+)/);
  if (!match) throw new Error(`couldn't find a task id for ${title}`);
  return match[1];
}

// Backdates done_at without waiting on the real 14-day window (SPEC
// gate 2.08) — the same test-only-route pattern already used for
// backups' set-last-backup-at.
async function setTaskDoneDaysAgo(server: { baseURL: string }, page: Page, taskID: string, daysAgo: number): Promise<void> {
  const res = await page.request.post(server.baseURL + '/__test/tasks/set-done-at', {
    form: { task_id: taskID, days_ago: String(daysAgo) },
    headers: { origin: server.baseURL },
  });
  expect(res.ok()).toBeTruthy();
}

// SPEC gate 2.08: a card in Done for more than 14 days no longer shows
// on the board; it appears in Finished tasks, and Reopen returns it to
// the bottom of To do.
test('gate 2.08: a task done over 14 days ago moves to Finished tasks, and Reopen returns it to To do', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  await addTask(page, uniqueName('Existing in To do'), 'todo');
  const title = uniqueName('Old finished task');
  await addTask(page, title, 'done');
  const id = await taskIDFromDetailsLink(page, title);

  await setTaskDoneDaysAgo(server, page, id, 15);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);

  await page.goto(server.baseURL + '/tasks/finished');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(1);

  await page.locator('.task-card', { hasText: title }).locator('button', { hasText: 'Reopen' }).click();
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  const todoTitles = await page.locator('.task-column[data-stage="todo"] .task-card-title').allTextContents();
  expect(todoTitles[todoTitles.length - 1]).toBe(title);
});

// SPEC gate 2.09: "Remove task" hides it from the board and lists it in
// Removed tasks, where it can be restored.
test('gate 2.09: Remove hides a task from the board and lists it in Removed tasks; Restore brings it back', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Removable task');
  await addTask(page, title, 'idea');
  await openTaskPage(page, title);

  await page.locator('.task-remove-form button[type="submit"]').click();
  await ready(page);
  await expect(page).toHaveURL(/\/tasks\/board$/);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);

  await page.goto(server.baseURL + '/tasks/removed');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(1);

  await page.locator('.task-card', { hasText: title }).locator('button', { hasText: 'Restore' }).click();
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(1);
});

test('standard page checks for Finished tasks and Removed tasks', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  await page.goto(server.baseURL + '/tasks/finished');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);

  await page.goto(server.baseURL + '/tasks/removed');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
