import fs from 'node:fs';
import path from 'node:path';
import { expect, test } from '../helpers/fixtures';
import { pasteHTML } from '../helpers/paste';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { startStubImageServer, UNREACHABLE_IMAGE_URL } from '../helpers/stub-image-server';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

const tinyPNG = fs.readFileSync(path.join(__dirname, '..', 'fixtures', 'images', 'tiny.png'));

async function createCategory(page: Page, baseURL: string, name: string) {
  await page.goto(baseURL + '/kb/categories');
  await page.fill('#new-category-name', name);
  await page.click('.create-category-form button[type="submit"]');
  await ready(page);
}

async function startArticle(page: Page, baseURL: string, title: string) {
  const category = uniqueName('External images');
  await createCategory(page, baseURL, category);
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
}

// SPEC gate 1.19: publishing copies a reachable external image onto the
// NAS and the article ends up pointing at the local copy. @fresh: needs
// its own worker-local stub server and a real fetch reaching it, neither
// of which is available in BASE_URL mode (server.stubImageHost is empty
// there) or meaningful against the genuinely offline container (PLAN.md
// P1-38) — nothing reachable exists for this test to prove copies.
test('@fresh an external image reachable at publish time is copied locally', async ({ page, server }) => {
  const stub = await startStubImageServer(server.stubImageHost, tinyPNG);
  try {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await startArticle(page, server.baseURL, uniqueName('Reachable image article'));

    await pasteHTML(page, '#article-editor .ProseMirror', `<p>See below.</p><img src="${stub.url}" alt="Diagram">`);
    await expect(page.locator('#article-editor img')).toHaveCount(1);

    await page.click('#btn-publish');
    await ready(page);

    const img = page.locator('.kb-article-body img');
    await expect(img).toHaveCount(1);
    const src = await img.getAttribute('src');
    expect(src).toMatch(/^\/images\//);
  } finally {
    await stub.close();
  }
});

// SPEC gate 1.19 (failure path): when a copy can't be made, the article
// still publishes, the failed image is named and marked in place, and the
// rest of the text is saved.
test('an unreachable external image is named, marked, and the rest of the text is saved', async ({ page, server }) => {
  const imageURL = UNREACHABLE_IMAGE_URL;

  await signInAsNewPerson(page, server.baseURL, '/kb');
  await startArticle(page, server.baseURL, uniqueName('Unreachable image article'));

  await pasteHTML(page, '#article-editor .ProseMirror', `<p>Kept text.</p><img src="${imageURL}" alt="Missing">`);
  await expect(page.locator('#article-editor img')).toHaveCount(1);

  await page.click('#btn-publish');
  await ready(page);

  const body = page.locator('.kb-article-body');
  await expect(body).toContainText('Kept text.');
  await expect(body.locator(`div[data-missing-src="${imageURL}"]`)).toHaveCount(1);
  await expect(page.locator('.message')).toContainText(imageURL);
});
