import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

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

// SPEC gate 1.21: if two people edit the same article and both publish,
// the second person sees the exact conflict message, their text is still
// on screen, "Publish mine anyway" saves it, and both versions are in
// History.
test('a second publisher sees the conflict message, keeps their text, and can publish anyway', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  const title = uniqueName('Toner article');
  await page.goto(server.baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
  await typeIntoEditor(page, 'Original text.');
  await page.click('#btn-publish');
  await ready(page);
  const articleURL = page.url();

  // A second person opens the same article for editing, in a separate
  // browser context, before the first person's edit lands.
  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/kb');
  await otherPage.goto(articleURL);
  await otherPage.locator('a', { hasText: 'Edit' }).click();
  await otherPage.waitForSelector('body[data-editor-ready]');

  // The first person edits and publishes, becoming version 2.
  await page.goto(articleURL);
  await page.locator('a', { hasText: 'Edit' }).click();
  await page.waitForSelector('body[data-editor-ready]');
  await typeIntoEditor(page, 'First editor\'s text.');
  await page.click('#btn-publish');
  await ready(page);

  // The second person, still holding the stale version_no from their
  // already-open editor, publishes their own edit.
  await typeIntoEditor(otherPage, 'Second editor\'s text.');
  await otherPage.click('#btn-publish');
  await ready(otherPage);

  await expect(otherPage.locator('.message').first()).toContainText(
    'Someone else changed this article while you were editing.',
  );
  await expect(otherPage.locator('#article-editor')).toContainText("Second editor's text.");

  await otherPage.click('#btn-publish'); // now labelled "Publish mine anyway"
  await ready(otherPage);

  await expect(otherPage.locator('.kb-article-body')).toContainText("Second editor's text.");

  await otherPage.locator('a', { hasText: 'History' }).click();
  await ready(otherPage);
  const rows = otherPage.locator('.kb-history-list li');
  await expect(rows).toHaveCount(3);

  await otherContext.close();
});
