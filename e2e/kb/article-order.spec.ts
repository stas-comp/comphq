import { expect, test } from '../helpers/fixtures';
import { dragCardTo } from '../helpers/drag';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 7.55-7.57 (B13.8, D-98): articles in their own order.

test.describe.configure({ timeout: 60_000 });

async function newCategory(page: Page, baseURL: string): Promise<{ id: string; name: string }> {
  const name = uniqueName('Order');
  const res = await page.request.post(baseURL + '/kb/categories', { form: { name }, headers: { origin: baseURL } });
  expect(res.ok()).toBe(true);
  await page.goto(baseURL + '/kb/new');
  const value = await page.locator('select[name="category_id"] option', { hasText: name }).getAttribute('value');
  return { id: value!, name };
}

async function publish(page: Page, baseURL: string, categoryID: string, title: string): Promise<string> {
  const res = await page.request.post(baseURL + '/kb/articles', {
    form: { title, category_id: categoryID, body_html: `<p>${title}</p>` },
    headers: { origin: baseURL },
  });
  expect(res.ok()).toBe(true);
  return /\/kb\/articles\/(\d+)/.exec(res.url())![1];
}

const titlesOf = (page: Page) => page.locator('.kb-article-list li:not(.sortable-fallback) > a').allInnerTexts();

async function setup(page: Page, baseURL: string, n = 3) {
  await signInAsNewPerson(page, baseURL, '/kb');
  const cat = await newCategory(page, baseURL);
  const titles = Array.from({ length: n }, (_, i) => `${uniqueName('Art')} ${String.fromCharCode(65 + i)}`);
  const ids: string[] = [];
  for (const t of titles) ids.push(await publish(page, baseURL, cat.id, t));
  await page.goto(`${baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  return { cat, titles, ids };
}

test('gate 7.55: a category lists its articles in their own order, with move icons that put them in order', async ({ page, server, browser }) => {
  const { cat, titles } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  expect(await titlesOf(page)).toEqual([a, b, c]); // creation order: new ones at the bottom

  const row = (t: string) => page.locator('.kb-article-list li', { hasText: t });
  await expect(row(a).getByRole('button', { name: `Move ${a} up` })).toBeDisabled();
  await expect(row(c).getByRole('button', { name: `Move ${c} down` })).toBeDisabled();
  await row(c).getByRole('button', { name: `Move ${c} up` }).click();
  await ready(page);
  expect(await titlesOf(page)).toEqual([a, c, b]);
  await row(a).getByRole('button', { name: `Move ${a} down` }).click();
  await ready(page);
  expect(await titlesOf(page)).toEqual([c, a, b]);

  // The same on every computer.
  const other = await browser.newContext();
  await other.addCookies(await page.context().cookies(server.baseURL));
  const op = await other.newPage();
  await op.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(op);
  expect(await titlesOf(op)).toEqual([c, a, b]);
  await other.close();
});

test('gate 7.55: articles can be dragged up and down', async ({ page, server }) => {
  const { cat, titles } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  const row = (t: string) => page.locator('.kb-article-list li:not(.sortable-fallback)', { hasText: t });
  const status = await dragCardTo(
    page,
    row(c),
    async () => {
      const box = (await row(a).boundingBox())!;
      return { x: box.x + box.width / 2, y: box.y + box.height * 0.2 };
    },
    {
      endpoint: /\/kb\/articles\/\d+\/move$/,
      settled: async () => (await titlesOf(page)).join() === [c, a, b].join(),
    },
  );
  expect(status).toBe(302);
  await expect.poll(() => titlesOf(page)).toEqual([c, a, b]);
  await page.reload();
  await ready(page);
  expect(await titlesOf(page)).toEqual([c, a, b]);
  expect(cat.id).toBeTruthy();
});

test('gate 7.56: editing does not move an article, and reordering is not an edit', async ({ page, server }) => {
  const { cat, titles, ids } = await setup(page, server.baseURL);
  const [a, b, c] = titles;

  // Recently updated on the home page, before.
  await page.goto(server.baseURL + '/kb');
  await ready(page);
  const recentBefore = await page.locator('.kb-recent-list li').allInnerTexts();

  // Reorder: C to the top.
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  await page.locator('.kb-article-list li', { hasText: c }).getByRole('button', { name: `Move ${c} up` }).click();
  await ready(page);
  await page.locator('.kb-article-list li', { hasText: c }).getByRole('button', { name: `Move ${c} up` }).click();
  await ready(page);
  expect(await titlesOf(page)).toEqual([c, a, b]);

  // No History version was added by reordering: C still has version 1 only.
  await page.goto(`${server.baseURL}/kb/articles/${ids[2]}/history`);
  await ready(page);
  await expect(page.locator('.kb-history-list li')).toHaveCount(1);

  // Recently updated is unchanged by reordering.
  await page.goto(server.baseURL + '/kb');
  await ready(page);
  expect(await page.locator('.kb-recent-list li').allInnerTexts()).toEqual(recentBefore);

  // Editing B (the last one) does not move it to the top.
  await page.goto(`${server.baseURL}/kb/articles/${ids[1]}/edit`);
  await ready(page);
  await page.fill('#article-title', b + ' (edited)');
  await page.click('#btn-publish');
  await ready(page);
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  expect(await titlesOf(page)).toEqual([c, a, b + ' (edited)']);
});

test('gate 7.57: a new article, one moved to another category, and one brought back from Archived go to the bottom', async ({ page, server }) => {
  const { cat, titles, ids } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  const second = await newCategory(page, server.baseURL);
  const x = uniqueName('Other X');
  await publish(page, server.baseURL, second.id, x);

  // New article: the bottom.
  const d = uniqueName('Art D');
  await publish(page, server.baseURL, cat.id, d);
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  expect(await titlesOf(page)).toEqual([a, b, c, d]);

  // A moved to the other category: the bottom there, and gone from here.
  await page.goto(`${server.baseURL}/kb/articles/${ids[0]}/edit`);
  await ready(page);
  await page.selectOption('#article-category', second.id);
  await page.click('#btn-publish');
  await ready(page);
  await page.goto(`${server.baseURL}/kb/categories/${second.id}`);
  await ready(page);
  expect(await titlesOf(page)).toEqual([x, a]);
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  expect(await titlesOf(page)).toEqual([b, c, d]);

  // B archived, then brought back: the bottom.
  const archive = await page.request.post(`${server.baseURL}/kb/articles/${ids[1]}/archive`, { headers: { origin: server.baseURL } });
  expect(archive.ok()).toBe(true);
  const back = await page.request.post(`${server.baseURL}/kb/articles/${ids[1]}/unarchive`, { headers: { origin: server.baseURL } });
  expect(back.ok()).toBe(true);
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  expect(await titlesOf(page)).toEqual([c, d, b]);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 7.55: the move icons are ordinary forms', async ({ page, server }) => {
    const { titles } = await setup(page, server.baseURL);
    const [a, b, c] = titles;
    await page.locator('.kb-article-list li', { hasText: c }).getByRole('button', { name: `Move ${c} up` }).click();
    expect(await titlesOf(page)).toEqual([a, c, b]);
  });
});
