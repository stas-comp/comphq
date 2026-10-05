import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Article order across an upgrade and a rollback (SPEC gate 7.59, B13.8,
// D-98): a category's articles are put in order under the new build, then the
// container is rolled back to the previous release, which knows nothing about
// the order (and orders its lists by last edit), and an article is published
// there; upgraded again, the category must show the order that was set, with the
// article added while rolled back at the bottom, and no article missing or
// shown twice.
//
// Tagged @order-seed / @order-rolledback / @order-verify-upgraded, none a
// substring of another, so container-test.sh's grep-filtered phases pick
// exactly the right one (as steps.spec.ts and search.spec.ts do). The
// category and titles are fixed, because a later phase has to find them; the
// person is not. Only the plain publish endpoint is used for publishing, so
// the same code runs against the previous release.
const CATEGORY = 'Smoke Test Order Category';
const A = 'Smoke Order Article A';
const B = 'Smoke Order Article B';
const C = 'Smoke Order Article C';
const D = 'Smoke Order Article D (added while rolled back)';

async function categoryID(page: import('@playwright/test').Page, baseURL: string, create: boolean): Promise<string> {
  if (create) {
    const res = await page.request.post(baseURL + '/kb/categories', { form: { name: CATEGORY }, headers: { origin: baseURL } });
    expect(res.ok()).toBe(true);
  }
  await page.goto(baseURL + '/kb/new');
  const value = await page.locator('select[name="category_id"] option', { hasText: CATEGORY }).first().getAttribute('value');
  expect(value).toBeTruthy();
  return value!;
}

async function publish(page: import('@playwright/test').Page, baseURL: string, cat: string, title: string): Promise<string> {
  const res = await page.request.post(baseURL + '/kb/articles', {
    form: { title, category_id: cat, body_html: `<p>${title}</p>` },
    headers: { origin: baseURL },
  });
  expect(res.ok()).toBe(true);
  return /\/kb\/articles\/(\d+)/.exec(res.url())![1];
}

const titles = (page: import('@playwright/test').Page) => page.locator('.kb-article-list li > a, .kb-article-list li a').allInnerTexts();

test('@order-seed three articles put in order: C, A, B', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await categoryID(page, server.baseURL, true);
  await publish(page, server.baseURL, cat, A);
  await publish(page, server.baseURL, cat, B);
  const c = await publish(page, server.baseURL, cat, C);
  for (let i = 0; i < 2; i++) {
    const res = await page.request.post(`${server.baseURL}/kb/articles/${c}/move`, {
      form: { direction: 'up' },
      headers: { origin: server.baseURL },
    });
    expect(res.ok()).toBe(true);
  }
  await page.goto(`${server.baseURL}/kb/categories/${cat}`);
  await ready(page);
  expect(await titles(page)).toEqual([C, A, B]);
});

// Run against the PREVIOUS release's app: it publishes an article into the
// category, as anyone would.
test('@order-rolledback an article published while rolled back, and every article still shown once', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await categoryID(page, server.baseURL, false);
  await publish(page, server.baseURL, cat, D);
  await page.goto(`${server.baseURL}/kb/categories/${cat}`);
  await ready(page);
  const shown = await titles(page);
  for (const t of [A, B, C, D]) expect(shown.filter((x) => x === t), t).toHaveLength(1);
});

test('@order-verify-upgraded the order that was set survived, with the new article at the bottom', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await categoryID(page, server.baseURL, false);
  await page.goto(`${server.baseURL}/kb/categories/${cat}`);
  await ready(page);
  expect(await titles(page)).toEqual([C, A, B, D]);
});
