import { test, expect } from './fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gates 2.21 and 2.23: with the test library (300 events, 100 of
// them repeating), the month and list views are ready within 1.5s, and
// every Calendar page type still passes the accessibility and window-size
// checks. Never raised - a failure means fix the design (PLAN.md P2-17).
const READY_LIMIT_MS = 1500;
const SAMPLE_SIZE = 5;

function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.floor(sorted.length / 2)];
}

async function medianNavigationMs(page: Page, url: string): Promise<number> {
  const times: number[] = [];
  for (let i = 0; i < SAMPLE_SIZE; i++) {
    const start = Date.now();
    await page.goto(url);
    await ready(page);
    times.push(Date.now() - start);
  }
  return median(times);
}

test.describe('Calendar speed (SPEC gates 2.21, 2.23)', () => {
  let eventPath: string;

  test.beforeAll(async ({ browser, server }) => {
    const page = await browser.newPage();
    await signInAsNewPerson(page, server.baseURL, '/calendar/list');
    await ready(page);
    const href = await page.locator('.calendar-list-title a').first().getAttribute('href');
    if (!href) throw new Error('seeded List view is missing an event link');
    eventPath = href.split('?')[0];
    await page.close();
  });

  test('month and list views stay within budget', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar');
    for (const [label, path] of [
      ['month view', '/calendar'],
      ['list view', '/calendar/list'],
    ]) {
      const ms = await medianNavigationMs(page, server.baseURL + path);
      expect(ms, `${label} took ${ms}ms, want <= ${READY_LIMIT_MS}ms`).toBeLessThanOrEqual(READY_LIMIT_MS);
    }
  });

  test('every Calendar page passes standard page checks with the seeded library', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/calendar');
    for (const path of ['/calendar', '/calendar/list', '/calendar/new', '/calendar/removed', eventPath]) {
      await page.goto(server.baseURL + path);
      await ready(page);
      await axeCheck(page);
      await expectNoSideScroll(page);
    }
  });
});
