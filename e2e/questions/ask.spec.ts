import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { newDataDir, startServerAt } from '../helpers/server-at';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 8.70-8.74, 8.80 (B14.6, D-110): the Questions section. Titles are unique
// because these tests share a server with everything else.

test.describe.configure({ timeout: 90_000 });

const row = (page: Page, text: string) => page.locator('.q-row', { hasText: text });
const waiting = (page: Page) => page.locator('section[aria-labelledby="q-waiting-heading"]');
const answered = (page: Page) => page.locator('section[aria-labelledby="q-answered-heading"]');

async function ask(page: Page, baseURL: string, text: string, details = ''): Promise<void> {
  await page.goto(baseURL + '/questions');
  await ready(page);
  await page.getByLabel('Your question').fill(text);
  if (details) await page.getByLabel('More details').fill(details);
  await page.getByRole('button', { name: 'Ask' }).click();
  await ready(page);
}

async function questionPath(page: Page, text: string): Promise<string> {
  return (await row(page, text).locator('a').first().getAttribute('href'))!;
}

test('gate 8.70: Questions is in the sidebar between Calendar and Settings, with its own icon', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  await ready(page);
  const labels = await page.locator('.sidebar nav a').allTextContents();
  const clean = labels.map((l) => l.trim());
  expect(clean.indexOf('Questions')).toBe(clean.indexOf('Calendar') + 1);
  expect(clean.indexOf('Questions')).toBeLessThan(clean.indexOf('Settings'));
  const link = page.locator('.sidebar nav a[href="/questions"]');
  await expect(link).toHaveAttribute('aria-current', 'page');
  const icon = link.locator('.icon.icon-questions');
  await expect(icon).toBeVisible();
  // The icon takes the colour of the link (currentColor), so both brands colour it with no file of their own.
  const linkColour = await link.evaluate((el) => getComputedStyle(el).color);
  await expect(icon).toHaveCSS('background-color', linkColour);
  expect(await icon.evaluate((el) => getComputedStyle(el).maskImage || getComputedStyle(el).webkitMaskImage)).toContain('questions.svg');
});

test('@fresh gate 8.70: MC HQ has the same entry and icon in its own colours', async ({ page }) => {
  const dataDir = newDataDir('questions-mc');
  const server = await startServerAt(dataDir, '2026-11-21', { COMPHQ_BRAND: 'mc' });
  try {
    await signInAsNewPerson(page, server.baseURL, '/questions');
    await ready(page);
    await expect(page.locator('html')).toHaveAttribute('data-brand', 'mc');
    const link = page.locator('.sidebar nav a[href="/questions"]');
    await expect(link).toBeVisible();
    await expect(link.locator('.icon-questions')).toBeVisible();
    await expect(page.locator('.sidebar nav a').nth(4)).toHaveText(/Questions/);
    await axeCheck(page);
  } finally {
    await server.stop();
  }
});

test('gates 8.71, 8.72: ask with details; it is Waiting, with who asked and when and 0 answers, newest first', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/questions');
  const first = uniqueName('First question');
  const second = uniqueName('Second question');
  await ask(page, server.baseURL, first);
  await ask(page, server.baseURL, second, 'Line one\nLine two\n\nLine four');
  const rows = await waiting(page).locator('.q-row-title').allTextContents();
  expect(rows.indexOf(second)).toBeGreaterThanOrEqual(0);
  expect(rows.indexOf(second)).toBeLessThan(rows.indexOf(first)); // newest first
  const meta = (await row(page, second).locator('.q-row-meta').innerText()).replace(/\s+/g, ' ');
  expect(meta).toContain(`Asked by ${me}`);
  expect(meta).toMatch(/Asked by .+, \w{3} \d{1,2} \w{3}/); // a date, as "Sat 21 Nov"
  expect(meta).toContain('0 answers');
  // It is not in Answered.
  await expect(answered(page).locator('.q-row', { hasText: second })).toHaveCount(0);
  // The details keep their line breaks on the question's page.
  await row(page, second).locator('a').click();
  await ready(page);
  await expect(page.locator('.q-details')).toHaveText('Line one\nLine two\n\nLine four');
  await expect(page.locator('.q-details')).toHaveCSS('white-space', 'pre-line');
});

test('gate 8.71: refusals keep what was typed, in plain words; the boxes carry their limits', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  await ready(page);
  await expect(page.getByLabel('Your question')).toHaveAttribute('maxlength', '300');
  await expect(page.getByLabel('More details')).toHaveAttribute('maxlength', '2000');
  const post = (text: string, details: string) =>
    page.request.post(server.baseURL + '/questions', {
      data: new URLSearchParams([['text', text], ['details', details]]).toString(),
      headers: { origin: server.baseURL, 'content-type': 'application/x-www-form-urlencoded' },
    });
  let res = await post('x'.repeat(301), 'kept details');
  expect(await res.text()).toContain('A question can be up to 300 characters.');
  expect(await res.text()).toContain('kept details');
  res = await post('   ', 'only details');
  expect(await res.text()).toContain('Type your question first.');
  res = await post('Fine', 'd'.repeat(2001));
  expect(await res.text()).toContain('The details can be up to 2,000 characters.');
  // And from the page itself: whitespace passes the browser's own required check, but not ours.
  await page.getByLabel('Your question').fill('   ');
  await page.getByLabel('More details').fill('Keep these words');
  await page.getByRole('button', { name: 'Ask' }).click();
  await expect(page.locator('.message[role="alert"]')).toHaveText('Type your question first.');
  await expect(page.getByLabel('More details')).toHaveValue('Keep these words');
});

test('gate 8.73: a question page shows its answers oldest first, each with who and when; anyone answers, any number of times', async ({ page, server, browser }) => {
  const asker = await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Who orders the stamps');
  await ask(page, server.baseURL, text);
  const path = await questionPath(page, text);

  const other = await browser.newContext();
  const op = await other.newPage();
  const helper = await signInAsNewPerson(op, server.baseURL, path);
  await ready(op);
  await op.getByLabel('Add an answer').fill('Ask the office manager');
  await op.getByRole('button', { name: 'Add answer' }).click();
  await ready(op);

  await page.goto(server.baseURL + path);
  await ready(page);
  await page.getByLabel('Add an answer').fill('Thanks, and one more thing:\nsecond line');
  await page.getByRole('button', { name: 'Add answer' }).click();
  await ready(page);
  await page.getByLabel('Add an answer').fill('A third one from the asker');
  await page.getByRole('button', { name: 'Add answer' }).click();
  await ready(page);

  const answers = page.locator('.q-answer');
  await expect(answers).toHaveCount(3);
  await expect(answers.nth(0)).toContainText('Ask the office manager');
  await expect(answers.nth(0).locator('.q-meta')).toContainText(helper);
  await expect(answers.nth(1)).toContainText('Thanks, and one more thing:');
  await expect(answers.nth(1).locator('.q-answer-text')).toHaveCSS('white-space', 'pre-line');
  await expect(answers.nth(2).locator('.q-meta')).toContainText(asker);
  await expect(answers.nth(2).locator('.q-meta')).toContainText(/\w{3} \d{1,2} \w{3}/);
  // The list counts them, and adding answers didn't mark it answered.
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await expect(waiting(page).locator('.q-row', { hasText: text }).locator('.q-row-meta')).toContainText('3 answers');
  await other.close();
});

test('gate 8.73: an empty or too long answer is refused with the typed words kept', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Answer limits');
  await ask(page, server.baseURL, text);
  await page.goto(server.baseURL + (await questionPath(page, text)));
  await ready(page);
  await expect(page.getByLabel('Add an answer')).toHaveAttribute('maxlength', '2000');
  await page.getByLabel('Add an answer').fill('   ');
  await page.getByRole('button', { name: 'Add answer' }).click();
  await expect(page.locator('.message[role="alert"]')).toHaveText('Type your answer first.');
  const id = /\/questions\/(\d+)/.exec(page.url())![1];
  const res = await page.request.post(`${server.baseURL}/questions/${id}/answers`, {
    data: new URLSearchParams([['text', 'z'.repeat(2001)]]).toString(),
    headers: { origin: server.baseURL, 'content-type': 'application/x-www-form-urlencoded' },
  });
  expect(await res.text()).toContain('An answer can be up to 2,000 characters.');
  await expect(page.locator('.q-answer')).toHaveCount(0);
});

test('gate 8.74: Mark answered moves it to Answered with who and when; Not answered yet puts it back', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Is this settled');
  await ask(page, server.baseURL, text);
  const path = await questionPath(page, text);

  const other = await browser.newContext();
  const op = await other.newPage();
  const marker = await signInAsNewPerson(op, server.baseURL, path);
  await ready(op);
  await op.getByRole('button', { name: 'Mark answered' }).click();
  await ready(op);
  await expect(op.locator('.q-answered-line')).toHaveText(new RegExp(`^Answered · marked by ${marker}, \\w{3} \\d{1,2} \\w{3}$`));
  await other.close();

  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await expect(answered(page).locator('.q-row', { hasText: text })).toHaveCount(1);
  await expect(waiting(page).locator('.q-row', { hasText: text })).toHaveCount(0);
  await expect(answered(page).locator('.q-row', { hasText: text }).locator('.q-row-meta')).toContainText(`marked by ${marker}`);

  await page.goto(server.baseURL + path);
  await ready(page);
  await page.getByRole('button', { name: 'Not answered yet' }).click();
  await ready(page);
  await expect(page.getByRole('button', { name: 'Mark answered' })).toBeVisible();
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await expect(waiting(page).locator('.q-row', { hasText: text })).toHaveCount(1);
  await expect(answered(page).locator('.q-row', { hasText: text })).toHaveCount(0);
});

test('plain text: what looks like markup is shown as typed, never made into markup', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Is <b>bold</b> ok') + ' & <i>x</i>';
  await ask(page, server.baseURL, text, '<script>window.__pwned = 1</script>');
  await page.goto(server.baseURL + (await questionPath(page, text)));
  await ready(page);
  await expect(page.locator('.q-title')).toHaveText(text);
  await expect(page.locator('.q-title b, .q-details script')).toHaveCount(0);
  expect(await page.evaluate(() => (window as unknown as { __pwned?: number }).__pwned)).toBeUndefined();
  await expect(page.locator('.q-details')).toHaveText('<script>window.__pwned = 1</script>');
});

test('gate 8.80: every Questions page works with JavaScript switched off', async ({ browser, server }) => {
  const normal = await browser.newContext();
  const p = await normal.newPage();
  await signInAsNewPerson(p, server.baseURL, '/questions');
  const context = await browser.newContext({ javaScriptEnabled: false });
  await context.addCookies(await normal.cookies());
  await normal.close();
  const page = await context.newPage();
  const text = uniqueName('No script question');
  await page.goto(server.baseURL + '/questions');
  await page.getByLabel('Your question').fill(text);
  await page.getByLabel('More details').fill('Some details');
  await page.getByRole('button', { name: 'Ask' }).click();
  await expect(page.locator('.q-row', { hasText: text })).toBeVisible();
  await page.locator('.q-row', { hasText: text }).getByRole('link').click();
  await page.getByLabel('Add an answer').fill('An answer with no script');
  await page.getByRole('button', { name: 'Add answer' }).click();
  await expect(page.locator('.q-answer')).toHaveCount(1);
  await page.getByRole('button', { name: 'Mark answered' }).click();
  await expect(page.locator('.q-answered-line')).toContainText('Answered · marked by');
  await page.getByRole('button', { name: 'Not answered yet' }).click();
  await expect(page.getByRole('button', { name: 'Mark answered' })).toBeVisible();
  await context.close();
});

test('standard checks: axe and no sideways scroll on the list and a question, at both window sizes', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Axe question');
  await ask(page, server.baseURL, text, 'Details for the check');
  await axeCheck(page);
  await expectNoSideScroll(page);
  await page.goto(server.baseURL + (await questionPath(page, text)));
  await ready(page);
  await page.getByLabel('Add an answer').fill('An answer');
  await page.getByRole('button', { name: 'Add answer' }).click();
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});

test('@fresh gate 8.72: Answered shows the 50 most recent, then Show all', async ({ page }) => {
  test.setTimeout(240_000);
  const dataDir = newDataDir('questions-many');
  const server = await startServerAt(dataDir, '2026-11-21');
  try {
    await signInAsNewPerson(page, server.baseURL, '/questions');
    for (let i = 1; i <= 55; i++) {
      const res = await page.request.post(server.baseURL + '/questions', {
        data: new URLSearchParams([['text', `Question ${String(i).padStart(2, '0')}`]]).toString(),
        headers: { origin: server.baseURL, 'content-type': 'application/x-www-form-urlencoded' },
        maxRedirects: 0,
      });
      expect(res.status()).toBeLessThan(400);
      await page.request.post(`${server.baseURL}/questions/${i}/answered`, { headers: { origin: server.baseURL }, maxRedirects: 0 });
      if (i % 11 === 0) await page.waitForTimeout(1100); // the answered times are in whole seconds
    }
    await page.goto(server.baseURL + '/questions');
    await ready(page);
    await expect(answered(page).locator('.q-row')).toHaveCount(50);
    await expect(answered(page).locator('.q-more a')).toHaveText('Show all 55');
    await answered(page).locator('.q-more a').click();
    await ready(page);
    await expect(answered(page).locator('.q-row')).toHaveCount(55);
    await axeCheck(page);
  } finally {
    await server.stop();
  }
});
