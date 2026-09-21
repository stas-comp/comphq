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

// The job e2e/seed gives exactly 50 steps (seedSteps).
const FIFTY_STEPS_TITLE = 'The fifty steps job';

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

  // SPEC gate 5.17: a job holding a full 50 steps still opens within the
  // budget, on its own page and in the window; and the Board, which now draws
  // a small 3/7 on the jobs that have steps, stays within the same budget the
  // 'every Tasks page' test above holds it to (gate 4.53 unchanged).
  test('a job with 50 steps opens within budget, on its page and in the window', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await page.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(FIFTY_STEPS_TITLE));
    await ready(page);
    const href = await page.locator('.task-card', { hasText: FIFTY_STEPS_TITLE }).locator('.task-card-title a').first().getAttribute('href');
    if (!href) throw new Error('the seeded 50-step job is missing');

    const pageMs = await medianNavigationMs(page, server.baseURL + href);
    await expect(page.locator('.task-steps .step')).toHaveCount(50);
    assertUnderLimit('a job page with 50 steps', pageMs, READY_LIMIT_MS);

    await page.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(FIFTY_STEPS_TITLE));
    await ready(page);
    const times: number[] = [];
    for (let i = 0; i < SAMPLE_SIZE; i++) {
      const start = Date.now();
      await page.locator('.task-card', { hasText: FIFTY_STEPS_TITLE }).locator('.task-card-title a').first().click();
      await expect(page.locator('#task-window .task-steps .step')).toHaveCount(50);
      times.push(Date.now() - start);
      await page.keyboard.press('Escape');
      await expect(page.locator('#task-window')).toBeHidden();
    }
    assertUnderLimit('the window with 50 steps', median(times), READY_LIMIT_MS);
  });

  test('the Board draws its 3/7 badges on the seeded library and passes the standard checks', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    await ready(page);
    expect(await page.locator('.task-card .step-badge').count(), 'seeded jobs with steps show a badge').toBeGreaterThan(0);
    await axeCheck(page);
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

  // SPEC gate 2.23: every remaining Tasks page type, too.
  test('Board, a task page, Finished and Removed pass standard page checks with the seeded library', async ({ page, server }) => {
    // axe walks every node of a 1,500-row Removed list: the check itself is
    // slow on the real library, so this test needs more than the default 30s.
    test.setTimeout(120_000);
    await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    for (const path of ['/tasks/board', taskPath, '/tasks/finished', '/tasks/removed']) {
      await page.goto(server.baseURL + path);
      await ready(page);
      await axeCheck(page);
      await expectNoSideScroll(page);
    }
  });
});
