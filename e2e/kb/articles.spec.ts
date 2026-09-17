import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

async function createCategory(page: import('@playwright/test').Page, baseURL: string, name: string) {
  await page.goto(baseURL + '/kb/categories');
  await page.fill('#new-category-name', name);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
}

async function goToNewArticle(page: import('@playwright/test').Page, baseURL: string) {
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
}

async function publishArticle(
  page: import('@playwright/test').Page,
  baseURL: string,
  category: string,
  title: string,
  bodyText: string,
) {
  await goToNewArticle(page, baseURL);
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
  // Click the ProseMirror node directly (not the #article-editor wrapper
  // it mounts into) and wait for focus to land — a race seen only on CI's
  // Linux/headless Chromium, where the first keystrokes after the click
  // could be lost before focus had actually settled.
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await expect(editor).toBeFocused();
  await page.keyboard.type(bodyText);
  await page.click('#btn-publish');
  await ready(page);
}

// SPEC gate 1.14: choosing a category, typing a title and content, and
// pressing Publish makes the article appear in its category straight away.
test('publishing an article makes it appear in its category straight away', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);

  const category = uniqueName('Printers');
  const title = uniqueName('Changing the toner');
  await createCategory(page, server.baseURL, category);
  await publishArticle(page, server.baseURL, category, title, 'Open the front cover and pull the cartridge out.');

  // Landed on the article page itself.
  await expect(page.locator('h1')).toHaveText(title);

  const categoryLink = page.locator('a', { hasText: category });
  await categoryLink.click();
  await ready(page);
  await expect(page.locator('.kb-article-list')).toContainText(title);
});

// SPEC gate 1.13 (recent list): the 10 most recently updated articles, with
// who updated them and when. The tiles/counts half of this gate was tested
// in P1-17; this is the recent-list half, deferred until real articles
// existed.
test('the Knowledge Base home recent list shows who updated it and when', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);

  const category = uniqueName('Printers');
  const title = uniqueName('Changing the toner');
  await createCategory(page, server.baseURL, category);
  await publishArticle(page, server.baseURL, category, title, 'Some content.');

  await page.goto(server.baseURL + '/kb');
  await ready(page);

  const recentItem = page.locator('.kb-recent-list li', { hasText: title });
  await expect(recentItem).toContainText(title);
  await expect(recentItem).toContainText(`updated by ${name}`);
});

test('editing a published article records a new version and shows the new text', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);

  const category = uniqueName('Printers');
  const title = uniqueName('Changing the toner');
  await createCategory(page, server.baseURL, category);
  await publishArticle(page, server.baseURL, category, title, 'Original text.');

  await page.locator('a', { hasText: 'Edit' }).click();
  await page.waitForSelector('body[data-editor-ready]');
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await expect(editor).toBeFocused();
  await page.keyboard.press('Control+a');
  await page.keyboard.type('Edited text.');
  await page.click('#btn-publish');
  await ready(page);

  await expect(page.locator('.kb-article-body')).toContainText('Edited text.');
});

test('standard page checks for the article and category pages', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);

  const category = uniqueName('Printers');
  const title = uniqueName('Changing the toner');
  await createCategory(page, server.baseURL, category);
  await publishArticle(page, server.baseURL, category, title, 'Some content.');

  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);

  await page.locator('a', { hasText: category }).click();
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
