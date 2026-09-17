import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
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

// Drives the real "Import from Word" button through Playwright's file
// chooser API (SPEC gate 1.45), the same way a person clicking it and
// picking a file in the OS dialog would.
async function importViaButton(page: Page, absPath: string) {
  const chooserPromise = page.waitForEvent('filechooser');
  await page.click('#btn-import-word');
  const chooser = await chooserPromise;
  await chooser.setFiles(absPath);
}

async function importViaDrop(page: Page, absPath: string) {
  const buffer = fs.readFileSync(absPath);
  await page.evaluate(
    async ({ name, base64 }) => {
      const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
      const file = new File([bytes], name, {
        type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      });
      const dt = new DataTransfer();
      dt.items.add(file);
      const el = document.querySelector('#article-editor .ProseMirror') as HTMLElement;
      const rect = el.getBoundingClientRect();
      const event = new DragEvent('drop', {
        bubbles: true,
        cancelable: true,
        clientX: rect.x + rect.width / 2,
        clientY: rect.y + rect.height / 2,
      });
      Object.defineProperty(event, 'dataTransfer', { value: dt });
      el.dispatchEvent(event);
    },
    { name: path.basename(absPath), base64: buffer.toString('base64') },
  );
}

// Asserts everything gate 1.46/1.47/1.48/1.49 promise for sample.docx,
// regardless of whether it arrived by button or by drop.
async function expectSampleDocxImported(page: Page, server: { dataDir: string }) {
  await expect(page.locator('#article-title')).toHaveValue('Printer Supplies Handbook', { timeout: 10_000 });

  const editorEl = page.locator('#article-editor');
  await expect(editorEl.locator('h2', { hasText: 'Getting started' })).toHaveCount(1);
  await expect(editorEl.locator('h3', { hasText: 'Ordering toner' })).toHaveCount(1);

  // Nested lists (SPEC gate 1.46: "bulleted and numbered lists, including
  // indented levels").
  await expect(editorEl.locator('ul li ul li', { hasText: 'Look on the top shelf' })).toHaveCount(1);
  await expect(editorEl.locator('ol li ol li', { hasText: 'cc the office manager' })).toHaveCount(1);

  // Links.
  await expect(editorEl.locator('a', { hasText: "the supplier's ordering page" })).toHaveCount(1);
  await expect(editorEl.locator('a', { hasText: 'full manual' })).toHaveCount(1);

  // Table with a horizontal and a vertical merge.
  await expect(editorEl.locator('table th[colspan="2"]')).toHaveCount(1);
  await expect(editorEl.locator('table td[rowspan="2"]')).toHaveCount(1);

  // Two real pictures, served locally with alt text and a file on disk
  // (SPEC gate 1.47). The on-disk check is skipped in BASE_URL mode
  // (server.dataDir is empty there — PLAN.md P1-38 runs this file against
  // a container whose filesystem the test process can't see); everything
  // else about gate 1.47 still runs unconditionally.
  const imgs = editorEl.locator('img');
  await expect(imgs).toHaveCount(2);
  for (let i = 0; i < 2; i++) {
    const src = await imgs.nth(i).getAttribute('src');
    expect(src).toMatch(/^\/images\/[0-9a-f]+\.(png|jpg)$/);
    const alt = await imgs.nth(i).getAttribute('alt');
    expect(alt).toBeTruthy();
    if (server.dataDir) {
      const filename = src!.replace('/images/', '');
      const sha = filename.split('.')[0];
      const onDisk = path.join(server.dataDir, 'images', sha.slice(0, 2), filename);
      expect(fs.existsSync(onDisk)).toBe(true);
    }
  }

  // Unsupported items each become a placeholder (SPEC gate 1.49): the
  // EMF picture, the chart, the textless shape, the equation. SmartArt
  // isn't among them here — its mc:Fallback happens to be plain text.
  await expect(editorEl.locator('div[data-missing-kind="picture"]')).toHaveCount(1);
  await expect(editorEl.locator('div[data-missing-kind="chart"]')).toHaveCount(1);
  await expect(editorEl.locator('div[data-missing-kind="shape"]')).toHaveCount(1);
  await expect(editorEl.locator('div[data-missing-kind="equation"]')).toHaveCount(1);

  // The text box's own text, and tracked changes resolved as if accepted.
  await expect(editorEl).toContainText('Note: check toner levels weekly.');
  await expect(editorEl).toContainText('please order two boxes');
  await expect(editorEl).not.toContainText('order one box');

  // The 1.49 summary message.
  await expect(page.locator('#editor-message')).toContainText("couldn't be brought in");
}

test('importing sample.docx by button shows headings, lists, table spans, pictures, placeholders and the title — without publishing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);

  await importViaButton(page, path.join(FIXTURES_DIR, 'sample.docx'));
  await expectSampleDocxImported(page, server);

  // Never published (SPEC B4: "the endpoint never creates or changes an
  // article"): leaving without publishing, the category has no article
  // with this title.
  await page.goto(server.baseURL + '/kb');
  await page.locator('a', { hasText: category }).click();
  await ready(page);
  await expect(page.locator('a', { hasText: 'Printer Supplies Handbook' })).toHaveCount(0);
});

test('importing sample.docx by drop shows the same result as the button', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);

  await importViaDrop(page, path.join(FIXTURES_DIR, 'sample.docx'));
  await expectSampleDocxImported(page, server);
});

// SPEC gate 1.45: "A .docx downloaded from Google Docs imports too."
test('googledocs.docx imports', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);

  await importViaButton(page, path.join(FIXTURES_DIR, 'googledocs.docx'));
  await expect(page.locator('#article-title')).toHaveValue('Office Wi-Fi Guide', { timeout: 10_000 });
  await expect(page.locator('#article-editor h2', { hasText: 'Connecting a laptop' })).toHaveCount(1);
});

// SPEC gate 1.50: replacing existing content asks first; Cancel changes
// nothing; publishing afterwards adds a History version whose previous
// version still restores correctly.
test('importing into an existing article confirms the replace, and the old version still restores after publishing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);

  const title = uniqueName('Existing article');
  await page.fill('#article-title', title);
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await page.keyboard.type('Original hand-typed content.');
  await page.click('#btn-publish');
  await ready(page);

  await page.locator('a', { hasText: 'Edit' }).click();
  await page.waitForSelector('body[data-editor-ready]');

  // Cancel: dismissing the confirm changes nothing.
  let dialogMessage = '';
  page.once('dialog', async (dialog) => {
    dialogMessage = dialog.message();
    await dialog.dismiss();
  });
  await importViaButton(page, path.join(FIXTURES_DIR, 'sample.docx'));
  expect(dialogMessage).toBe("Replace what's in the editor with this document?");
  await expect(page.locator('#article-editor')).toContainText('Original hand-typed content.');
  await expect(page.locator('#article-title')).toHaveValue(title);

  // Accept: the import replaces the content and title.
  page.once('dialog', (dialog) => dialog.accept());
  await importViaButton(page, path.join(FIXTURES_DIR, 'sample.docx'));
  await expect(page.locator('#article-title')).toHaveValue('Printer Supplies Handbook', { timeout: 10_000 });
  await expect(page.locator('#article-editor')).not.toContainText('Original hand-typed content.');

  // Publishing the import adds a second version.
  await page.click('#btn-publish');
  await ready(page);
  await page.locator('a', { hasText: 'History' }).click();
  await ready(page);
  await expect(page.locator('.kb-history-list li')).toHaveCount(2);

  // The old (pre-import) version still opens and restores correctly.
  await page.locator('a', { hasText: 'Version 1' }).click();
  await ready(page);
  await expect(page.locator('.kb-article-body')).toContainText('Original hand-typed content.');
  await page.locator('button', { hasText: 'Restore this version' }).click();
  await ready(page);
  await expect(page.locator('.kb-article-body')).toContainText('Original hand-typed content.');
});

// SPEC gate 1.51: an unreadable file, or one over 50 MB, shows the exact
// message and leaves the editor untouched.
test('each bad fixture and an oversize file show the exact message and leave the editor unchanged', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);

  await page.fill('#article-title', 'Untouched title');
  const editor = page.locator('#article-editor .ProseMirror');
  await editor.click();
  await page.keyboard.type('Untouched content.');

  // The editor already has content, so every import attempt below also
  // asks to replace it; accept every time to reach the actual import.
  page.on('dialog', (dialog) => dialog.accept());

  for (const fixture of ['bad/oldword.doc', 'bad/protected.docx', 'bad/file.pdf', 'bad/truncated.docx']) {
    await importViaButton(page, path.join(FIXTURES_DIR, fixture));
    await expect(page.locator('#editor-message')).toContainText("Comp HQ can't open this file", { timeout: 5000 });
    await expect(page.locator('#article-editor')).toContainText('Untouched content.');
    await expect(page.locator('#article-title')).toHaveValue('Untouched title');
  }

  // Playwright's setFiles refuses an in-memory buffer over 50MB, so the
  // oversize file has to be a real one on disk instead.
  const hugePath = path.join(os.tmpdir(), `comphq-huge-${Date.now()}.docx`);
  fs.writeFileSync(hugePath, Buffer.alloc(50 * 1024 * 1024 + 1));
  try {
    await importViaButton(page, hugePath);
    await expect(page.locator('#editor-message')).toContainText('50 MB', { timeout: 10_000 });
    await expect(page.locator('#article-editor')).toContainText('Untouched content.');
  } finally {
    fs.unlinkSync(hugePath);
  }
});

// SPEC gate 1.52: a 20-page, 10-picture document is in the editor within
// 10 seconds of choosing it.
test('a 20-page document with 10 pictures is in the editor within 10 seconds', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Printers');
  await createCategory(page, server.baseURL, category);
  await goToNewArticleIn(page, server.baseURL, category);

  const start = Date.now();
  await importViaButton(page, path.join(FIXTURES_DIR, 'big-20pages.docx'));
  await expect(page.locator('#article-title')).toHaveValue('Twenty Section Reference Document', { timeout: 10_000 });
  const elapsed = Date.now() - start;
  expect(elapsed, `import took ${elapsed}ms, want <= 10000ms`).toBeLessThanOrEqual(10_000);
});

// Optional: the owner's own real documents, per PLAN.md P1-33 — imports
// without error, produces at least one block, and every picture is local.
const samplesDir = path.join(__dirname, '..', '..', 'samples', 'word');
const sampleWordFiles = fs.existsSync(samplesDir)
  ? fs.readdirSync(samplesDir).filter((f) => f.toLowerCase().endsWith('.docx'))
  : [];

for (const filename of sampleWordFiles) {
  test(`importing the owner's sample ${filename} succeeds with only local images`, async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const category = uniqueName('Printers');
    await createCategory(page, server.baseURL, category);
    await goToNewArticleIn(page, server.baseURL, category);

    await importViaButton(page, path.join(samplesDir, filename));
    await expect(page.locator('#article-title')).not.toHaveValue('', { timeout: 15_000 });

    await expect(page.locator('#editor-message')).not.toContainText("Comp HQ can't open this file");
    await expect(page.locator('#editor-message')).not.toContainText('50 MB');

    const blockCount = await page.locator('#article-editor > *').count();
    expect(blockCount).toBeGreaterThan(0);

    const imgs = page.locator('#article-editor img');
    const count = await imgs.count();
    for (let i = 0; i < count; i++) {
      const src = await imgs.nth(i).getAttribute('src');
      expect(src).toMatch(/^\/images\//);
    }
  });
}
