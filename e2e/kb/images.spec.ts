import fs from 'node:fs';
import path from 'node:path';
import { expect, test } from '../helpers/fixtures';
import { generateOversizePNG } from '../helpers/images';
import { dropFile, fixtureFromBytes, loadFixtureFile, pasteFile } from '../helpers/paste';
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
  const category = uniqueName('Screenshots');
  await createCategory(page, baseURL, category);
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
}

// SPEC gate 1.16: pasting a screenshot shows it in the editor within 3
// seconds; after publishing, the image is served by Comp HQ itself and the
// uploaded file exists on disk. The disk check needs this test's own
// worker's local /data folder, so it's excluded under BASE_URL mode (see
// e2e/helpers/fixtures.ts).
test('@fresh pasting a screenshot shows it within 3s, publishes locally-served, and lands on disk', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const title = uniqueName('Screenshot article');
  await startArticle(page, server.baseURL, title);

  const png = loadFixtureFile('images/tiny.png', 'image/png');
  await pasteFile(page, '#article-editor .ProseMirror', png);

  const img = page.locator('#article-editor img');
  await expect(img).toHaveCount(1, { timeout: 3000 });
  const src = await img.getAttribute('src');
  expect(src).toMatch(/^\/images\/[0-9a-f]+\.png$/);

  const filename = src!.replace('/images/', '');
  const sha = filename.split('.')[0];
  const onDisk = path.join(server.dataDir, 'images', sha.slice(0, 2), filename);
  expect(fs.existsSync(onDisk)).toBe(true);

  const otherHostRequests: string[] = [];
  page.on('request', (req) => {
    if (new URL(req.url()).host !== new URL(server.baseURL).host) {
      otherHostRequests.push(req.url());
    }
  });

  await page.click('#btn-publish');
  await ready(page);

  await expect(page.locator('.kb-article-body img')).toHaveAttribute('src', src!);
  expect(otherHostRequests, JSON.stringify(otherHostRequests)).toEqual([]);
});

// SPEC gate 1.17: dragging in a JPEG, PNG, GIF or WebP up to 20 MB inserts
// it.
for (const [file, mime] of [
  ['tiny.png', 'image/png'],
  ['tiny.jpg', 'image/jpeg'],
  ['tiny.gif', 'image/gif'],
  ['tiny.webp', 'image/webp'],
] as const) {
  test(`dropping a ${mime} image inserts it`, async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await startArticle(page, server.baseURL, uniqueName('Drop article'));

    const fixture = loadFixtureFile(`images/${file}`, mime);
    await dropFile(page, '#article-editor .ProseMirror', fixture);

    const img = page.locator('#article-editor img');
    await expect(img).toHaveCount(1, { timeout: 3000 });
    await expect(img).toHaveAttribute('src', /^\/images\//);
  });
}

test('a file larger than 20 MB shows a plain message and changes nothing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await startArticle(page, server.baseURL, uniqueName('Oversize article'));

  const oversize = fixtureFromBytes('big.png', 'image/png', generateOversizePNG());
  await dropFile(page, '#article-editor .ProseMirror', oversize);

  await expect(page.locator('#editor-message')).toContainText('20 MB', { timeout: 5000 });
  await expect(page.locator('#article-editor img')).toHaveCount(0);
});

test('a file that is neither an image nor a Word document shows a plain message and changes nothing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await startArticle(page, server.baseURL, uniqueName('Wrong type article'));

  const txt = loadFixtureFile('images/not-an-image.txt', 'text/plain');
  await dropFile(page, '#article-editor .ProseMirror', txt);

  await expect(page.locator('#editor-message')).toContainText("isn't a JPEG, PNG, GIF or WebP", { timeout: 5000 });
  await expect(page.locator('#article-editor img')).toHaveCount(0);
});

test('dropping a .docx imports it instead of uploading it as an image', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await startArticle(page, server.baseURL, uniqueName('Docx drop article'));

  // Not a real .docx — proves the drop is routed to Import from Word
  // (SPEC gate 1.45: "dragging a .docx file onto the editor does the
  // same" as the button) rather than the image-upload path, via the
  // exact SPEC gate 1.51 message a genuinely unreadable file gets.
  // e2e/kb/import-word.spec.ts covers the real end-to-end import.
  const docx = fixtureFromBytes(
    'sample.docx',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    Buffer.from('not a real docx, just proving the message path'),
  );
  await dropFile(page, '#article-editor .ProseMirror', docx);

  await expect(page.locator('#editor-message')).toContainText("Comp HQ can't open this file", { timeout: 5000 });
  await expect(page.locator('#article-editor img')).toHaveCount(0);
});
