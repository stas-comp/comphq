import { expect, test } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

test.use({ today: '2026-09-19' });

async function addTask(page: Page, baseURL: string, title: string, due: string): Promise<void> {
  await page.goto(baseURL + '/tasks/board');
  await ready(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', 'todo');
  await page.fill('#new-task-due-date', due);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

async function addEvent(page: Page, baseURL: string, title: string, startDate: string): Promise<void> {
  await page.goto(baseURL + '/calendar/new');
  await ready(page);
  await page.fill('#event-title', title);
  await page.fill('#event-start-date', startDate);
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);
}

// A second "computer": its own browser context, with the clock installed
// before the page loads so refresh.js's 60-second timer can be
// fast-forwarded (the same technique gate 2.10's tests use).
async function openSecondComputer(browser: import('@playwright/test').Browser, baseURL: string) {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.clock.install();
  await signInAsNewPerson(page, baseURL, '/briefing');
  await ready(page);
  return { context, page };
}

// SPEC gate 3.12: a task due this week, added on one computer, appears
// on another's Briefing within 60 seconds.
test('@fresh gate 3.12: a new task appears on another computer within 60 seconds', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/briefing');
  const other = await openSecondComputer(browser, server.baseURL);

  await addTask(page, server.baseURL, 'Added elsewhere', '2026-09-22');

  await expect(other.page.locator('.briefing-card', { hasText: 'Added elsewhere' })).toHaveCount(0);
  await other.page.clock.runFor(61_000);
  await expect(other.page.locator('.briefing-card', { hasText: 'Added elsewhere' })).toHaveCount(1);

  await other.context.close();
});

// ...and straight away when that window is clicked back into.
test('@fresh gate 3.12: a new task appears straight away when the window is focused', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/briefing');
  const other = await openSecondComputer(browser, server.baseURL);

  await addTask(page, server.baseURL, 'Added elsewhere', '2026-09-22');

  await other.page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(other.page.locator('.briefing-card', { hasText: 'Added elsewhere' })).toHaveCount(1);

  await other.context.close();
});

// The same for an event edit: a renamed event, and a newly added one.
test('@fresh gate 3.12: an event edit and a new event appear on another computer', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/briefing');
  await addEvent(page, server.baseURL, 'Exams', '2026-09-21');
  const other = await openSecondComputer(browser, server.baseURL);
  await expect(other.page.locator('.briefing-card', { hasText: 'Exams' })).toHaveCount(1);

  // Rename it from the first computer.
  await page.goto(server.baseURL + '/briefing');
  await ready(page);
  await page.getByRole('link', { name: 'Exams', exact: true }).click();
  await ready(page);
  await page.fill('#event-title', 'Exams in the hall');
  await page.locator('.calendar-event-form button[type="submit"]').click();
  await ready(page);

  await other.page.clock.runFor(61_000);
  await expect(other.page.locator('.briefing-card', { hasText: 'Exams in the hall' })).toHaveCount(1);

  await addEvent(page, server.baseURL, 'Open evening', '2026-09-24');
  await other.page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(other.page.locator('.briefing-card', { hasText: 'Open evening' })).toHaveCount(1);

  await other.context.close();
});
