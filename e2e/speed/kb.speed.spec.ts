import { test, expect } from './fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC B7 layer 5 / gate 1.33 limits. Never raised — a failure here means
// fix the design (indexes, query shape, template work), per PLAN.md P1-27.
const READY_LIMIT_MS = 1500;
const PANEL_LIMIT_MS = 1000;
const SEARCH_JSON_P95_LIMIT_MS = 200;
const SAMPLE_SIZE = 5;

function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.floor(sorted.length / 2)];
}

function p95(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.min(sorted.length - 1, Math.ceil(sorted.length * 0.95) - 1)];
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

// A handful of words the seeder's own vocabulary actually contains
// (e2e/seed/main.go), so these queries exercise real FTS5 matches against
// the seeded library rather than always hitting the empty-results path.
const REAL_QUERY_WORDS = ['toner', 'printer', 'network', 'password', 'invoice', 'safety', 'backup', 'wifi', 'badge', 'server'];

test.describe('Knowledge Base speed (SPEC gate 1.33)', () => {
  let categoryPath: string;
  let articlePath: string;

  test.beforeAll(async ({ browser, server }) => {
    const page = await browser.newPage();
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await ready(page);

    const categoryHref = await page.locator('.kb-tile').first().getAttribute('href');
    const articleHref = await page.locator('.kb-recent-list a[href^="/kb/articles/"]').first().getAttribute('href');
    if (!categoryHref || !articleHref) {
      throw new Error('seeded KB home is missing the expected category/article links');
    }
    categoryPath = categoryHref;
    articlePath = articleHref;

    await page.close();
  });

  test('navigation to data-ready stays within budget for every page type', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');

    const pages: [string, string][] = [
      ['KB home', '/kb'],
      ['a category', categoryPath],
      ['an article', articlePath],
      ['the editor', '/kb/new'],
      ['History', articlePath + '/history'],
      ['Archived', '/kb/archived'],
      ['search results', '/kb/search?q=toner'],
    ];

    for (const [label, path] of pages) {
      const ms = await medianNavigationMs(page, server.baseURL + path);
      assertUnderLimit(label, ms, READY_LIMIT_MS);
    }
  });

  test('last keystroke to the search panel rendering stays within budget', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await ready(page);

    const times: number[] = [];
    for (let i = 0; i < SAMPLE_SIZE; i++) {
      const start = Date.now();
      await page.fill('#search-box', REAL_QUERY_WORDS[i % REAL_QUERY_WORDS.length]);
      await page.waitForSelector('.search-panel');
      times.push(Date.now() - start);

      await page.fill('#search-box', '');
      await page.waitForSelector('.search-panel', { state: 'detached' });
    }
    assertUnderLimit('search panel render', median(times), PANEL_LIMIT_MS);
  });

  test('search.json p95 latency over 50 queries stays within budget', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');

    const times: number[] = [];
    for (let i = 0; i < 50; i++) {
      const q = REAL_QUERY_WORDS[i % REAL_QUERY_WORDS.length];
      const start = Date.now();
      const res = await page.request.get(server.baseURL + '/kb/search.json?q=' + encodeURIComponent(q));
      expect(res.ok()).toBe(true);
      times.push(Date.now() - start);
    }
    assertUnderLimit('search.json p95', p95(times), SEARCH_JSON_P95_LIMIT_MS);
  });

  // SPEC gate 6.35: a two-letter last word is the widest possible
  // half-typed-word expansion (B12.5's own vocabulary lookup, capped at
  // 30 matches, most common first) — still within the same budget as any
  // other query against the test library.
  test('search.json p95 latency with a two-letter last word stays within budget', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');

    const times: number[] = [];
    for (let i = 0; i < 50; i++) {
      const start = Date.now();
      const res = await page.request.get(server.baseURL + '/kb/search.json?q=' + encodeURIComponent('pa'));
      expect(res.ok()).toBe(true);
      times.push(Date.now() - start);
    }
    assertUnderLimit('search.json p95 (two-letter word)', p95(times), SEARCH_JSON_P95_LIMIT_MS);
  });
});
