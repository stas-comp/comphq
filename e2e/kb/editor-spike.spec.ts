import { expect, test } from '../helpers/fixtures';
import { ready } from '../helpers/ready';

// P1-09: proves the vendored TipTap bundle works, under the app's real
// CSP, against the GET /__test/editor harness (COMPHQ_TEST_MODE only).

async function gotoHarness(page: import('@playwright/test').Page, baseURL: string) {
  await page.goto(baseURL + '/__test/editor');
  await ready(page);
  await page.click('#editor');
}

test.describe('editor toolbar', () => {
  test('heading 2', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.click('#btn-h2');
    await expect(page.locator('#editor h2')).toHaveCount(1);
  });

  test('heading 3', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.click('#btn-h3');
    await expect(page.locator('#editor h3')).toHaveCount(1);
  });

  test('bold', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.keyboard.type('hello');
    await page.keyboard.press('Control+a');
    await page.click('#btn-bold');
    await expect(page.locator('#editor strong')).toHaveCount(1);
  });

  test('italic', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.keyboard.type('hello');
    await page.keyboard.press('Control+a');
    await page.click('#btn-italic');
    await expect(page.locator('#editor em')).toHaveCount(1);
  });

  test('bullet list', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.click('#btn-bullet');
    await expect(page.locator('#editor ul')).toHaveCount(1);
  });

  test('numbered list', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.click('#btn-ordered');
    await expect(page.locator('#editor ol')).toHaveCount(1);
  });

  test('link', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.keyboard.type('hello');
    await page.keyboard.press('Control+a');
    await page.click('#btn-link');
    await expect(page.locator('#editor a[href^="mailto:"]')).toHaveCount(1);
  });

  test('image', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.click('#btn-image');
    await expect(page.locator('#editor img')).toHaveCount(1);
  });

  test('table with add/remove row and column', async ({ page, server }) => {
    await gotoHarness(page, server.baseURL);
    await page.click('#btn-table');
    await expect(page.locator('#editor table')).toHaveCount(1);

    const rows = page.locator('#editor table tr');
    const firstRowCells = page.locator('#editor table tr:first-child > *');
    await expect(rows).toHaveCount(2);
    await expect(firstRowCells).toHaveCount(2);

    await page.click('#btn-add-row');
    await expect(rows).toHaveCount(3);
    await page.click('#btn-remove-row');
    await expect(rows).toHaveCount(2);

    await page.click('#btn-add-col');
    await expect(firstRowCells).toHaveCount(3);
    await page.click('#btn-remove-col');
    await expect(firstRowCells).toHaveCount(2);
  });
});

test('a FileHandler paste callback fires for a synthetic image paste', async ({ page, server }) => {
  await gotoHarness(page, server.baseURL);

  await page.evaluate(() => {
    const bytes = Uint8Array.from(atob(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII='
    ), (c) => c.charCodeAt(0));
    const file = new File([bytes], 'pasted.png', { type: 'image/png' });
    const dt = new DataTransfer();
    dt.items.add(file);
    const editorEl = document.querySelector('#editor .ProseMirror') as HTMLElement;
    const event = new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: dt });
    editorEl.dispatchEvent(event);
  });

  const log = page.locator('#paste-log');
  await expect(log).toHaveAttribute('data-pasted', 'true', { timeout: 3000 });
  await expect(log).toHaveText('pasted.png');
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

  await gotoHarness(page, server.baseURL);
  await page.keyboard.type('hello world');
  await page.keyboard.press('Control+a');

  for (const id of [
    'btn-bold', 'btn-italic', 'btn-link', 'btn-h2', 'btn-h3',
    'btn-bullet', 'btn-ordered', 'btn-table', 'btn-add-row', 'btn-add-col',
    'btn-remove-row', 'btn-remove-col', 'btn-image',
  ]) {
    await page.click('#' + id);
  }

  const violations = await page.evaluate(
    () => (window as unknown as { __cspViolations: unknown[] }).__cspViolations,
  );
  expect(violations, JSON.stringify(violations)).toEqual([]);
  expect(consoleErrors, JSON.stringify(consoleErrors)).toEqual([]);
  expect(pageErrors, JSON.stringify(pageErrors)).toEqual([]);
});
