import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { openNewTask } from '../helpers/tasks';

// SPEC gate 4.05: the date field shows and accepts UK order (day first),
// wherever a date is typed. D-61.

test('a task due date typed day first is saved and shown day first', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Dated job');
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.fill('#new-task-due-date', '5/11/2026');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const card = page.locator('.task-card', { hasText: title });
  await expect(card).toHaveCount(1);
  await card.locator('.task-card-title a').click();
  await ready(page);
  await expect(page.locator('#details-due-date')).toHaveValue('05/11/2026');
  await expect(page.locator('#details-due-date')).toHaveAttribute('placeholder', 'dd/mm/yyyy');

  // Edit it, still day first.
  await page.fill('#details-due-date', '06-11-2026');
  await page.click('.task-details-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('#details-due-date')).toHaveValue('06/11/2026');
});

test('a task due date that is not a real day is refused with a plain message and nothing is created', async ({
  page,
  server,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  for (const bad of ['31/02/2026', '09/25/2026', 'next week']) {
    const title = uniqueName('Bad date job');
    await openNewTask(page);
    await page.fill('#new-task-title', title);
    await page.fill('#new-task-due-date', bad);
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
    await expect(page.getByRole('alert')).toContainText('day/month/year');
    await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);
  }
});

test('an event date typed day first lands on the right day, and a bad one is refused', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/calendar/new');
  await ready(page);
  const title = uniqueName('Dated event');
  await page.fill('#event-title', title);
  await page.fill('#event-start-date', '5/12/2026');
  await page.click('.calendar-event-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('h1')).toHaveText(title);
  await expect(page.locator('#event-start-date')).toHaveValue('05/12/2026');

  await page.goto(server.baseURL + '/calendar?month=2026-12');
  await ready(page);
  await expect(page.locator('[data-date="2026-12-05"]', { hasText: title })).toHaveCount(1);

  await page.goto(server.baseURL + '/calendar/new');
  await ready(page);
  await page.fill('#event-title', uniqueName('Bad event'));
  await page.fill('#event-start-date', '30/02/2026');
  await page.click('.calendar-event-form button[type="submit"]');
  await ready(page);
  await expect(page.getByRole('alert')).toContainText('day/month/year');
});
