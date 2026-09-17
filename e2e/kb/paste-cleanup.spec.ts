import fs from 'node:fs';
import path from 'node:path';
import { expect, test } from '../helpers/fixtures';
import { pasteHTML } from '../helpers/paste';
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

async function startArticle(page: Page, baseURL: string, title: string) {
  const category = uniqueName('Paste cleanup');
  await createCategory(page, baseURL, category);
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
}

function readFixture(name: string): string {
  return fs.readFileSync(path.join(__dirname, '..', 'fixtures', 'paste', name), 'utf8');
}

// SPEC gate 1.18 (web page half): pasting a sample web page keeps
// headings, bold, lists, links, tables and images; fonts, colours and
// sizes are removed.
test('pasting a web page keeps structure and images, publishing strips all styling', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('Web page paste article');
  await startArticle(page, server.baseURL, title);

  await pasteHTML(page, '#article-editor .ProseMirror', readFixture('web-page.html'));

  const editor = page.locator('#article-editor');
  await expect(editor.locator('h2')).toHaveText('Printer maintenance schedule');
  await expect(editor.locator('strong')).toContainText('toner');
  await expect(editor.locator('ul li')).toHaveCount(3);
  await expect(editor.locator('a[href="https://printers.example/manual"]')).toHaveCount(1);
  await expect(editor.locator('table tr')).toHaveCount(2);
  await expect(editor.locator('img[src="https://printers.example/toner-diagram.png"]')).toHaveCount(1);

  await page.click('#btn-publish');
  await ready(page);

  const body = page.locator('.kb-article-body');
  await expect(body.locator('h2')).toHaveText('Printer maintenance schedule');
  await expect(body.locator('strong')).toContainText('toner');
  await expect(body.locator('ul li')).toHaveCount(3);
  await expect(body.locator('a[href="https://printers.example/manual"]')).toHaveCount(1);
  await expect(body.locator('table tr')).toHaveCount(2);
  await expect(body.locator('a[rel="noopener"]')).toHaveCount(1);

  // No presentation survives publish, anywhere in the article.
  await expect(body.locator('[style]')).toHaveCount(0);
  await expect(body.locator('[class]')).toHaveCount(0);
  await expect(body.locator('font')).toHaveCount(0);

  // The pasted image isn't yet a local /images/ file (that's P1-23's job),
  // so today's sanitiser correctly drops it rather than publish a link to
  // another host.
  await expect(body.locator('img')).toHaveCount(0);
});

// SPEC gate 1.18 (Word half): pictures that can't come across are marked
// in place with the exact message; the rest of the content and the
// style/class/font stripping behave the same as the web-page case.
test('pasting from Word marks unreachable pictures with the exact message', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('Word paste article');
  await startArticle(page, server.baseURL, title);

  await pasteHTML(page, '#article-editor .ProseMirror', readFixture('word.html'));

  await expect(page.locator('#editor-message')).toContainText(
    "Some pictures from Word couldn't be pasted. Use Import from Word to bring them in.",
  );

  const editor = page.locator('#article-editor');
  await expect(editor.locator('img')).toHaveCount(0);
  await expect(editor.locator('div[data-missing-kind="picture"]')).toHaveCount(1);
  await expect(editor).toContainText('Printer Supplies Policy');
  await expect(editor.locator('strong', { hasText: 'toner' })).toHaveCount(1);

  await page.click('#btn-publish');
  await ready(page);

  const body = page.locator('.kb-article-body');
  await expect(body.locator('div[data-missing-kind="picture"]')).toHaveCount(1);
  await expect(body).toContainText('Printer Supplies Policy');
  await expect(body.locator('[style]')).toHaveCount(0);
  await expect(body.locator('[class]')).toHaveCount(0);
  await expect(body.locator('font')).toHaveCount(0);
});
