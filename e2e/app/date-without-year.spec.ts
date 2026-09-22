import { expect, test } from '../helpers/fixtures-today';
import { ready } from '../helpers/ready';
import { openNewTask, openTaskPage } from '../helpers/tasks';
import { signInAsNewPerson } from '../helpers/people';
import { uniqueName } from '../helpers/unique-name';

// SPEC gates 6.18, 6.19 (D-82): a date typed without its year means the
// nearest such date to today, and everything D-61 already refused is
// still refused. Every test here is @fresh: it needs its own server with
// its own COMPHQ_TEST_TODAY.

test.describe('on 22 September 2026', () => {
  test.use({ today: '2026-09-22' });

  test('@fresh gate 6.18: a task due date typed without its year picks the nearest such date', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await ready(page);
    const title = uniqueName('Yearless job');
    await openNewTask(page);
    await page.fill('#new-task-title', title);
    await page.fill('#new-task-due-date', '25/9');
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);

    await openTaskPage(page, title);
    await expect(page.locator('#details-due-date')).toHaveValue('25/09/2026');
  });

  test('@fresh gate 6.18: "." and "-" separators without a year both resolve the same way', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar/new');
    await ready(page);
    const title = uniqueName('Yearless event');
    await page.fill('#event-title', title);
    await page.fill('#event-start-date', '1.7');
    await page.click('.calendar-event-form button[type="submit"]');
    await ready(page);
    await expect(page.locator('#event-start-date')).toHaveValue('01/07/2026'); // earlier this year, not next July
  });

  test('@fresh gate 6.19: a short date that is not a real day is still refused with the same plain message', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await ready(page);
    const title = uniqueName('Bad short date job');
    await openNewTask(page);
    await page.fill('#new-task-title', title);
    await page.fill('#new-task-due-date', '31/9');
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
    await expect(page.getByRole('alert')).toContainText('day/month/year');
    await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);
  });

  test.describe('with JavaScript off', () => {
    test.use({ javaScriptEnabled: false });

    test('@fresh gate 6.18: the server applies the same nearest-year rule with no script at all', async ({ page, server }) => {
      await signInAsNewPerson(page, server.baseURL, '/tasks/board');
      await ready(page);
      const title = uniqueName('No script yearless job');
      await openNewTask(page);
      await page.fill('#new-task-title', title);
      await page.fill('#new-task-due-date', '25-9');
      await page.click('.add-task-form button[type="submit"]');
      await ready(page);

      await openTaskPage(page, title);
      await expect(page.locator('#details-due-date')).toHaveValue('25/09/2026');
    });
  });
});

test.describe('on 20 December 2026', () => {
  test.use({ today: '2026-12-20' });

  test('@fresh gate 6.18: a yearless date just after New Year resolves to next year', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar/new');
    await ready(page);
    const title = uniqueName('New year event');
    await page.fill('#event-title', title);
    await page.fill('#event-start-date', '5/1');
    await page.click('.calendar-event-form button[type="submit"]');
    await ready(page);
    await expect(page.locator('#event-start-date')).toHaveValue('05/01/2027');
  });
});
