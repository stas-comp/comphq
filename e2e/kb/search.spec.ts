import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// A single alphanumeric token, unique per call, safe to use as a search
// word: BuildQuery tokenises on Unicode letter/digit runs, so a plain
// uniqueName() (which has dashes) would split into several words. Workers
// reuse one server across many tests, so search terms need to be unique
// enough that one test's articles can never be matched by another's query.
function uniqueWord(prefix: string): string {
  return prefix + Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

async function createCategory(page: Page, baseURL: string, name: string) {
  await page.goto(baseURL + '/kb/categories');
  await page.fill('#new-category-name', name);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
}

async function typeIntoEditor(page: Page, text: string) {
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await expect(editor).toBeFocused();
  await page.keyboard.press('Control+a');
  await page.keyboard.type(text);
}

async function publishArticle(page: Page, baseURL: string, category: string, title: string, bodyText: string) {
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
  await typeIntoEditor(page, bodyText);
  await page.click('#btn-publish');
  await ready(page);
}

async function searchAndWaitForPanel(page: Page, term: string) {
  await page.fill('#search-box', term);
  await expect(page.locator('.search-panel')).toBeVisible();
}

// SPEC gate 1.28, the exact B7 script: type "toner" (here, a unique
// stand-in word); the first result contains a <mark>; click it; the URL
// has #b-N; that block is in the viewport, has the highlight class, and
// contains a <mark> with the matched word.
test('gate 1.28: search result scrolls to and highlights the matching passage', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  const word = uniqueWord('toner');
  const title = uniqueName('Printer troubleshooting');
  await publishArticle(page, server.baseURL, category, title, `Ask Sam about the ${word} cartridges before ordering more.`);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await searchAndWaitForPanel(page, word);

  const firstResult = page.locator('.search-result').first();
  await expect(firstResult.locator('mark')).toHaveText(word);

  await firstResult.click();
  await ready(page);

  expect(page.url()).toMatch(/#b-\d+$/);
  const blockID = /#b-(\d+)$/.exec(page.url())![1];
  const target = page.locator(`[data-b="${blockID}"]`);
  await expect(target).toHaveClass(/kb-highlighted/);
  await expect(target.locator('mark')).toHaveText(word);
  await expect(target).toBeInViewport();
});

// SPEC gate 1.26: up to 20 results, most relevant first, a title match
// ranking above a body-only match; nothing appears for fewer than 2
// letters.
test('gate 1.26: title matches rank above body matches, and short input shows nothing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  const word = uniqueWord('widget');

  const bodyOnlyTitle = uniqueName('Body match article');
  await publishArticle(page, server.baseURL, category, bodyOnlyTitle, `This article mentions ${word} once in passing.`);
  const titleMatchTitle = `${word} instructions`;
  await publishArticle(page, server.baseURL, category, titleMatchTitle, 'Nothing relevant in the body.');

  await page.goto(server.baseURL + '/kb');
  await ready(page);

  await page.fill('#search-box', word[0]);
  await page.waitForTimeout(300);
  await expect(page.locator('.search-panel')).toHaveCount(0);

  await searchAndWaitForPanel(page, word);
  const results = page.locator('.search-result');
  await expect(results).toHaveCount(2);
  await expect(results.first()).toContainText(titleMatchTitle);
  await expect(results.nth(1)).toContainText(bodyOnlyTitle);
});

// SPEC gate 1.27: each result shows the title, its category, and the
// matching passage with the search words highlighted.
test('gate 1.27: a result shows the title, category and a highlighted passage', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printer supplies');
  await createCategory(page, server.baseURL, category);
  const word = uniqueWord('cartridge');
  const title = uniqueName('Supply article');
  await publishArticle(page, server.baseURL, category, title, `Order a new ${word} when the light comes on.`);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await searchAndWaitForPanel(page, word);

  const result = page.locator('.search-result').first();
  await expect(result.locator('.search-result-title')).toHaveText(title);
  await expect(result.locator('.search-result-category')).toHaveText(category);
  await expect(result.locator('.search-snippet mark')).toHaveText(word);
});

// SPEC gate 1.29: stemming ("printers" finds "printer") and prefix
// matching ("ton" finds "toner").
test('gate 1.29: stemming and prefix matching find related words', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);

  const stem = uniqueWord('grinder');
  const stemTitle = uniqueName('Stemming article');
  await publishArticle(page, server.baseURL, category, stemTitle, `The ${stem} needs oiling.`);

  const prefixWord = uniqueWord('toner');
  const prefixTitle = uniqueName('Prefix article');
  await publishArticle(page, server.baseURL, category, prefixTitle, `Buy more ${prefixWord} soon.`);

  await page.goto(server.baseURL + '/kb');
  await ready(page);

  // Plural of the indexed word.
  await searchAndWaitForPanel(page, stem + 's');
  await expect(page.locator('.search-result')).toContainText(stemTitle);

  await page.fill('#search-box', '');
  await page.locator('.search-panel').waitFor({ state: 'detached' });

  // A short prefix of the indexed word.
  await searchAndWaitForPanel(page, prefixWord.slice(0, prefixWord.length - 3));
  await expect(page.locator('.search-result')).toContainText(prefixTitle);
});

// SPEC gate 1.30: a newly published article is found by the very next
// search; a word added in an edit is found next search, and a word
// removed no longer is.
test('gate 1.30: publishing and editing are reflected in the very next search', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);

  const original = uniqueWord('original');
  const title = uniqueName('Freshness article');
  await publishArticle(page, server.baseURL, category, title, `This mentions ${original} right away.`);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await searchAndWaitForPanel(page, original);
  await expect(page.locator('.search-result')).toContainText(title);

  // Edit: remove the original word, add a new one. Click the search
  // panel's own link rather than a bare text match — the kb home page's
  // recently-updated list also links to this article by the same title.
  const added = uniqueWord('added');
  await page.locator('.search-panel .search-result', { hasText: title }).click();
  await page.locator('a', { hasText: 'Edit' }).click();
  await page.waitForSelector('body[data-editor-ready]');
  await typeIntoEditor(page, `Now it mentions ${added} instead.`);
  await page.click('#btn-publish');
  await ready(page);

  await page.goto(server.baseURL + '/kb');
  await ready(page);

  await searchAndWaitForPanel(page, added);
  await expect(page.locator('.search-result')).toContainText(title);

  await page.fill('#search-box', original);
  await page.waitForTimeout(400);
  const panel = page.locator('.search-panel');
  await expect(panel).toBeVisible();
  await expect(panel).not.toContainText(title);
});

// SPEC gate 1.31: archived articles never appear in search results.
test('gate 1.31: an archived article is excluded from search', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  const word = uniqueWord('archivable');
  const title = uniqueName('Archivable article');
  await publishArticle(page, server.baseURL, category, title, `This contains ${word} for the test.`);

  await page.locator('button', { hasText: 'Archive' }).click();
  await ready(page);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await page.fill('#search-box', word);
  await page.waitForTimeout(400);
  await expect(page.locator('.search-panel')).toContainText('No articles match');
});

// SPEC gate 1.32: no-match text, and typing symbols never causes an
// error.
test('gate 1.32: no-match message and symbol fuzzing cause no errors', async ({ page, server }) => {
  const consoleErrors: string[] = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });
  const pageErrors: string[] = [];
  page.on('pageerror', (err) => pageErrors.push(String(err)));

  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);

  const nonsense = uniqueWord('nomatch');
  await searchAndWaitForPanel(page, nonsense);
  await expect(page.locator('.search-panel')).toContainText('No articles match');
  await expect(page.locator('.search-panel')).toContainText(nonsense);

  for (const symbols of ['"', '*', '(', '-', ':', 'NEAR', 'AND', 'OR', '^', "'", '" * ( - : NEAR AND OR ^ \'']) {
    await page.fill('#search-box', symbols);
    await page.waitForTimeout(300);
  }

  expect(consoleErrors, JSON.stringify(consoleErrors)).toEqual([]);
  expect(pageErrors, JSON.stringify(pageErrors)).toEqual([]);
});

test('standard page checks for the search results page, and axe with the panel open', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  const word = uniqueWord('standard');
  const title = uniqueName('Standard checks article');
  await publishArticle(page, server.baseURL, category, title, `This contains ${word} for the test.`);

  await page.goto(server.baseURL + '/kb/search?q=' + encodeURIComponent(word));
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);

  await page.goto(server.baseURL + '/kb');
  await ready(page);
  await searchAndWaitForPanel(page, word);
  await axeCheck(page);
});
