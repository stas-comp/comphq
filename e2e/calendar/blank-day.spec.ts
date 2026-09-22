import { expect, test } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// Every test here is @fresh: it needs its own server with its own
// COMPHQ_TEST_TODAY, so the month grid (and which days fall in/outside
// September 2026) is exactly known.

async function addEvent(page: Page, baseURL: string, fields: Record<string, string>): Promise<void> {
  await page.goto(baseURL + '/calendar/new');
  await ready(page);
  for (const [name, value] of Object.entries(fields)) {
    await page.locator(`[name="${name}"]`).fill(value);
  }
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);
}

async function addTask(page: Page, title: string, dueDate: string): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.fill('#new-task-due-date', dueDate);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// Clicks the lower, blank part of a day cell — away from the day number
// (top left) and any chip lines beneath it — so this never accidentally
// lands on a link even when SPEC gate 6.22's "+" mark is showing.
async function clickBlankPartOf(page: Page, cellLocator: ReturnType<Page['locator']>): Promise<void> {
  await cellLocator.scrollIntoViewIfNeeded();
  const box = (await cellLocator.boundingBox())!;
  await page.mouse.click(box.x + box.width - 10, box.y + box.height - 8);
}

test.describe('gates 6.21-6.23: a blank day adds an event', () => {
  test.use({ today: '2026-09-16' });

  test('@fresh clicking the blank part of a current-month day starts a new event there', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-09');
    await ready(page);
    const cell = page.locator('.calendar-day[data-date="2026-09-24"]');
    await expect(cell).not.toHaveClass(/calendar-day-outside/);
    await clickBlankPartOf(page, cell);
    await ready(page);
    await expect(page.locator('#event-start-date')).toHaveValue('24/09/2026');
  });

  test('@fresh clicking the blank part of a greyed next-month day also starts a new event there', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-09');
    await ready(page);
    // September 2026's grid runs to 3 Oct 2026 (grid_test.go); the 1st is
    // outside the month and greyed.
    const cell = page.locator('.calendar-day[data-date="2026-10-01"]');
    await expect(cell).toHaveClass(/calendar-day-outside/);
    await clickBlankPartOf(page, cell);
    await ready(page);
    await expect(page.locator('#event-start-date')).toHaveValue('01/10/2026');
  });

  test('@fresh clicking an event chip still opens the event, and a task chip still opens the task, not a new event', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-09');
    await ready(page);
    const eventTitle = uniqueName('Existing event');
    const taskTitle = uniqueName('Existing task');
    await addEvent(page, server.baseURL, { title: eventTitle, start_date: '20/9/2026' });
    await addTask(page, taskTitle, '20/9/2026');

    await page.goto(server.baseURL + '/calendar?month=2026-09');
    await ready(page);
    const cell = page.locator('.calendar-day[data-date="2026-09-20"]');

    await cell.locator('.calendar-chip', { hasText: eventTitle }).click();
    await ready(page);
    await expect(page.locator('h1')).toHaveText(eventTitle);

    await page.goto(server.baseURL + '/calendar?month=2026-09');
    await ready(page);
    await cell.locator('.calendar-chip', { hasText: taskTitle }).click();
    await ready(page);
    await expect(page.locator('h1')).toHaveText(taskTitle);
  });

  test('@fresh the keyboard route: the day number is a link named "Add an event on …"', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-09');
    await ready(page);
    const link = page.getByRole('link', { name: 'Add an event on Thursday 24 September' });
    await expect(link).toBeVisible();
    await link.focus();
    await expect(link).toBeFocused();
    await page.keyboard.press('Enter');
    await ready(page);
    await expect(page.locator('#event-start-date')).toHaveValue('24/09/2026');
  });

  test('@fresh today\'s day number link is also named as today', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-09');
    await ready(page);
    await expect(page.getByRole('link', { name: 'Add an event on Wednesday 16 September (today)' })).toBeVisible();
  });

  test('@fresh axe on the month view with the blank-day behaviour in place', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-09');
    await ready(page);
    await axeCheck(page);
  });

  test.describe('with JavaScript off', () => {
    test.use({ javaScriptEnabled: false });

    test('@fresh the day number link still works with no script', async ({ page, server }) => {
      await signInAsNewPerson(page, server.baseURL, '/calendar?month=2026-09');
      await ready(page);
      await page.locator('.calendar-day[data-date="2026-09-24"] .calendar-day-number').click();
      await ready(page);
      await expect(page.locator('#event-start-date')).toHaveValue('24/09/2026');
    });
  });
});
