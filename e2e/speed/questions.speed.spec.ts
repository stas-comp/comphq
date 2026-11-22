import { test, expect } from './fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gate 8.83 (B14.6, D-110): with 1,000 questions and 3,000 answers in the test
// library, the Questions page and a question's page are ready within 1.5 seconds,
// and search answers within 1 second. Never raised: a failure means fix the design.
const READY_LIMIT_MS = 1500;
const SEARCH_LIMIT_MS = 1000;
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

test.describe('Questions speed (SPEC gate 8.83)', () => {
  test('the Questions page and a question page are ready within budget, with the seeded library', async ({ page, server }) => {
    test.setTimeout(120_000);
    await signInAsNewPerson(page, server.baseURL, '/questions');
    await ready(page);
    // The library really is there: about 400 waiting, and the 50 newest answered.
    expect(await page.locator('section[aria-labelledby="q-waiting-heading"] .q-row').count()).toBeGreaterThan(300);
    expect(await page.locator('section[aria-labelledby="q-answered-heading"] .q-row').count()).toBe(50);

    assertUnderLimit('Questions page', await medianNavigationMs(page, server.baseURL + '/questions'), READY_LIMIT_MS);
    // Show all prints 600 answered questions as well, the heaviest the page gets.
    assertUnderLimit('Questions page, Show all', await medianNavigationMs(page, server.baseURL + '/questions?all=1'), READY_LIMIT_MS);

    const href = await page.locator('.q-row a').first().getAttribute('href');
    expect(href).toMatch(/^\/questions\/\d+$/);
    assertUnderLimit('a question page', await medianNavigationMs(page, server.baseURL + href!), READY_LIMIT_MS);
    await expect(page.locator('.q-answer')).toHaveCount(3);
  });

  test('last keystroke to the question results rendering stays within budget', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/questions');
    await ready(page);
    // Real words from the seeded questions.
    const titles = await page.locator('.q-row-title a').allInnerTexts();
    const words = titles
      .flatMap((t) => t.split(/[^\p{L}\p{N}]+/u))
      .filter((w) => w.length >= 4 && !/^\d+$/.test(w) && w !== 'Question')
      .slice(0, 5);
    expect(words.length).toBeGreaterThan(0);

    const times: number[] = [];
    for (let i = 0; i < SAMPLE_SIZE; i++) {
      const start = Date.now();
      await page.fill('#search-box', words[i % words.length]);
      await page.waitForSelector('.search-panel .search-result');
      times.push(Date.now() - start);
      await page.fill('#search-box', '');
      await page.waitForSelector('.search-panel', { state: 'detached' });
    }
    assertUnderLimit('question search panel render', median(times), SEARCH_LIMIT_MS);

    // The word every question has: the most results the search ever gives.
    const start = Date.now();
    const res = await page.request.get(server.baseURL + '/questions/search.json?q=question');
    expect(res.status()).toBe(200);
    assertUnderLimit('question search of a word every question has', Date.now() - start, SEARCH_LIMIT_MS);
  });

  test('symbols and odd input never cause an error', async ({ page, server }) => {
    const problems: string[] = [];
    page.on('pageerror', (e) => problems.push(String(e)));
    page.on('console', (m) => {
      if (m.type() === 'error') problems.push(m.text());
    });
    await signInAsNewPerson(page, server.baseURL, '/questions');
    await ready(page);
    for (const q of ['%', '_', '"', "'", '\\', 'NEAR AND OR', '100%', ')(', '<script>', 'a b', '😀😀', '%%__']) {
      const res = await page.request.get(server.baseURL + '/questions/search.json?q=' + encodeURIComponent(q));
      expect(res.status(), q).toBe(200);
      await page.fill('#search-box', q);
      await page.waitForTimeout(400);
    }
    expect(problems).toEqual([]);
  });

  test('the Questions pages pass standard page checks with the seeded library', async ({ page, server }) => {
    test.setTimeout(120_000);
    await signInAsNewPerson(page, server.baseURL, '/questions');
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
  });

  test('the Briefing stays within budget with 400 questions waiting, and lists five', async ({ page, server }) => {
    test.setTimeout(120_000);
    await signInAsNewPerson(page, server.baseURL, '/briefing');
    await ready(page);
    await expect(page.locator('.briefing-question')).toHaveCount(5);
    await expect(page.getByRole('link', { name: /^and \d+ more$/ })).toBeVisible();
    assertUnderLimit('Briefing with questions', await medianNavigationMs(page, server.baseURL + '/briefing'), READY_LIMIT_MS);
  });
});
