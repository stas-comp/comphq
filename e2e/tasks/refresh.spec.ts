import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string): Promise<void> {
  await page.fill('#new-task-title', title);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// SPEC gate 2.10: a change made on one computer appears on another's
// board within 60 seconds — the poll interval refresh.js uses.
// page.clock (installed before the page loads, so it governs the very
// setInterval refresh.js schedules) fast-forwards past that interval
// instead of a real 65-second wait, while staying faithful to "within
// 60 seconds": fetches inside the callback still complete over the
// real network, only the timer itself is virtual.
test('gate 2.10: a change on one computer appears on another within 60 seconds', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await otherPage.clock.install();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);

  const title = uniqueName('Seen after 60s');
  await addTask(page, title);

  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(0);
  await otherPage.clock.runFor(61_000);
  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(1);

  await otherContext.close();
});

// SPEC gate 2.10: "...or straight away when that window is clicked
// back into."
test('gate 2.10: a change appears immediately after the window is focused', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);

  const title = uniqueName('Seen on focus');
  await addTask(page, title);

  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(0);
  await otherPage.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(1);

  await otherContext.close();
});

// SPEC gate 2.10: "A card someone is part-way through editing is never
// wiped out by this." A drag in progress marks document.body's
// data-refresh-busy attribute (board.js), which is exactly what this
// asserts against directly, rather than choreographing a real
// mid-gesture drag just to hold that same flag briefly.
test('gate 2.10: a change is held back while the page is busy, and the notice applies it', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);

  await otherPage.evaluate(() => {
    document.body.dataset.refreshBusy = '1';
  });

  const title = uniqueName('Held back while busy');
  await addTask(page, title);

  await otherPage.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(0);
  await expect(otherPage.locator('#refresh-notice')).toBeVisible();

  await otherPage.evaluate(() => {
    delete document.body.dataset.refreshBusy;
  });
  await otherPage.locator('#refresh-notice').click();
  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(1);

  await otherContext.close();
});
