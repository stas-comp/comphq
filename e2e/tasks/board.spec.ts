import { expect, test } from '../helpers/fixtures';
import { test as testToday } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, stage = 'idea'): Promise<void> {
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// Titles as they appear, top to bottom, within one stage column — used
// to check relative order among a test's own uniquely-named tasks
// without assuming the column is otherwise empty (the shared server
// this test runs against may have other tasks in it already).
async function columnOrder(page: Page, stage: string): Promise<string[]> {
  return page.locator(`.task-column[data-stage="${stage}"] .task-card-title`).allTextContents();
}

function relativeOrder(all: string[], wanted: string[]): string[] {
  return all.filter((title) => wanted.includes(title));
}

// SPEC gate 2.01: the Board shows four columns, in order.
test('gate 2.01: Board shows Ideas, To do, In progress, Done in order', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const headings = await page.locator('.task-column h2').allTextContents();
  expect(headings.length).toBe(4);
  expect(headings[0]).toContain('Ideas');
  expect(headings[1]).toContain('To do');
  expect(headings[1]).toContain('Top = most important');
  expect(headings[2]).toContain('In progress');
  expect(headings[3]).toContain('Done');
});

// SPEC gate 2.02: adding a task with just a title puts it at the bottom
// of the chosen column for everyone.
test('gate 2.02: a title-only task is visible in a second context after reload', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Fix the printer');
  await page.fill('#new-task-title', title);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(1);
  // Defaults: bottom of Ideas (the default starting column), size M
  // isn't shown on the card itself (SPEC A6 only shows title, people, due
  // date on a card), but the task must land in Ideas.
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: title })).toHaveCount(1);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(1);
  await otherContext.close();
});

test('gate 2.02: a task can be added directly into a chosen column', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Plan the offsite');
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', 'doing');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await expect(page.locator('.task-column[data-stage="doing"] .task-card', { hasText: title })).toHaveCount(1);
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: title })).toHaveCount(0);
});

// SPEC gate 2.03 (assignees/due date part) and 2.26 (size default): a
// task with people and a due date shows both on its card.
test('gate 2.03: a card shows its assigned people and due date', async ({ page, server }) => {
  const you = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Team task');
  await page.fill('#new-task-title', title);
  await page.fill('#new-task-due-date', '2099-01-01');
  await page.selectOption('#new-task-people', { label: you });
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const card = page.locator('.task-card', { hasText: title });
  await expect(card).toHaveCount(1);
  await expect(card.locator('.task-avatar')).toHaveCount(1);
  await expect(card.locator('.task-avatar')).toHaveAttribute('title', you);
  await expect(card).toContainText('2099-01-01');
  await expect(card.locator('.task-overdue-stamp')).toHaveCount(0);
});

// SPEC gate 2.03 (OVERDUE part): an unfinished task past its due date
// shows an OVERDUE stamp; a task in Done never does, regardless of date.
// @fresh: needs its own server with a fixed COMPHQ_TEST_TODAY.
testToday.use({ today: '2026-09-19' });
testToday('@fresh gate 2.03: an unfinished task past its due date shows OVERDUE', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const overdueTitle = uniqueName('Overdue task');
  await page.fill('#new-task-title', overdueTitle);
  await page.fill('#new-task-due-date', '2026-09-01');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const overdueCard = page.locator('.task-card', { hasText: overdueTitle });
  await expect(overdueCard.locator('.task-overdue-stamp')).toHaveText('OVERDUE');

  const futureTitle = uniqueName('Future task');
  await page.fill('#new-task-title', futureTitle);
  await page.fill('#new-task-due-date', '2026-12-01');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: futureTitle }).locator('.task-overdue-stamp')).toHaveCount(0);

  const doneTitle = uniqueName('Done but past due');
  await page.fill('#new-task-title', doneTitle);
  await page.fill('#new-task-due-date', '2026-09-01');
  await page.selectOption('#new-task-stage', 'done');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: doneTitle }).locator('.task-overdue-stamp')).toHaveCount(0);
});

// SPEC gate 2.04 (button part): Move up/Move down reorder within a
// stage, consistent on reload from a second context.
test('gate 2.04: Move up and Move down reorder cards, visible in a second context', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const stage = 'todo';
  const a = uniqueName('A');
  const b = uniqueName('B');
  const c = uniqueName('C');
  await addTask(page, a, stage);
  await addTask(page, b, stage);
  await addTask(page, c, stage);

  expect(relativeOrder(await columnOrder(page, stage), [a, b, c])).toEqual([a, b, c]);

  // Move C up one slot: swaps with B -> [A, C, B].
  const cardC = page.locator('.task-card', { hasText: c });
  await cardC.locator('.inline-form', { hasText: 'Move up' }).locator('button').click();
  await ready(page);
  expect(relativeOrder(await columnOrder(page, stage), [a, b, c])).toEqual([a, c, b]);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  expect(relativeOrder(await columnOrder(otherPage, stage), [a, b, c])).toEqual([a, c, b]);
  await otherContext.close();
});

// SPEC gate 2.05: a card moved into a different column by button lands
// at the bottom of that column.
test('gate 2.05: Move to… lands a card at the bottom of the target column', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const existing = uniqueName('Existing');
  await addTask(page, existing, 'todo');
  const moving = uniqueName('Moving');
  await addTask(page, moving, 'idea');

  const movingCard = page.locator('.task-card', { hasText: moving });
  await movingCard.locator('.task-move-to-form select').selectOption('todo');
  await movingCard.locator('.task-move-to-form').locator('button').click();
  await ready(page);

  expect(relativeOrder(await columnOrder(page, 'todo'), [existing, moving])).toEqual([existing, moving]);
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: moving })).toHaveCount(0);
});

// Keyboard-only E2E (PLAN.md P2-02): move a task by buttons with no mouse.
test('gate 2.04: a task can be moved with keyboard only', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const stage = 'todo';
  const a = uniqueName('KbA');
  const b = uniqueName('KbB');
  await addTask(page, a, stage);
  await addTask(page, b, stage);
  expect(relativeOrder(await columnOrder(page, stage), [a, b])).toEqual([a, b]);

  const moveUpButton = page.locator('.task-card', { hasText: b }).locator('.inline-form', { hasText: 'Move up' }).locator('button');
  await moveUpButton.focus();
  await page.keyboard.press('Enter');
  await ready(page);

  expect(relativeOrder(await columnOrder(page, stage), [a, b])).toEqual([b, a]);
});
