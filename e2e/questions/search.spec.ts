import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

type Page = import('@playwright/test').Page;

// SPEC gate 8.78 (B14.6, D-110): searching questions, and the links between the
// three searches. Words are unique per test, because the server is shared.

test.describe.configure({ timeout: 90_000 });

const token = () => 'q' + Math.random().toString(36).slice(2, 9) + 'x';
const box = (page: Page) => page.locator('#search-box');
const rows = (page: Page) => page.locator('.search-panel .search-result');

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

// A waiting question matching by its text, one by its details, an answered one by
// an answer, and a removed one that must never be found.
async function seed(page: Page, baseURL: string, tok: string) {
  await signInAsNewPerson(page, baseURL, '/questions');
  const answeredPath = await ask(page, baseURL, `Who orders the ${tok} paper`);
  await answer(page, `The ${tok} order is done by Dana in the office`);
  await page.getByRole('button', { name: 'Mark answered' }).click();
  await ready(page);
  await ask(page, baseURL, 'Where does the kettle live', `It might be near the ${tok} shelf`);
  await ask(page, baseURL, `${tok} waiting for the plumber`);
  await ask(page, baseURL, `${tok} removed question`);
  await page.getByRole('button', { name: 'Remove question' }).click();
  await ready(page);
  return answeredPath;
}

test('gate 8.78: on Questions pages the top box reads Search questions and finds waiting before answered, never removed', async ({ page, server }) => {
  const tok = token();
  await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await expect(box(page)).toHaveAttribute('placeholder', 'Search questions');
  await box(page).fill(tok);
  await expect(rows(page)).toHaveCount(3);
  await expect(page.locator('.search-panel')).not.toContainText('removed question');
  // Waiting ones first (question text before details), then the answered one.
  await expect(rows(page).nth(0)).toContainText('waiting for the plumber');
  await expect(rows(page).nth(1)).toContainText('Where does the kettle live');
  await expect(rows(page).nth(2)).toContainText('Who orders the');
  await expect(rows(page).nth(2)).toContainText('Answered');
  // The words found are marked, with who asked, when and how many answers.
  await expect(rows(page).nth(0).locator('mark')).toHaveText(tok);
  await expect(rows(page).nth(2)).toContainText('Asked by');
  await expect(rows(page).nth(2)).toContainText('1 answer');
  // A passage shows where a word was found outside the question itself.
  await expect(rows(page).nth(1).locator('.search-snippet mark')).toHaveText(tok);
  await expect(rows(page).nth(2).locator('.search-snippet mark')).toHaveText(tok);
});

test('gate 8.78: every word must be there, and the last word may be half typed', async ({ page, server }) => {
  const tok = token();
  await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await box(page).fill(`${tok} plum`);
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page).first()).toContainText('waiting for the plumber');
  await box(page).fill(`${tok} plumbing`);
  await expect(page.locator('.search-panel-empty')).toContainText('No questions match');
  await expect(rows(page)).toHaveCount(0);
});

test('gate 8.78: a result opens the question; Enter shows the full results page', async ({ page, server }) => {
  const tok = token();
  const answeredPath = await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await box(page).fill(tok);
  await rows(page).nth(2).click();
  await expect(page).toHaveURL(new RegExp(answeredPath + '$'));

  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await box(page).fill(tok);
  await expect(rows(page)).toHaveCount(3);
  await box(page).press('Enter');
  await expect(page).toHaveURL(new RegExp(`/questions/search\\?q=${tok}`));
  await ready(page);
  await expect(page.locator('main .q-row')).toHaveCount(3);
  await expect(page.locator('main mark').first()).toHaveText(tok);
  await expect(page.locator('main')).not.toContainText('removed question');
  await axeCheck(page);
  await expectNoSideScroll(page);
});

test('gate 8.78: question results end with Search articles instead and Search jobs instead', async ({ page, server }) => {
  const tok = token();
  await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  await box(page).fill(tok);
  await expect(rows(page)).toHaveCount(3);
  const links = page.locator('.search-panel .search-instead-link');
  await expect(links).toHaveText(['Search articles instead', 'Search jobs instead']);
  await expect(links.nth(0)).toHaveAttribute('href', `/kb/search?q=${tok}`);
  await expect(links.nth(1)).toHaveAttribute('href', `/tasks/search?q=${tok}`);

  await box(page).press('Enter');
  await ready(page);
  await page.getByRole('link', { name: 'Search jobs instead' }).click();
  await expect(page).toHaveURL(new RegExp(`/tasks/search\\?q=${tok}`));
  await ready(page);
  await expect(box(page)).toHaveAttribute('placeholder', 'Search jobs');
  // ...and from there, back to questions.
  await page.getByRole('link', { name: 'Search questions instead' }).click();
  await expect(page).toHaveURL(new RegExp(`/questions/search\\?q=${tok}`));
  await ready(page);
  await expect(box(page)).toHaveAttribute('placeholder', 'Search questions');
});

test('gate 8.78: article results gain Search questions instead', async ({ page, server }) => {
  const tok = token();
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);
  await box(page).fill(tok + ' nothing');
  await box(page).press('Enter');
  await expect(page).toHaveURL(/\/kb\/search\?q=/);
  await ready(page);
  const link = page.getByRole('link', { name: 'Search questions instead' });
  await expect(link).toHaveAttribute('href', new RegExp(`/questions/search\\?q=${tok}`));
  await link.click();
  await ready(page);
  await expect(page).toHaveURL(/\/questions\/search\?q=/);
  await expect(box(page)).toHaveAttribute('placeholder', 'Search questions');
});

test('gate 8.80: the search page and its links work with JavaScript switched off', async ({ browser, server }) => {
  const normal = await browser.newContext();
  const p = await normal.newPage();
  const tok = token();
  await seed(p, server.baseURL, tok);
  const context = await browser.newContext({ javaScriptEnabled: false });
  await context.addCookies(await normal.cookies());
  await normal.close();
  const page = await context.newPage();
  await page.goto(server.baseURL + '/questions');
  await page.locator('#search-box').fill(tok);
  await page.locator('#search-box').press('Enter');
  await expect(page).toHaveURL(new RegExp(`/questions/search\\?q=${tok}`));
  await expect(page.locator('main .q-row')).toHaveCount(3);
  await page.getByRole('link', { name: 'Search articles instead' }).click();
  await expect(page).toHaveURL(new RegExp(`/kb/search\\?q=${tok}`));
  await context.close();
});
