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

async function goToNewArticle(page: Page, baseURL: string) {
  await page.goto(baseURL + '/kb/new');
  await page.waitForSelector('body[data-editor-ready]');
}

// Creates a category, opens a new article against it with the given title,
// and returns once the editor is ready to type into.
async function startArticle(page: Page, baseURL: string, title: string) {
  const category = uniqueName('Printers');
  await createCategory(page, baseURL, category);
  await goToNewArticle(page, baseURL);
  await page.selectOption('#article-category', { label: category });
  await page.fill('#article-title', title);
  await page.click('#article-editor');
}

async function publish(page: Page) {
  await page.click('#btn-publish');
  await ready(page);
}

test.describe('editor toolbar (SPEC gate 1.15)', () => {
  test('large heading', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const title = uniqueName('Large heading article');
    await startArticle(page, server.baseURL, title);
    await page.click('#btn-h2');
    await page.keyboard.type('A large heading');
    await publish(page);
    await expect(page.locator('.kb-article-body h2')).toHaveText('A large heading');
  });

  test('small heading', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const title = uniqueName('Small heading article');
    await startArticle(page, server.baseURL, title);
    await page.click('#btn-h3');
    await page.keyboard.type('A small heading');
    await publish(page);
    await expect(page.locator('.kb-article-body h3')).toHaveText('A small heading');
  });

  test('bold and italic', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const title = uniqueName('Bold italic article');
    await startArticle(page, server.baseURL, title);
    await page.keyboard.type('emphasised');
    await page.keyboard.press('Control+a');
    await page.click('#btn-bold');
    await page.click('#btn-italic');
    await publish(page);
    await expect(page.locator('.kb-article-body strong em, .kb-article-body em strong')).toHaveText('emphasised');
  });

  test('bullet list', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const title = uniqueName('Bullet list article');
    await startArticle(page, server.baseURL, title);
    await page.click('#btn-bullet');
    await page.keyboard.type('one');
    await page.keyboard.press('Enter');
    await page.keyboard.type('two');
    await publish(page);
    await expect(page.locator('.kb-article-body ul li')).toHaveCount(2);
  });

  test('numbered list', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const title = uniqueName('Numbered list article');
    await startArticle(page, server.baseURL, title);
    await page.click('#btn-ordered');
    await page.keyboard.type('one');
    await page.keyboard.press('Enter');
    await page.keyboard.type('two');
    await publish(page);
    await expect(page.locator('.kb-article-body ol li')).toHaveCount(2);
  });

  test('link', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const title = uniqueName('Link article');
    await startArticle(page, server.baseURL, title);
    await page.keyboard.type('our supplier');
    await page.keyboard.press('Control+a');
    page.once('dialog', (dialog) => dialog.accept('https://example.test/supplier'));
    await page.click('#btn-link');
    await publish(page);
    await expect(page.locator('.kb-article-body a')).toHaveAttribute('href', 'https://example.test/supplier');
    await expect(page.locator('.kb-article-body a')).toHaveAttribute('rel', 'noopener');
  });

  test('table with add/remove row and column', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    const title = uniqueName('Table article');
    await startArticle(page, server.baseURL, title);
    await page.click('#btn-table');
    await expect(page.locator('#article-editor table')).toHaveCount(1);

    await page.click('#btn-add-row');
    await expect(page.locator('#article-editor table tr')).toHaveCount(3);
    await page.click('#btn-remove-row');
    await expect(page.locator('#article-editor table tr')).toHaveCount(2);

    await page.click('#btn-add-col');
    await expect(page.locator('#article-editor table tr').first().locator('> *')).toHaveCount(3);
    await page.click('#btn-remove-col');
    await expect(page.locator('#article-editor table tr').first().locator('> *')).toHaveCount(2);

    await publish(page);
    await expect(page.locator('.kb-article-body table tr')).toHaveCount(2);
    await expect(page.locator('.kb-article-body table tr').first().locator('> *')).toHaveCount(2);
  });

  test('the table row/column buttons are disabled outside a table', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await startArticle(page, server.baseURL, uniqueName('Not a table article'));
    await page.keyboard.type('no table here');

    for (const id of ['#btn-add-row', '#btn-remove-row', '#btn-add-col', '#btn-remove-col']) {
      await expect(page.locator(id)).toBeDisabled();
    }

    await page.click('#btn-table');
    for (const id of ['#btn-add-row', '#btn-remove-row', '#btn-add-col', '#btn-remove-col']) {
      await expect(page.locator(id)).toBeEnabled();
    }
  });

  test('Import from Word is disabled with a tooltip', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await goToNewArticle(page, server.baseURL);
    const button = page.locator('#btn-import-word');
    await expect(button).toBeDisabled();
    await expect(button).toHaveAttribute('title', /.+/);
  });
});

// SPEC gate 1.20: a beforeunload prompt when dirty, and Cancel on a dirty
// editor asks "Leave without saving?" first.
test.describe('leaving the editor (SPEC gate 1.20)', () => {
  test('Cancel on a dirty editor asks first, and dismissing it stays put', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await startArticle(page, server.baseURL, uniqueName('Dirty cancel article'));
    await page.keyboard.type('unsaved work');

    let message = '';
    page.once('dialog', async (dialog) => {
      message = dialog.message();
      await dialog.dismiss();
    });
    await page.click('#btn-cancel');
    expect(message).toBe('Leave without saving?');
    // Dismissing the confirm keeps the editor open with the typed text.
    await expect(page.locator('#article-editor')).toContainText('unsaved work');
  });

  test('Cancel on a dirty editor navigates away once confirmed', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await startArticle(page, server.baseURL, uniqueName('Dirty cancel confirm article'));
    await page.keyboard.type('unsaved work');

    page.once('dialog', (dialog) => dialog.accept());
    await page.click('#btn-cancel');
    await page.waitForURL('**/kb');
  });

  test('Cancel on a clean editor leaves without asking', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    // A truly untouched page: create a category so the form renders, but
    // don't fill anything in it (startArticle's own selectOption/fill
    // would themselves count as unsaved changes).
    await createCategory(page, server.baseURL, uniqueName('Printers'));
    await goToNewArticle(page, server.baseURL);

    let dialogSeen = false;
    page.once('dialog', async (dialog) => {
      dialogSeen = true;
      await dialog.dismiss();
    });
    await page.click('#btn-cancel');
    await page.waitForURL('**/kb');
    expect(dialogSeen).toBe(false);
  });

  test('navigating away from a dirty editor triggers the browser prompt', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/kb');
    await startArticle(page, server.baseURL, uniqueName('Beforeunload article'));
    await page.keyboard.type('unsaved work');

    let sawBeforeUnload = false;
    page.on('dialog', async (dialog) => {
      if (dialog.type() === 'beforeunload') sawBeforeUnload = true;
      await dialog.dismiss().catch(() => {});
    });
    await page.locator('a[href="/kb"]').first().click();
    await page.waitForTimeout(250);
    expect(sawBeforeUnload).toBe(true);
  });
});

test('publishing an article works with keyboard only', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  const category = uniqueName('Keyboard printers');
  await createCategory(page, server.baseURL, category);
  await goToNewArticle(page, server.baseURL);

  const title = uniqueName('Keyboard article');
  await page.locator('#article-category').focus();
  await page.locator('#article-category').selectOption({ label: category });
  await page.locator('#article-title').focus();
  await page.keyboard.type(title);
  await page.locator('#article-editor .ProseMirror').focus();
  await page.keyboard.type('Typed entirely from the keyboard.');
  await page.locator('#btn-publish').focus();
  await page.keyboard.press('Enter');
  await ready(page);

  await expect(page.locator('h1')).toHaveText(title);
});

test('no CSP violations while exercising every toolbar feature', async ({ page, server }) => {
  const consoleErrors: string[] = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });
  const pageErrors: string[] = [];
  page.on('pageerror', (err) => pageErrors.push(String(err)));
  await page.addInitScript(() => {
    (window as unknown as { __cspViolations: unknown[] }).__cspViolations = [];
    document.addEventListener('securitypolicyviolation', (e) => {
      (window as unknown as { __cspViolations: unknown[] }).__cspViolations.push({
        directive: e.violatedDirective,
        blockedURI: e.blockedURI,
      });
    });
  });

  await signInAsNewPerson(page, server.baseURL, '/kb');
  await startArticle(page, server.baseURL, uniqueName('CSP article'));
  await page.keyboard.type('hello world');
  await page.keyboard.press('Control+a');

  page.on('dialog', (dialog) => dialog.dismiss());
  for (const id of [
    'btn-bold', 'btn-italic', 'btn-h2', 'btn-h3',
    'btn-bullet', 'btn-ordered', 'btn-table', 'btn-add-row', 'btn-add-col',
    'btn-remove-row', 'btn-remove-col', 'btn-link',
  ]) {
    await page.click('#' + id);
  }

  const violations = await page.evaluate(() => (window as unknown as { __cspViolations: unknown[] }).__cspViolations);
  expect(violations, JSON.stringify(violations)).toEqual([]);
  expect(consoleErrors, JSON.stringify(consoleErrors)).toEqual([]);
  expect(pageErrors, JSON.stringify(pageErrors)).toEqual([]);
});
