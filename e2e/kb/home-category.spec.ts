import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 7.70-7.72 and 7.77 (B13.10, D-100): the Knowledge Base home and a
// category's own page.

test.describe.configure({ timeout: 60_000 });

async function newCategory(page: Page, baseURL: string, label = 'Tidy'): Promise<{ id: string; name: string }> {
  const name = uniqueName(label);
  const res = await page.request.post(baseURL + '/kb/categories', { form: { name }, headers: { origin: baseURL } });
  expect(res.ok()).toBe(true);
  await page.goto(baseURL + '/kb/new');
  const id = await page.locator('select[name="category_id"] option', { hasText: name }).getAttribute('value');
  return { id: id!, name };
}

async function publish(page: Page, baseURL: string, categoryID: string, title: string, body = `<p>${title}</p>`): Promise<string> {
  const res = await page.request.post(baseURL + '/kb/articles', {
    form: { title, category_id: categoryID, body_html: body },
    headers: { origin: baseURL },
  });
  expect(res.ok()).toBe(true);
  return /\/kb\/articles\/(\d+)/.exec(res.url())![1];
}

const tile = (page: Page, name: string) => page.locator('.kb-tile', { hasText: name });

test('gate 7.70: the home has a heading row, and a tile per category with its count, first three articles in order, and the All link', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/kb');
  expect(me).toBeTruthy();
  const cat = await newCategory(page, server.baseURL, 'Tiles');
  const titles = ['A', 'B', 'C', 'D'].map((l) => `${uniqueName('Tile')} ${l}`);
  const ids: string[] = [];
  for (const t of titles) ids.push(await publish(page, server.baseURL, cat.id, t));
  // Put D first: the tile follows the category's own order, not who was edited last.
  await page.request.post(`${server.baseURL}/kb/articles/${ids[3]}/move`, {
    form: { before_id: ids[0] },
    headers: { origin: server.baseURL },
    maxRedirects: 0,
  });

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await expect(page.getByRole('heading', { level: 1, name: 'Knowledge Base' })).toBeVisible();
  const row = page.locator('.kb-head-actions');
  await expect(row.getByRole('link', { name: 'New article' })).toHaveClass(/\bprimary\b/);
  await expect(row.getByRole('link', { name: 'Categories' })).toHaveAttribute('href', '/kb/categories');
  await expect(row.getByRole('link', { name: 'Archived' })).toHaveAttribute('href', '/kb/archived');

  const t = tile(page, cat.name);
  await expect(t.locator('h2')).toHaveText(cat.name);
  await expect(t.locator('.count')).toHaveText('4 articles');
  expect(await t.locator('.kb-tile-articles a').allInnerTexts()).toEqual([titles[3], titles[0], titles[1]]);
  const all = t.getByRole('link', { name: 'All 4 articles →' });
  await expect(all).toHaveAttribute('href', `/kb/categories/${cat.id}`);
  await all.click();
  await expect(page).toHaveURL(new RegExp(`/kb/categories/${cat.id}$`));
});

test('gate 7.70: an empty category says so and links to adding one, already filed in it', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL, 'Empty');
  await page.goto(server.baseURL + '/kb');
  await ready(page);
  const t = tile(page, cat.name);
  await expect(t).toContainText('0 articles');
  await expect(t).toContainText('No articles yet');
  await expect(t.locator('.kb-tile-all')).toHaveCount(0);
  await t.getByRole('link', { name: 'Add one' }).click();
  await expect(page.locator('#article-category')).toHaveValue(cat.id);
});

test('gate 7.71: Recently updated is a panel of the 10 newest, each row with title, category, and who · when', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL, 'Recent');
  const titles: string[] = [];
  for (let i = 0; i < 12; i++) {
    const t = `${uniqueName('Rec')} ${i}`;
    titles.push(t);
    await publish(page, server.baseURL, cat.id, t);
  }
  await page.goto(server.baseURL + '/kb');
  await ready(page);
  const panel = page.locator('section.kb-recent');
  await expect(panel.locator('.panel-head h2')).toHaveText('Recently updated');
  const rows = panel.locator('.kb-recent-list li');
  await expect(rows).toHaveCount(10);
  // The server is shared with other tests, so the 10 newest are not all mine; each row is checked for its parts,
  // and the order for being newest first (gate 1.13, unchanged).
  const stamps: number[] = [];
  for (const row of await rows.all()) {
    await expect(row.locator('a.kb-recent-title')).not.toBeEmpty();
    await expect(row.locator('.kb-recent-category')).not.toBeEmpty();
    const when = await row.locator('.kb-recent-by .date').innerText();
    expect(when).toMatch(/^\w{3} \d{1,2} \w{3} \d{4}, \d{2}:\d{2}$/); // the app's date style
    stamps.push(Date.parse(when.replace(/^\w{3} /, '').replace(',', '')));
  }
  expect([...stamps].sort((x, y) => y - x)).toEqual(stamps);
  const mine = rows.filter({ hasText: me }).first();
  await expect(mine.locator('.kb-recent-by')).toContainText(`${me} ·`);
  await expect(mine.locator('.kb-recent-category')).toHaveText(cat.name);
  await expect(panel).not.toContainText(titles[0]); // older than the 10 newest
});

test('gate 7.72: a category page has the trail, the name with its count, and a New article button that files it there', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL, 'Office');
  const long = 'alpha beta gamma delta '.repeat(20); // 460 characters
  const a = `${uniqueName('Page')} first`;
  const b = `${uniqueName('Page')} second`;
  const c = `${uniqueName('Page')} third`;
  await publish(page, server.baseURL, cat.id, a, `<h2>A heading comes first</h2><p>Open the <b>front</b> door.</p><p>Second paragraph.</p>`);
  await publish(page, server.baseURL, cat.id, b, `<p>${long}</p>`);
  await publish(page, server.baseURL, cat.id, c, `<h2>Only a heading</h2>`);

  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  const crumbs = page.getByRole('navigation', { name: 'Breadcrumb' });
  await expect(crumbs).toContainText('Knowledge Base › ' + cat.name);
  await expect(crumbs.getByRole('link', { name: 'Knowledge Base' })).toHaveAttribute('href', '/kb');
  await expect(page.getByRole('heading', { level: 1 })).toContainText(cat.name);
  await expect(page.locator('h1 .count')).toHaveText('3 articles');

  const row = (t: string) => page.locator('.kb-article-list li', { hasText: t });
  // The opening words are the first block that is not a heading.
  await expect(row(a).locator('.kb-article-opening')).toHaveText('Open the front door.');
  // Cut at a word, about 140 characters, with an ellipsis.
  const cut = (await row(b).locator('.kb-article-opening').innerText()).trim();
  expect(cut.endsWith('…')).toBe(true);
  expect(cut.length).toBeLessThanOrEqual(141);
  expect(long.startsWith(cut.slice(0, -1))).toBe(true);
  expect(long[cut.length - 1]).toBe(' '); // the next character was a space: the cut is on a word boundary
  // An article with nothing but a heading has no opening words line.
  await expect(row(c).locator('.kb-article-opening')).toHaveCount(0);
  // Who and when, in the app's date style (without the time).
  await expect(row(a).locator('.kb-article-by')).toHaveText(new RegExp(`^Updated by ${me} · \\w{3} \\d{1,2} \\w{3} \\d{4}$`));
  // Each row still has its move icons.
  await expect(row(b).getByRole('button', { name: `Move ${b} up` })).toBeVisible();
  await expect(row(b).getByRole('button', { name: `Move ${b} down` })).toBeVisible();

  const add = page.getByRole('link', { name: 'New article' });
  await expect(add).toHaveClass(/\bprimary\b/);
  await add.click();
  await expect(page).toHaveURL(new RegExp(`/kb/new\\?category=${cat.id}$`));
  await expect(page.locator('#article-category')).toHaveValue(cat.id);
});

test('gate 7.72: an empty category says so in a friendly way, with the New article button', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL, 'Quiet');
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  await expect(page.locator('h1 .count')).toHaveText('0 articles');
  await expect(page.locator('.kb-empty')).toContainText('No articles in this category yet.');
  await expect(page.getByRole('link', { name: 'New article' })).toHaveAttribute('href', `/kb/new?category=${cat.id}`);
  await expectNoSideScroll(page);
  await axeCheck(page);
});

test('gate 7.72: a category that does not exist is ignored on New article, and the choice stays open', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await newCategory(page, server.baseURL, 'Any');
  await page.goto(server.baseURL + '/kb/new?category=99999999');
  await ready(page);
  await expect(page.locator('#article-category')).toBeVisible();
  await page.goto(server.baseURL + '/kb/new?category=oops');
  await expect(page.locator('#article-category')).toBeVisible();
});

test('gate 7.77: both screens pass the accessibility check and never scroll sideways, with long names and a full list', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL, 'A really quite long category name that goes on and on and on and on');
  for (let i = 0; i < 6; i++) {
    await publish(page, server.baseURL, cat.id, `${uniqueName('Long')} ${'wordy '.repeat(18)}${i}`, `<p>${'text '.repeat(80)}</p>`);
  }
  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await expectNoSideScroll(page);
  await axeCheck(page);
  await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
  await ready(page);
  await expectNoSideScroll(page);
  await axeCheck(page);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 7.72: the New article button and the move icons work with no script', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const cat = await newCategory(page, server.baseURL, 'Plain');
    const a = uniqueName('Plain') + ' A';
    const b = uniqueName('Plain') + ' B';
    await publish(page, server.baseURL, cat.id, a);
    await publish(page, server.baseURL, cat.id, b);
    await page.goto(`${server.baseURL}/kb/categories/${cat.id}`);
    await page.locator('.kb-article-list li', { hasText: b }).getByRole('button', { name: `Move ${b} up` }).click();
    await expect(page.locator('.kb-article-list li > a').first()).toHaveText(b);
    await page.getByRole('link', { name: 'New article' }).click();
    await expect(page.locator('#article-category')).toHaveValue(cat.id);
  });
});
