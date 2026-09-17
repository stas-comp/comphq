import path from 'node:path';
import { expect, test } from '../helpers/fixtures';
import { pasteHTML } from '../helpers/paste';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { UNREACHABLE_IMAGE_URL } from '../helpers/stub-image-server';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

const FIXTURES_DIR = path.join(__dirname, '..', 'fixtures', 'docx');

async function createCategory(page: Page, baseURL: string, name: string) {
  await page.goto(baseURL + '/kb/categories');
  await page.fill('#new-category-name', name);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
}

async function goToNewArticleIn(page: Page, baseURL: string, category: string) {
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
}

async function publishArticle(page: Page, baseURL: string, category: string, title: string, bodyText: string) {
  await goToNewArticleIn(page, baseURL, category);
  await page.fill('#article-title', title);
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await expect(editor).toBeFocused();
  await page.keyboard.type(bodyText);
  await page.click('#btn-publish');
  await ready(page);
}

async function importViaButton(page: Page, absPath: string) {
  const chooserPromise = page.waitForEvent('filechooser');
  await page.click('#btn-import-word');
  const chooser = await chooserPromise;
  await chooser.setFiles(absPath);
}

// SPEC gate 1.36: Settings > Content check shows how many articles
// contain images not stored on the NAS, and lists them. The exact count
// (0, N) is a Go-level test (internal/kb's TestListContentCheck*) run
// against an isolated in-memory database — here, against the shared
// per-worker server every spec file in this run reuses, assertions are
// scoped to this test's own uniquely named article rather than an
// absolute total, the same way every other cross-file-shared-server test
// in this suite avoids asserting a global count (e.g.
// e2e/kb/categories.spec.ts's tile-count test scopes to its own category).
test('a clean article is not listed in Content check', async ({ page, server }) => {
  const category = uniqueName('Content check clean');
  const title = uniqueName('Clean article');
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await createCategory(page, server.baseURL, category);
  await publishArticle(page, server.baseURL, category, title, 'Nothing wrong here.');

  await page.goto(server.baseURL + '/settings/content-check');
  await ready(page);
  await expect(page.locator('#content-check-count')).toBeVisible();
  await expect(page.locator('a', { hasText: title })).toHaveCount(0);
});

test('an article with a failed external image is listed in Content check', async ({ page, server }) => {
  const category = uniqueName('Content check image');
  const title = uniqueName('Unreachable image article');
  const imageURL = UNREACHABLE_IMAGE_URL;

  await signInAsNewPerson(page, server.baseURL, '/kb');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);
  await page.fill('#article-title', title);
  await pasteHTML(page, '#article-editor .ProseMirror', `<p>Kept text.</p><img src="${imageURL}" alt="Missing">`);
  await expect(page.locator('#article-editor img')).toHaveCount(1);
  await page.click('#btn-publish');
  await ready(page);

  await page.goto(server.baseURL + '/settings/content-check');
  await ready(page);
  await expect(page.locator('a', { hasText: title })).toHaveCount(1);
});

// SPEC gates 1.36/1.49: an imported article with an unresolved placeholder
// (here, a chart) is listed until the placeholder is removed and the
// article is republished, at which point it drops off the list.
test('an imported article with a chart placeholder is listed until fixed and republished', async ({ page, server }) => {
  const category = uniqueName('Content check import');
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);

  await importViaButton(page, path.join(FIXTURES_DIR, 'sample.docx'));
  await expect(page.locator('#article-title')).toHaveValue('Printer Supplies Handbook', { timeout: 10_000 });
  await expect(page.locator('#article-editor div[data-missing-kind="chart"]')).toHaveCount(1);

  const title = uniqueName('Imported with a chart');
  await page.fill('#article-title', title);
  await page.click('#btn-publish');
  await ready(page);

  await page.goto(server.baseURL + '/settings/content-check');
  await ready(page);
  await expect(page.locator('a', { hasText: title })).toHaveCount(1);

  // Fix it: open the article, replace the whole body with clean text, and
  // republish.
  await page.locator('a', { hasText: title }).click();
  await ready(page);
  await page.locator('a', { hasText: 'Edit' }).click();
  await page.waitForSelector('body[data-editor-ready]');
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await expect(editor).toBeFocused();
  await page.keyboard.press('Control+a');
  await page.keyboard.type('The chart was removed.');
  await page.click('#btn-publish');
  await ready(page);

  await page.goto(server.baseURL + '/settings/content-check');
  await ready(page);
  await expect(page.locator('a', { hasText: title })).toHaveCount(0);
});
