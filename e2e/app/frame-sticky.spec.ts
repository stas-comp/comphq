import { expect, test } from '../helpers/fixtures';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// Enough cards that the Board is taller than the viewport at every size
// this file tests, without going near the test library's own scale (that's
// the speed suite's job).
const CARD_COUNT = 25;

async function seedBoard(page: Page, baseURL: string, prefix: string): Promise<void> {
  for (let i = 0; i < CARD_COUNT; i++) {
    const res = await page.request.post(baseURL + '/tasks', {
      form: { title: `${prefix} ${i}` },
      headers: { origin: baseURL },
    });
    if (!res.ok()) throw new Error(`seed task ${i} failed: ${res.status()}`);
  }
}

async function scrollToBottom(page: Page): Promise<void> {
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  // The sticky elements reposition on scroll, not on a timer; give layout
  // one frame before reading positions back.
  await page.waitForFunction(
    () => Math.ceil(window.scrollY + window.innerHeight) >= document.documentElement.scrollHeight,
  );
}

// SPEC gate 6.01: with the Board scrolled to the very bottom, Settings and
// the version number (the sidebar's foot) are still on screen.
for (const size of [{ width: 1024, height: 700 }, { width: 1920, height: 1080 }]) {
  test(`6.01: sidebar foot stays in view at bottom of a long Board (${size.width}x${size.height})`, async ({ page, server }) => {
    await page.setViewportSize(size);
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await seedBoard(page, server.baseURL, uniqueName('Sticky'));
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);

    await scrollToBottom(page);

    await expect(page.locator('.sidebar nav a[href="/settings"]')).toBeInViewport();
    await expect(page.locator('.sidebar .ver')).toBeInViewport();
  });
}

// SPEC gate 6.02: the top bar (search box, "You: name * Change") also
// stays at y=0 while the page scrolls.
test('6.02: top bar stays pinned to the top while the page scrolls', async ({ page, server }) => {
  await page.setViewportSize({ width: 1024, height: 700 });
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await seedBoard(page, server.baseURL, uniqueName('Sticky'));
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);

  const before = await page.locator('.topbar').boundingBox();
  expect(before?.y).toBe(0);

  await scrollToBottom(page);

  const after = await page.locator('.topbar').boundingBox();
  expect(after?.y).toBe(0);
  await expect(page.locator('#search-box')).toBeInViewport();
  await expect(page.locator('.you')).toBeInViewport();
});

// SPEC gate 6.02: the live search panel still opens fully visible over the
// page once scrolled.
test('6.02: the live search panel opens fully visible after scrolling', async ({ page, server }) => {
  await page.setViewportSize({ width: 1024, height: 700 });
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await seedBoard(page, server.baseURL, uniqueName('Sticky'));
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await scrollToBottom(page);

  await page.fill('#search-box', 'ab');
  const panel = page.locator('.search-panel');
  await expect(panel).toBeVisible();
  await expect(panel).toBeInViewport();
});

// SPEC gate 6.03: the task window (a native <dialog>) and a card's
// people-menu still open above the bar once the page is scrolled.
test('6.03: the task window and a card\'s people menu still open once scrolled', async ({ page, server }) => {
  await page.setViewportSize({ width: 1024, height: 700 });
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await seedBoard(page, server.baseURL, uniqueName('Sticky'));
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await scrollToBottom(page);

  // A card's people menu, opened from wherever the card lands once scrolled.
  const trigger = page.locator('.people-menu-trigger').first();
  await trigger.scrollIntoViewIfNeeded();
  await trigger.click();
  const panel = page.locator('.people-menu-panel').first();
  await expect(panel).toBeVisible();
  await expect(panel).toBeInViewport();
  await page.keyboard.press('Escape');

  // The task window, opened by clicking a card's title.
  await page.locator('.task-card-title a').first().click();
  const dialog = page.locator('.task-window');
  await expect(dialog).toBeVisible();
  await expect(dialog).toBeInViewport();
});

// SPEC gate 6.04: a window too short to show the whole sidebar lets it
// scroll on its own, so Settings can still be reached. 300px is short
// enough, measured against the sidebar's real content, to fully clip
// Settings out of view until the sidebar itself is scrolled.
test('6.04: a short window lets the sidebar scroll to reach Settings', async ({ page, server }) => {
  await page.setViewportSize({ width: 1024, height: 300 });
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const sidebar = page.locator('.sidebar');
  const settings = page.locator('.sidebar nav a[href="/settings"]');
  await expect(settings).not.toBeInViewport();

  await sidebar.evaluate((el) => el.scrollTo(0, el.scrollHeight));
  await expect(settings).toBeInViewport();
  await settings.click();
  await ready(page);
  expect(new URL(page.url()).pathname).toMatch(/^\/settings/);
});

// SPEC gate 6.05: the backup banner (D-80) sits above the top bar and
// scrolls away with the page, rather than staying pinned itself.
test('6.05: the backup banner scrolls away above the pinned top bar', async ({ page, server }) => {
  await page.setViewportSize({ width: 1024, height: 700 });
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await seedBoard(page, server.baseURL, uniqueName('Sticky'));
  const setRes = await page.request.post(server.baseURL + '/__test/backups/set-last-backup-at', {
    form: { days_ago: '3' },
    headers: { origin: server.baseURL },
  });
  expect(setRes.ok()).toBeTruthy();

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);

  const banner = page.locator('.backup-banner');
  await expect(banner).toBeInViewport();
  const topbarBefore = await page.locator('.topbar').boundingBox();
  expect(topbarBefore?.y).toBeGreaterThan(0); // the banner sits above it at the top of the page

  await page.evaluate(() => window.scrollTo(0, 400));
  await expect(banner).not.toBeInViewport();
  const topbarAfter = await page.locator('.topbar').boundingBox();
  expect(topbarAfter?.y).toBe(0);

  // The server (and its backup timestamp) is shared with every other test
  // on this worker (SPEC §2.8): leave it fresh again so a later test
  // doesn't inherit today's stale backup.
  await page.request.post(server.baseURL + '/__test/backups/set-last-backup-at', {
    form: { days_ago: '0' },
    headers: { origin: server.baseURL },
  });
});

test('no sideways scroll with the calendar-length Board', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await seedBoard(page, server.baseURL, uniqueName('Sticky'));
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await expectNoSideScroll(page);
});
