import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
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

async function publishNewArticle(page: Page, baseURL: string, title: string, bodyText: string) {
  const category = uniqueName('Printers');
  await createCategory(page, baseURL, category);
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
  await typeIntoEditor(page, bodyText);
  await page.click('#btn-publish');
  await ready(page);
  return category;
}

async function editCurrentArticle(page: Page, bodyText: string) {
  await page.locator('a', { hasText: 'Edit' }).click();
  await page.waitForSelector('body[data-editor-ready]');
  await typeIntoEditor(page, bodyText);
  await page.click('#btn-publish');
  await ready(page);
}

// SPEC gate 1.22: every publish is recorded; History lists each version
// with name, date and time, newest first.
test('History lists each version newest first with name and time', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('Toner article');
  await publishNewArticle(page, server.baseURL, title, 'Original text.');
  await editCurrentArticle(page, 'Edited text.');

  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);

  const rows = page.locator('.kb-history-list li');
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText('Version 2');
  await expect(rows.nth(0)).toContainText('edited');
  await expect(rows.nth(1)).toContainText('Version 1');
  await expect(rows.nth(1)).toContainText('created');
  for (const i of [0, 1]) {
    await expect(rows.nth(i)).toContainText(name);
  }
});

// SPEC gate 1.23: opening an old version shows it as it was; "Restore
// this version" makes it current and adds a history entry with the
// restorer's name.
test('an old version displays as it was and can be restored', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('Restorable article');
  await publishNewArticle(page, server.baseURL, title, 'Original text.');
  await editCurrentArticle(page, 'Edited text.');

  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await page.locator('a', { hasText: 'Version 1' }).click();
  await ready(page);
  await expect(page.locator('.kb-article-body')).toContainText('Original text.');

  const restorerName = await signInAsNewPerson(page, server.baseURL, '/kb');
  await page.goto(server.baseURL + '/kb');
  await page.locator('.kb-recent-list a', { hasText: title }).click();
  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await page.locator('a', { hasText: 'Version 1' }).click();
  await ready(page);
  await page.locator('button', { hasText: 'Restore this version' }).click();
  await ready(page);

  await expect(page.locator('.kb-article-body')).toContainText('Original text.');

  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  const newest = page.locator('.kb-history-list li').first();
  await expect(newest).toContainText('restored');
  await expect(newest).toContainText(restorerName);
});

// SPEC gate 1.24: archiving removes an article from its category and
// lists it in Archived; restoring puts it back.
test('archiving hides an article from its category and lists it in Archived; unarchiving brings it back', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('Archivable article');
  const category = await publishNewArticle(page, server.baseURL, title, 'Some text.');

  await page.locator('button', { hasText: 'Archive' }).click();
  await ready(page);
  await expect(page.locator('.message')).toContainText('archived');

  await page.locator('a', { hasText: category }).click();
  await ready(page);
  await expect(page.locator('a', { hasText: title })).toHaveCount(0);

  await page.goto(server.baseURL + '/kb/archived');
  await ready(page);
  await expect(page.locator('a', { hasText: title })).toHaveCount(1);

  await page.locator('a', { hasText: title }).click();
  await ready(page);
  await page.locator('button', { hasText: 'Unarchive' }).click();
  await ready(page);

  await page.locator('a', { hasText: category }).click();
  await ready(page);
  await expect(page.locator('.kb-article-list')).toContainText(title);

  await page.goto(server.baseURL + '/kb/archived');
  await ready(page);
  await expect(page.locator('a', { hasText: title })).toHaveCount(0);
});

// SPEC gate 1.05 (History part): renaming a person changes their name in
// History; removing them still shows their old name there.
test('History shows a renamed person\'s new name, and a removed person\'s old name', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('People page article');
  await publishNewArticle(page, server.baseURL, title, 'Some text.');

  await page.goto(server.baseURL + '/settings/people');
  await ready(page);
  const newName = uniqueName('Renamed');
  const row = page.locator('.settings-person-row', { has: page.locator(`input[value="${name}"]`) });
  await row.locator('input[name="name"]').fill(newName);
  await row.locator('.rename-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/kb');
  await page.locator('.kb-recent-list a', { hasText: title }).click();
  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await expect(page.locator('.kb-history-list')).toContainText(newName);
  await expect(page.locator('.kb-history-list')).not.toContainText(name);

  await page.goto(server.baseURL + '/settings/people');
  await ready(page);
  const rowAfterRename = page.locator('.settings-person-row', { has: page.locator(`input[value="${newName}"]`) });
  await rowAfterRename.locator('.remove-form button[type="submit"]').click();
  await ready(page);

  await page.goto(server.baseURL + '/who');
  await ready(page);
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await page.goto(server.baseURL + '/kb');
  await page.locator('.kb-recent-list a', { hasText: title }).click();
  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await expect(page.locator('.kb-history-list')).toContainText(newName);
});

test('standard page checks for History and a version view', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('Checked article');
  await publishNewArticle(page, server.baseURL, title, 'Some text.');
  await editCurrentArticle(page, 'More text.');

  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);

  await page.locator('a', { hasText: 'Version 1' }).click();
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
