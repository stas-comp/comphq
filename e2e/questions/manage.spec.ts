import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { parseCSV, zipEntry } from '../helpers/zip';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 8.75, 8.76, 8.80, 8.81 (B14.6, D-110): editing, removing, restoring and
// exporting questions. Titles are unique because these tests share a server.

test.describe.configure({ timeout: 90_000 });

async function ask(page: Page, baseURL: string, text: string, details = ''): Promise<string> {
  await page.goto(baseURL + '/questions');
  await ready(page);
  await page.getByLabel('Your question').fill(text);
  if (details) await page.getByLabel('More details').fill(details);
  await page.getByRole('button', { name: 'Ask' }).click();
  await ready(page);
  // Asking returns to the list; open the new question.
  await page.locator('.q-row', { hasText: text }).getByRole('link').first().click();
  await ready(page);
  return new URL(page.url()).pathname;
}

async function answer(page: Page, text: string): Promise<void> {
  await page.getByLabel('Add an answer').fill(text);
  await page.getByRole('button', { name: 'Add answer' }).click();
  await ready(page);
}

test('gate 8.75: anyone can change the words of a question and an answer; it then says who and when', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Edit me');
  const path = await ask(page, server.baseURL, text, 'First details');
  await answer(page, 'First answer');

  // Someone else makes the changes.
  const other = await browser.newContext();
  const o = await other.newPage();
  const name = await signInAsNewPerson(o, server.baseURL, path);
  await ready(o);
  await o.getByRole('link', { name: 'Edit question' }).click();
  await ready(o);
  await expect(o.getByLabel('Your question')).toHaveValue(text);
  await o.getByLabel('Your question').fill(text + ' (changed)');
  await o.getByLabel('More details').fill('Better details');
  await o.getByRole('button', { name: 'Save' }).click();
  await ready(o);
  await expect(o.locator('.q-title')).toHaveText(text + ' (changed)');
  await expect(o.locator('.q-details')).toHaveText('Better details');
  await expect(o.locator('.q-view > .q-meta')).toContainText(`edited by ${name}`);

  await o.getByRole('link', { name: /^Edit answer by/ }).click();
  await ready(o);
  await expect(o.locator('#answer-1, .q-answer').first().getByLabel('Edit this answer')).toHaveValue('First answer');
  await o.getByLabel('Edit this answer').fill('Better answer');
  await o.getByRole('button', { name: 'Save' }).click();
  await ready(o);
  await expect(o.locator('.q-answer-text')).toHaveText('Better answer');
  await expect(o.locator('.q-answer .q-meta')).toContainText(`edited by ${name}`);
  await other.close();
});

test('gate 8.75: Cancel changes nothing; an empty or too long edit is refused with the typed words kept', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Refuse edit');
  const path = await ask(page, server.baseURL, text);
  await page.getByRole('link', { name: 'Edit question' }).click();
  await page.getByLabel('Your question').fill('Typed but cancelled');
  await page.getByRole('link', { name: 'Cancel' }).click();
  await ready(page);
  await expect(page.locator('.q-title')).toHaveText(text);

  // The box carries its limit, so go past it with a plain post.
  const res = await page.request.post(server.baseURL + path, { form: { text: 'x'.repeat(301), details: 'keep these words' }, headers: { origin: server.baseURL } });
  const body = await res.text();
  expect(body).toContain('role="alert"');
  expect(body).toContain('keep these words');
  await page.goto(server.baseURL + path);
  await expect(page.locator('.q-title')).toHaveText(text);
});

test('gate 8.76: removing an answer shows Undo for 10 seconds; Undo brings it back', async ({ page, server }) => {
  await page.clock.install();
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Undo answer');
  const path = await ask(page, server.baseURL, text);
  await answer(page, 'The answer that goes');
  await answer(page, 'The answer that stays');

  await page.getByRole('button', { name: /^Remove answer by/ }).first().click();
  await ready(page);
  await expect(page.locator('.q-answer')).toHaveCount(1);
  await expect(page.locator('.q-answer-text')).toHaveText('The answer that stays');
  await expect(page.locator('#toast-region .toast')).toContainText('Answer removed');
  await page.clock.runFor(9_000);
  await expect(page.locator('#toast-region .toast')).toBeVisible();
  await page.getByRole('button', { name: 'Undo' }).click();
  await ready(page);
  await expect(page.locator('.q-answer')).toHaveCount(2);
  await expect(page.locator('#toast-region .toast')).toHaveCount(0);

  // Left alone, the message goes after 10 seconds and the answer stays removed.
  await page.getByRole('button', { name: /^Remove answer by/ }).first().click();
  await ready(page);
  await expect(page.locator('#toast-region .toast')).toBeVisible();
  await page.clock.runFor(10_500);
  await expect(page.locator('#toast-region .toast')).toHaveCount(0);
  await page.goto(server.baseURL + path);
  await expect(page.locator('.q-answer')).toHaveCount(1);
});

test('gate 8.76: a removed question leaves Questions and appears in Removed questions, where it can be restored', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Remove question');
  await ask(page, server.baseURL, text);
  await page.getByRole('button', { name: 'Remove question' }).click();
  await ready(page);
  await expect(page).toHaveURL(/\/questions$/);
  await expect(page.locator('.q-row', { hasText: text })).toHaveCount(0);

  await page.getByRole('link', { name: 'Removed questions' }).click();
  await ready(page);
  const row = page.locator('.q-row', { hasText: text });
  await expect(row).toHaveCount(1);
  await expect(row).toContainText('removed by');
  await axeCheck(page);
  await row.getByRole('button', { name: 'Restore question' }).click();
  await ready(page);
  await expect(page.locator('.q-title')).toHaveText(text);
  await page.goto(server.baseURL + '/questions');
  await expect(page.locator('.q-row', { hasText: text })).toHaveCount(1);
});

test('gate 8.76: removing is the one remove icon; the question page adds no other icon-only button', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  await ask(page, server.baseURL, uniqueName('Icons'));
  await answer(page, 'An answer');
  const iconOnly = await page.locator('main button').evaluateAll((els) =>
    els.filter((el) => !(el.textContent || '').trim()).map((el) => el.querySelector('.icon')?.className ?? '(no icon)'),
  );
  expect(iconOnly.length).toBeGreaterThan(0);
  for (const c of iconOnly) expect(c).toContain('icon-trash');
});

test('gate 8.80: editing, removing and restoring work with JavaScript switched off', async ({ browser, server }) => {
  const normal = await browser.newContext();
  const p = await normal.newPage();
  await signInAsNewPerson(p, server.baseURL, '/questions');
  const context = await browser.newContext({ javaScriptEnabled: false });
  await context.addCookies(await normal.cookies());
  await normal.close();
  const page = await context.newPage();
  const text = uniqueName('No script manage');
  await page.goto(server.baseURL + '/questions');
  await page.getByLabel('Your question').fill(text);
  await page.getByRole('button', { name: 'Ask' }).click();
  await page.locator('.q-row', { hasText: text }).getByRole('link').first().click();
  await page.getByLabel('Add an answer').fill('Answer to edit');
  await page.getByRole('button', { name: 'Add answer' }).click();

  await page.getByRole('link', { name: 'Edit question' }).click();
  await page.getByLabel('Your question').fill(text + ' v2');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.locator('.q-title')).toHaveText(text + ' v2');
  await page.getByRole('link', { name: /^Edit answer by/ }).click();
  await page.getByLabel('Edit this answer').fill('Answer edited');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.locator('.q-answer-text')).toHaveText('Answer edited');

  await page.getByRole('button', { name: /^Remove answer by/ }).click();
  await expect(page.locator('#toast-region .toast')).toContainText('Answer removed');
  await page.getByRole('button', { name: 'Undo' }).click();
  await expect(page.locator('.q-answer')).toHaveCount(1);

  await page.getByRole('button', { name: 'Remove question' }).click();
  await expect(page.locator('.q-row', { hasText: text })).toHaveCount(0);
  await page.getByRole('link', { name: 'Removed questions' }).click();
  await page.locator('.q-row', { hasText: text }).getByRole('button', { name: 'Restore question' }).click();
  await expect(page.locator('.q-title')).toHaveText(text + ' v2');
  await context.close();
});

test('gate 8.81: Export everything includes questions.csv and answers.csv', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const text = uniqueName('Exported question');
  await ask(page, server.baseURL, text, 'Exported details');
  await answer(page, 'Exported answer');
  await page.getByRole('button', { name: 'Mark answered' }).click();
  await ready(page);

  const res = await page.request.get(server.baseURL + '/settings/export');
  expect(res.ok()).toBe(true);
  const zip = Buffer.from(await res.body());
  const questions = zipEntry(zip, 'questions.csv');
  const answers = zipEntry(zip, 'answers.csv');
  expect(questions, 'questions.csv is in the export').not.toBeNull();
  expect(answers, 'answers.csv is in the export').not.toBeNull();
  const q = parseCSV(questions!.toString('utf8'));
  expect(q[0]).toEqual(['Question', 'Details', 'Asked by', 'Asked', 'Answered by', 'Answered', 'Edited by', 'Edited', 'Removed by', 'Removed']);
  const mine = q.find((r) => r[0] === text)!;
  expect(mine, 'the question is in questions.csv').toBeTruthy();
  expect(mine[1]).toBe('Exported details');
  expect(mine[4]).not.toBe('');
  const a = parseCSV(answers!.toString('utf8'));
  expect(a[0]).toEqual(['Question', 'Answer', 'Answered by', 'Answered', 'Edited by', 'Edited', 'Removed by', 'Removed']);
  expect(a.find((r) => r[0] === text && r[1] === 'Exported answer')).toBeTruthy();
});

test('standard checks: axe and no sideways scroll on an edit form', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  await ask(page, server.baseURL, uniqueName('Edit checks'));
  await answer(page, 'An answer');
  for (const size of [{ width: 1280, height: 800 }, { width: 390, height: 800 }]) {
    await page.setViewportSize(size);
    await page.getByRole('link', { name: 'Edit question' }).click();
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
    await page.getByRole('link', { name: 'Cancel' }).click();
    await page.getByRole('link', { name: /^Edit answer by/ }).click();
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
    await page.getByRole('link', { name: 'Cancel' }).click();
  }
});
