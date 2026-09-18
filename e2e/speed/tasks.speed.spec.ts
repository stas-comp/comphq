import { test, expect } from './fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gates 2.21, 2.31, 2.38: the test library (2,000 tasks, 10
// people) never pushes any Tasks page over this. Never raised — a
// failure here means fix the design (indexes, query shape, template
// work), per PLAN.md P1-27/P2-12.
const READY_LIMIT_MS = 1500;
const SAMPLE_SIZE = 5;

function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.floor(sorted.length / 2)];
}

function assertUnderLimit(label: string, ms: number, limit: number) {
  expect(ms, `${label} took ${ms}ms, want <= ${limit}ms`).toBeLessThanOrEqual(limit);
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

test.describe('Tasks speed (SPEC gates 2.21, 2.31, 2.38)', () => {
  let taskPath: string;

  test.beforeAll(async ({ browser, server }) => {
    const page = await browser.newPage();
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await ready(page);

    const taskHref = await page.locator('.task-card-title a').first().getAttribute('href');
    if (!taskHref) throw new Error('seeded Board is missing the expected task link');
    taskPath = taskHref;

    await page.close();
  });

  test('navigation to data-ready stays within budget for every Tasks page', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');

    const pages: [string, string][] = [
      ['Board', '/tasks/board'],
      ['Team', '/tasks/team'],
      ['My jobs', '/tasks'],
      ['a task page', taskPath],
      ['Finished tasks', '/tasks/finished'],
      ['Removed tasks', '/tasks/removed'],
    ];

    for (const [label, path] of pages) {
      const ms = await medianNavigationMs(page, server.baseURL + path);
      assertUnderLimit(label, ms, READY_LIMIT_MS);
    }
  });

  // SPEC gate 2.31: Team, with the real 2,000-task/10-person library,
  // still passes accessibility and the sideways-scroll check.
  test('Team view passes standard page checks with the seeded library', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/team');
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
  });

  // SPEC gate 2.38: My jobs, with the real library, still passes
  // accessibility and the sideways-scroll check.
  test('My jobs passes standard page checks with the seeded library', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks');
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
  });
});
