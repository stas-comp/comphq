import { test, expect } from './fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gate 3.13: with the test library (2,000 tasks, 300 events, 100 of
// them repeating), the Briefing is ready within 1.5s and still passes the
// accessibility and window-size checks. The library's task due dates are
// relative to the day it is seeded, so this run's Briefing genuinely
// lists a large share of the active tasks — overdue ones included — not
// an empty page. Never raised — a failure means fix the design (PLAN.md
// P3-03).
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

test.describe('Briefing speed (SPEC gate 3.13)', () => {
  test('the Briefing is ready within budget, and is not an empty page', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/briefing');
    await ready(page);

    const cards = await page.locator('.briefing-card').count();
    expect(cards, 'the seeded library should put real cards on the Briefing').toBeGreaterThan(20);

    for (const [label, path] of [
      ['Briefing at "/"', '/'],
      ['Briefing', '/briefing'],
      ['Briefing, just mine', '/briefing?mine=1'],
    ]) {
      const ms = await medianNavigationMs(page, server.baseURL + path);
      expect(ms, `${label} took ${ms}ms, want <= ${READY_LIMIT_MS}ms`).toBeLessThanOrEqual(READY_LIMIT_MS);
    }
  });

  test('the Briefing passes standard page checks with the seeded library', async ({ page, server }) => {
    test.setTimeout(120_000);
    await signInAsNewPerson(page, server.baseURL, '/briefing');
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
  });
});
