import path from 'node:path';
import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 7.73-7.76 (B13.10, D-100): the editor arranged as the mockup's
// editor screen is. What the editor does is unchanged (gates 1.14-1.25 and
// 1.45-1.52 are re-run as they were); this is about the arrangement.

test.describe.configure({ timeout: 60_000 });

async function newCategory(page: Page, baseURL: string): Promise<{ id: string; name: string }> {
  const name = uniqueName('Look');
  const res = await page.request.post(baseURL + '/kb/categories', { form: { name }, headers: { origin: baseURL } });
  expect(res.ok()).toBe(true);
  await page.goto(baseURL + '/kb/new');
  const id = (await page.locator('select[name="category_id"] option', { hasText: name }).getAttribute('value'))!;
  return { id, name };
}

async function openEditor(page: Page, baseURL: string, url = '/kb/new') {
  await page.goto(baseURL + url);
  await page.waitForSelector('body[data-editor-ready]');
}

const topbarHeight = (page: Page) =>
  page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--topbar-height')) || 0);

test('gate 7.73: the head holds the trail with the category chosen in it, Cancel and Publish', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL);
  await openEditor(page, server.baseURL, `/kb/new?category=${cat.id}`);

  const head = page.locator('.edit-head');
  await expect(head.getByRole('link', { name: 'Knowledge Base' })).toHaveAttribute('href', '/kb');
  await expect(head.getByLabel('Category')).toHaveValue(cat.id);
  await expect(head).toContainText('New article');
  await expect(head.getByRole('button', { name: 'Cancel' })).toBeVisible();
  await expect(head.getByRole('button', { name: 'Publish' })).toHaveClass(/\bprimary\b/);
  // Cancel and Publish sit at the right-hand end, after the trail.
  const crumbs = (await head.locator('.kb-crumbs').boundingBox())!;
  const cancel = (await head.getByRole('button', { name: 'Cancel' }).boundingBox())!;
  const publish = (await head.getByRole('button', { name: 'Publish' }).boundingBox())!;
  expect(cancel.x).toBeGreaterThan(crumbs.x + 100);
  expect(publish.x).toBeGreaterThan(cancel.x);
  // The only thing down the page is the toolbar and the article: no second Publish.
  await expect(page.getByRole('button', { name: 'Publish' })).toHaveCount(1);
});

test('gate 7.73: the title is a large box with a spoken label and no visible label', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await newCategory(page, server.baseURL);
  await openEditor(page, server.baseURL);
  const title = page.getByLabel('Title');
  await expect(title).toHaveAttribute('id', 'article-title');
  const label = (await page.locator('label[for="article-title"]').boundingBox())!;
  expect(label.width).toBeLessThanOrEqual(2); // visually hidden, still read aloud
  expect(parseFloat(await title.evaluate((el) => getComputedStyle(el).fontSize))).toBeGreaterThanOrEqual(40);
});

test('gate 7.73: the toolbar is the mockup’s compact one, in its order, with Import from Word at the end', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await newCategory(page, server.baseURL);
  await openEditor(page, server.baseURL);
  const names = await page.locator('.editor-toolbar > button:visible').evaluateAll((els) =>
    els.map((el) => el.getAttribute('aria-label') || (el.textContent || '').trim()),
  );
  expect(names).toEqual(['Heading', 'Subheading', 'Bold', 'Italic', '• List', '1. List', 'Link', 'Table', 'Picture', 'Import from Word']);
  await expect(page.locator('#btn-bold')).toHaveText('B');
  await expect(page.locator('#btn-bold')).toHaveAttribute('title', 'Bold');
  await expect(page.locator('#btn-italic')).toHaveText('I');
  await expect(page.locator('#btn-italic')).toHaveAttribute('title', 'Italic');
});

test('gate 7.73: the row and column buttons appear only while the cursor is in a table', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await newCategory(page, server.baseURL);
  await openEditor(page, server.baseURL);
  const tools = page.locator('#table-tools');
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await page.keyboard.type('Some words');
  await expect(tools).toBeHidden();
  await page.locator('#btn-table').click();
  await expect(tools).toBeVisible();
  for (const name of ['Add row', 'Remove row', 'Add column', 'Remove column']) {
    await expect(tools.getByRole('button', { name })).toBeVisible();
  }
  // Click back into the words above the table: the controls go.
  await editor.locator('p').first().click();
  await expect(tools).toBeHidden();
  // And click into a cell: they return.
  await editor.locator('td, th').first().click();
  await expect(tools).toBeVisible();
});

test('gate 7.73: the head and toolbar stay in view while a long article scrolls (1024 × 700)', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL);
  const body = Array.from({ length: 80 }, (_, i) => `<p>Paragraph ${i} of a long article, with some words in it.</p>`).join('');
  const res = await page.request.post(server.baseURL + '/kb/articles', {
    form: { title: uniqueName('Long'), category_id: cat.id, body_html: body },
    headers: { origin: server.baseURL },
  });
  const id = /\/kb\/articles\/(\d+)/.exec(res.url())![1];
  await page.setViewportSize({ width: 1024, height: 700 });
  await openEditor(page, server.baseURL, `/kb/articles/${id}/edit`);
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(1000);

  const top = await topbarHeight(page);
  const head = (await page.locator('.edit-head').boundingBox())!;
  const bar = (await page.locator('.editor-toolbar').boundingBox())!;
  expect(head.y).toBeGreaterThanOrEqual(top - 1);
  expect(head.y).toBeLessThan(top + 4); // right under the top bar
  expect(bar.y).toBeGreaterThanOrEqual(head.y + head.height - 1);
  expect(bar.y).toBeLessThan(head.y + head.height + 4); // right under the head
  await expect(page.getByRole('button', { name: 'Publish' })).toBeInViewport();
  await expect(page.locator('#btn-import-word')).toBeInViewport();
  // The article scrolls behind them without showing through: they are opaque.
  await expect(page.locator('.edit-head')).toHaveCSS('background-color', 'rgb(245, 243, 238)');
  await expectNoSideScroll(page);
  await axeCheck(page);
});

test('gate 7.74: every kind of box for what could not come across is a bold sentence with a tip under it', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL);
  const kinds = ['picture', 'external', 'chart', 'diagram', 'shape', 'drawing', 'equation', 'object'];
  const res = await page.request.post(server.baseURL + '/kb/articles', {
    form: {
      title: uniqueName('Boxes'),
      category_id: cat.id,
      body_html: kinds.map((k) => `<div data-missing-kind="${k}"></div>`).join('') + '<p>after</p>',
    },
    headers: { origin: server.baseURL },
  });
  const id = /\/kb\/articles\/(\d+)/.exec(res.url())![1];
  for (const where of [`/kb/articles/${id}/edit`, `/kb/articles/${id}`]) {
    if (where.endsWith('/edit')) await openEditor(page, server.baseURL, where);
    else {
      await page.goto(server.baseURL + where);
      await ready(page);
    }
    for (const kind of kinds) {
      const box = page.locator(`div[data-missing-kind="${kind}"]`);
      await expect(box, `${kind} on ${where}`).toHaveCount(1);
      const parts = await box.evaluate((el) => {
        const read = (pseudo: string) => {
          const cs = getComputedStyle(el, pseudo);
          return { content: cs.content, weight: cs.fontWeight };
        };
        return { sentence: read('::before'), tip: read('::after') };
      });
      expect(parts.sentence.content, `${kind} sentence`).toMatch(/couldn't be (brought in|copied)\.\"$/);
      expect(Number(parts.sentence.weight), `${kind} sentence is bold`).toBeGreaterThanOrEqual(700);
      expect(parts.tip.content, `${kind} tip`).toMatch(/^"Tip: /);
      expect(Number(parts.tip.weight), `${kind} tip is not bold`).toBeLessThan(700);
      if (kind === 'drawing') expect(parts.tip.content, 'the arrows in the EMF tip are real arrows').toContain('→');
    }
    await axeCheck(page);
  }
});

test('gate 7.75: the import message begins "Came across: …" with what the document held', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await newCategory(page, server.baseURL);
  await openEditor(page, server.baseURL);
  const chooser = page.waitForEvent('filechooser');
  await page.click('#btn-import-word');
  await (await chooser).setFiles(path.join(__dirname, '..', 'fixtures', 'docx', 'pictures-25.docx'));
  const notice = page.locator('#editor-message.imported');
  await expect(notice).toBeVisible();
  await expect(notice.locator('li').first()).toHaveText(/^Came across: .*25 pictures\.$/);
  await expect(notice).toContainText('is in the editor');
  await axeCheck(page);
});

test('gate 7.75: the count has singular and plural right, in this order: headings, lists, tables, pictures', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await newCategory(page, server.baseURL);
  await openEditor(page, server.baseURL);
  const sentences = await page.evaluate(() => {
    const m = (window as unknown as { ComphqMessages: { cameAcross(c: object): string } }).ComphqMessages;
    return [
      m.cameAcross({ headings: 2, lists: 1, tables: 1, pictures: 3 }),
      m.cameAcross({ headings: 1, lists: 0, tables: 2, pictures: 1 }),
      m.cameAcross({ headings: 0, lists: 0, tables: 0, pictures: 0 }),
    ];
  });
  expect(sentences).toEqual(['Came across: 2 headings, 1 list, 1 table, 3 pictures.', 'Came across: 1 heading, 2 tables, 1 picture.', 'Came across: the text.']);
});

test('gate 7.76: Cancel on a dirty editor still asks, and Publish still files the article where the head says', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const cat = await newCategory(page, server.baseURL);
  await openEditor(page, server.baseURL);
  const title = uniqueName('Head publish');
  await page.getByLabel('Title').fill(title);
  await page.locator('#article-editor .ProseMirror').click();
  await page.keyboard.type('Body text');
  let asked = '';
  page.once('dialog', (d) => {
    asked = d.message();
    void d.dismiss();
  });
  await page.getByRole('button', { name: 'Cancel' }).click();
  expect(asked).toBe('Leave without saving?');
  await page.getByLabel('Category').selectOption(cat.id);
  await page.getByRole('button', { name: 'Publish' }).click();
  await ready(page);
  await expect(page.getByRole('heading', { level: 1 })).toHaveText(title);
  await expect(page.locator('.kb-crumbs')).toContainText(cat.name);
});
