import path from 'node:path';
import { expect, test } from './fixtures';
import { expectMatchesMockup } from './mockup';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

// The Knowledge Base (PLAN P4-12, gates 4.39-4.43): an article, the search
// results, and the editor, compared with docs/design/mockup.html.

type Page = import('@playwright/test').Page;

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color'];
const BOX = ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding'];

// An article with a paragraph, a table, a section heading and a paragraph
// carrying a word nothing else on the shared server contains.
async function publishArticle(page: Page, baseURL: string) {
  await signInAsNewPerson(page, baseURL, '/kb');
  await ready(page);
  const category = uniqueName('Office facts');
  const word = 'zq' + Math.random().toString(36).slice(2, 9);
  await page.goto(baseURL + '/kb/categories');
  await page.fill('#new-category-name', category);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
  const categoryID = await page.locator(`input[name="name"][value="${category}"]`).getAttribute('id');
  expect(categoryID).toBeTruthy();
  const id = categoryID!.replace('category-name-', '');

  const title = uniqueName('Printers');
  const res = await page.request.post(baseURL + '/kb/articles', {
    form: {
      category_id: id,
      title,
      body_html:
        '<p>We have three printers. The big office copier handles anything over 50 pages.</p>' +
        '<table><thead><tr><th>Printer</th><th>Where</th></tr></thead><tbody><tr><td>Office copier</td><td>Main office</td></tr></tbody></table>' +
        '<h2>Spares</h2>' +
        `<p>Spare ${word} lives in the grey cabinet by the kitchen door.</p>`,
    },
    headers: { origin: baseURL },
    maxRedirects: 0,
  });
  const location = res.headers()['location'];
  expect(location, 'the published article').toMatch(/\/kb\/articles\/\d+/);
  await page.goto(baseURL + location);
  await ready(page);
  return { title, category, word, url: location };
}

test('gate 4.39: an article title is large condensed capitals, its headings the same face smaller, and the body a calm readable face at a generous size', async ({
  page,
  server,
  mockup,
}) => {
  await publishArticle(page, server.baseURL);
  await expectMatchesMockup(mockup, page, { mockup: '.article h1', screen: 'kb', app: '.kb-article h1' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.article h2', screen: 'kb', app: '.kb-article-body h2' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.article', screen: 'kb', app: '.kb-article' }, ['display', 'row-gap', 'font-size', 'line-height', 'max-width']);
  await expectMatchesMockup(mockup, page, { mockup: '.article > p', screen: 'kb', app: '.kb-article-body p' }, [...TEXT, 'line-height', 'margin-top']);
  // Body text is the readable face; the display face is only the title, headings and table headers.
  await expect(page.locator('.kb-article-body p').first()).toHaveCSS('font-family', /Atkinson Hyperlegible Next/);
  await expect(page.locator('.kb-article h1')).toHaveCSS('font-family', /Big Shoulders Display/);
  await expect(page.locator('.kb-article-body td').first()).toHaveCSS('font-family', /Atkinson Hyperlegible Next/);
  // Larger than ordinary text, and the paper shows behind it.
  const size = await page.locator('.kb-article-body p').first().evaluate((el) => parseFloat(getComputedStyle(el).fontSize));
  expect(size).toBeGreaterThanOrEqual(17);
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(245, 243, 238)');
});

test('gate 4.40: an article table has a solid ink header row with white condensed capitals', async ({ page, server, mockup }) => {
  await publishArticle(page, server.baseURL);
  await expectMatchesMockup(mockup, page, { mockup: '.article th', screen: 'kb', app: '.kb-article-body th' }, [...TEXT, 'background-color', 'padding', 'text-align']);
  await expectMatchesMockup(mockup, page, { mockup: '.article td', screen: 'kb', app: '.kb-article-body td' }, [...TEXT, 'padding', 'border-bottom-width', 'border-bottom-color']);
  await expectMatchesMockup(mockup, page, { mockup: '.article table', screen: 'kb', app: '.kb-article-body table' }, ['border-collapse', 'width', 'font-size', 'background-color']);
  await expect(page.locator('.kb-article-body th').first()).toHaveCSS('background-color', 'rgb(20, 27, 45)');
  await expect(page.locator('.kb-article-body th').first()).toHaveCSS('color', 'rgb(255, 255, 255)');
});

test('gate 4.41: the byline reads Updated by name · date, with History on the right, then Archive, then a primary Edit', async ({
  page,
  server,
  mockup,
}) => {
  await publishArticle(page, server.baseURL);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by', screen: 'kb', app: '.kb-byline' }, ['display', 'align-items', 'column-gap', 'flex-wrap', 'font-size', 'color']);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by .date', screen: 'kb', app: '.kb-byline .date' }, [...TEXT, 'font-variant-numeric']);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by .btn:not(.primary)', screen: 'kb', app: '.kb-byline .btn:not(.primary)' }, [...TEXT, ...BOX]);
  await expectMatchesMockup(mockup, page, { mockup: '.article .by .btn.primary', screen: 'kb', app: '.kb-byline .btn.primary' }, [...TEXT, ...BOX]);

  const me = (await page.locator('.you strong').textContent())!;
  await expect(page.locator('.kb-byline > span').first()).toContainText(`Updated by ${me} ·`);
  await expect(page.locator('.kb-byline b')).toHaveText(me);

  const history = (await page.getByRole('link', { name: /^History/ }).boundingBox())!;
  const archive = (await page.getByRole('button', { name: 'Archive' }).boundingBox())!;
  const edit = (await page.getByRole('link', { name: 'Edit' }).boundingBox())!;
  expect(history.x + history.width).toBeLessThanOrEqual(archive.x + 2);
  expect(archive.x + archive.width).toBeLessThanOrEqual(edit.x + 2);
  const article = (await page.locator('.kb-article').boundingBox())!;
  expect(edit.x + edit.width).toBeGreaterThan(article.x + article.width - 6); // the right-hand end
  await expect(page.getByRole('link', { name: 'Edit' })).toHaveClass(/\bprimary\b/);
});

test('gate 4.42: search results are a bordered panel with a shadow — category in small capitals, title, matching words in accent, the selected one tinted', async ({
  page,
  server,
  mockup,
}) => {
  const a = await publishArticle(page, server.baseURL);
  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', a.word);
  const panel = page.locator('.search-panel');
  await expect(panel).toBeVisible();
  await expect(panel.locator('.search-result')).toHaveCount(1);
  await page.keyboard.press('ArrowDown');
  await expect(panel.locator('.search-result.active')).toHaveCount(1);

  await expectMatchesMockup(mockup, page, { mockup: '.results', screen: 'kb', app: '.search-panel' }, [...BOX.slice(0, 5), 'box-shadow']);
  // With one result it is always the selected one; its shape is the row's.
  await expectMatchesMockup(mockup, page, { mockup: '.results .r.sel', screen: 'kb', app: '.search-panel .search-result.active' }, ['background-color', 'padding', 'display', 'row-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.results .r .cat', screen: 'kb', app: '.search-result-category' }, [...TEXT]);
  await expectMatchesMockup(mockup, page, { mockup: '.results .r b', screen: 'kb', app: '.search-result-title' }, [...TEXT]);
  await expectMatchesMockup(mockup, page, { mockup: '.results .r p', screen: 'kb', app: '.search-snippet' }, [...TEXT, 'margin-top']);
  await expectMatchesMockup(mockup, page, { mockup: '.results mark', screen: 'kb', app: '.search-snippet mark' }, [...TEXT, 'background-color', 'padding', 'border-radius']);
  await expectMatchesMockup(mockup, page, { mockup: '.results .foot', screen: 'kb', app: '.search-panel-foot' }, [...TEXT, 'background-color', 'padding']);
  await expect(page.locator('.search-panel-foot')).toHaveText('1 article · press Enter to see all results');
  await expect(page.locator('.search-snippet mark').first()).toHaveCSS('background-color', 'rgb(255, 107, 26)');
  await expect(page.locator('.search-snippet mark').first()).toHaveCSS('color', 'rgb(20, 27, 45)');
  // Category first, then the title, then the snippet, in reading order on screen.
  const cat = (await panel.locator('.search-result-category').boundingBox())!;
  const title = (await panel.locator('.search-result-title').boundingBox())!;
  const snippet = (await panel.locator('.search-snippet').boundingBox())!;
  expect(cat.y).toBeLessThan(title.y);
  expect(title.y).toBeLessThan(snippet.y);
  await axeCheck(page);
});

test('gate 4.42: opening a result outlines the found paragraph, with a Clear highlights link', async ({ page, server, mockup }) => {
  const a = await publishArticle(page, server.baseURL);
  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', a.word);
  await page.locator('.search-panel .search-result').first().click();
  await ready(page);
  const hit = page.locator('.kb-highlighted').first();
  await expect(hit).toBeVisible();
  await expectMatchesMockup(mockup, page, { mockup: '.article .hit', screen: 'kb', app: '.kb-highlighted' }, ['background-color', 'border-radius', 'padding', 'margin-left', 'margin-right', 'outline-width', 'outline-style', 'outline-color']);
  await expectMatchesMockup(mockup, page, { mockup: '.article .hit mark', screen: 'kb', app: '.kb-highlighted mark' }, [...TEXT, 'background-color']);
  await expect(page.getByRole('button', { name: 'Clear highlights' })).toBeVisible();
  await page.getByRole('button', { name: 'Clear highlights' }).click();
  await expect(page.locator('.kb-highlighted')).toHaveCount(0);
});

test('gate 4.43: the editor bar is one bordered bar with Import from Word as a primary button and its icon at the right-hand end', async ({
  page,
  server,
  mockup,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Editor category');
  await page.goto(server.baseURL + '/kb/categories');
  await page.fill('#new-category-name', category);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
  await page.goto(server.baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');

  await expectMatchesMockup(mockup, page, { mockup: '.bar', screen: 'editor', app: '.editor-toolbar' }, [...BOX.slice(0, 5), 'padding', 'display', 'flex-wrap', 'column-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.bar .tb', screen: 'editor', app: '.editor-toolbar .tb' }, [...TEXT, 'background-color', 'border-radius', 'height']);
  await expectMatchesMockup(mockup, page, { mockup: '.bar .sep', screen: 'editor', app: '.editor-toolbar .sep' }, ['width', 'height', 'background-color']);
  await expectMatchesMockup(mockup, page, { mockup: '.bar .word', screen: 'editor', app: '#btn-import-word' }, [...TEXT, ...BOX, 'display', 'column-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.bar .word svg', screen: 'editor', app: '#btn-import-word .icon' }, ['width', 'height']);
  await expect(page.locator('#btn-import-word')).toHaveClass(/\bprimary\b/);
  await expect(page.locator('#btn-import-word .icon')).toBeVisible();
  // At the right-hand end of the bar, after every other control.
  const bar = (await page.locator('.editor-toolbar').boundingBox())!;
  const word = (await page.locator('#btn-import-word').boundingBox())!;
  expect(word.x + word.width).toBeGreaterThan(bar.x + bar.width - 12);
  for (const id of ['btn-h2', 'btn-bold', 'btn-image']) {
    const other = (await page.locator(`#${id}`).boundingBox())!;
    expect(other.x + other.width, id).toBeLessThanOrEqual(word.x + 2);
  }
  await expectMatchesMockup(mockup, page, { mockup: '.title-in', screen: 'editor', app: '#article-title' }, [...TEXT, 'line-height', 'border-bottom-width', 'border-bottom-color']);
  await expectMatchesMockup(mockup, page, { mockup: '.article.doc', screen: 'editor', app: '.article-editor' }, ['background-color', 'border-top-width', 'border-top-color', 'border-radius', 'padding', 'font-size']);
  await axeCheck(page);
  await expectNoSideScroll(page);
  expect(category).toBeTruthy();
});

test('gate 4.43: the import summary is a bordered notice with an IMPORTED stamp, and anything not brought in is a red dashed box in place', async ({
  page,
  server,
  mockup,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Import category');
  await page.goto(server.baseURL + '/kb/categories');
  await page.fill('#new-category-name', category);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
  await page.goto(server.baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });

  const chooser = page.waitForEvent('filechooser');
  await page.click('#btn-import-word');
  await (await chooser).setFiles(path.join(__dirname, '..', 'fixtures', 'docx', 'sample.docx'));

  const notice = page.locator('#editor-message');
  await expect(notice).toBeVisible();
  await expect(notice).toHaveClass(/\bimported\b/);
  await expect(notice.locator('.stamp')).toHaveText('Imported');
  await expect(notice).toContainText('is in the editor');
  await expect(notice).toContainText("couldn't be brought in");
  await expectMatchesMockup(mockup, page, { mockup: '.imported', screen: 'editor', app: '#editor-message.imported' }, [...BOX.slice(0, 5), 'padding', 'display', 'column-gap', 'font-size', 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.imported .stamp', screen: 'editor', app: '#editor-message .stamp' }, [...TEXT, ...BOX]);
  await expectMatchesMockup(mockup, page, { mockup: '.imported .mini', screen: 'editor', app: '#editor-message .mini' }, [...TEXT, ...BOX]);

  // The red dashed box, in place.
  const missing = page.locator('#article-editor .ProseMirror div[data-missing-kind]').first();
  await expect(missing).toBeVisible();
  await expectMatchesMockup(mockup, page, { mockup: '.missing', screen: 'editor', app: '#article-editor .ProseMirror div[data-missing-kind]' }, [
    'background-color',
    'border-top-width',
    'border-top-style',
    'border-top-color',
    'border-radius',
    'padding',
    'color',
    'font-size',
    'line-height',
  ]);
  await expect(missing).toHaveCSS('border-top-style', 'dashed');
  await expect(missing).toHaveCSS('border-top-color', 'rgb(198, 45, 45)');

  await axeCheck(page);
  // Dismiss clears it.
  await notice.getByRole('button', { name: 'Dismiss' }).click();
  await expect(notice).toBeHidden();
});

test('standard page checks for an article, its search results page, and the editor', async ({ page, server }) => {
  const a = await publishArticle(page, server.baseURL);
  await axeCheck(page);
  await expectNoSideScroll(page);
  await page.goto(server.baseURL + '/kb/search?q=' + encodeURIComponent(a.word));
  await ready(page);
  await expect(page.locator('.kb-search-results .search-result')).toHaveCount(1);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
