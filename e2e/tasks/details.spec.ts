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

async function openDetails(page: Page, title: string): Promise<void> {
  await openTaskPage(page, title);
}

// SPEC gate 2.06: opening a card allows editing every detail, and its
// Activity shows who created it, moved it, changed its people, or
// changed its due date. Gate 2.26 covers the size part of the same
// Activity requirement, plus the size showing on the board card.
test('gate 2.06: opening a card allows editing every detail, and Activity records what changed', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Write the report');
  await addTask(page, title, 'idea');
  await openDetails(page, title);

  await expect(page).toHaveURL(/\/tasks\/\d+$/);
  await expect(page.locator('.task-activity-list')).toContainText(`${name} created this`);

  await page.fill('#details-title', title + ' (updated)');
  await page.fill('#details-notes', 'Draft the Q3 report.');
  await page.selectOption('#details-size', 'L');
  await page.selectOption('#details-stage', 'doing');
  await page.fill('#details-due-date', '2026-09-25');
  await page.selectOption('#details-people', { label: name });
  await page.click('.task-details-form button[type="submit"]');
  await ready(page);

  await expect(page.locator('#details-title')).toHaveValue(title + ' (updated)');
  await expect(page.locator('#details-notes')).toHaveValue('Draft the Q3 report.');
  await expect(page.locator('#details-size')).toHaveValue('L');
  await expect(page.locator('#details-stage')).toHaveValue('doing');
  await expect(page.locator('#details-due-date')).toHaveValue('25/09/2026');

  const activityText = await page.locator('.task-activity-list').innerText();
  expect(activityText).toContain(`${name} created this`);
  expect(activityText).toContain(`${name} edited this`);
  expect(activityText).toContain(`${name} moved this to In progress`);
  expect(activityText).toContain(`${name} changed the size to Large`);
  expect(activityText).toContain(`${name} changed the due date to`);
  expect(activityText).toContain(`${name} assigned ${name}`);

  // Unassigning is also recorded (SPEC gate 2.06's "changed its people").
  await page.selectOption('#details-people', []);
  await page.click('.task-details-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('.task-activity-list')).toContainText(`${name} unassigned ${name}`);
});

test('gate 2.26: cards on the board show their size', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Sized task');
  await addTask(page, title, 'idea');
  await expect(page.locator('.task-card', { hasText: title }).locator('.task-card-size')).toHaveText('Medium');

  await openDetails(page, title);
  await page.selectOption('#details-size', 'S');
  await page.click('.task-details-form button[type="submit"]');
  await ready(page);
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title }).locator('.task-card-size')).toHaveText('Small');
});

test('the task details page works with keyboard only', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Keyboard task');
  await addTask(page, title, 'idea');

  await openTaskPage(page, title); // the page itself, by its address (the title link opens the window)
  await expect(page).toHaveURL(/\/tasks\/\d+$/);

  const notesField = page.locator('#details-notes');
  await notesField.focus();
  await page.keyboard.type('Typed with a keyboard only.');

  const saveButton = page.locator('.task-details-form button[type="submit"]');
  await saveButton.focus();
  await page.keyboard.press('Enter');
  await ready(page);

  await expect(page.locator('#details-notes')).toHaveValue('Typed with a keyboard only.');
});

test('standard page checks for the task details page', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Checked task');
  await addTask(page, title, 'idea');
  await openDetails(page, title);

  await axeCheck(page);
  await expectNoSideScroll(page);
});
