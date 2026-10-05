import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { idFromHref, myPersonID, seedJob } from '../helpers/tasks';

type Page = import('@playwright/test').Page;

// SPEC gates 7.20-7.24, 7.26 (B13.4, D-94): the top bar's box finds jobs on
// Tasks pages and articles everywhere else.

test.describe.configure({ timeout: 60_000 });

// A word no other test uses, made of letters and digits only (a search word).
const token = () => 'q' + Math.random().toString(36).slice(2, 9) + 'x';

const box = (page: Page) => page.locator('#search-box');
const rows = (page: Page) => page.locator('.search-panel .search-result');

async function seed(page: Page, baseURL: string, tok: string) {
  const me = await signInAsNewPerson(page, baseURL, '/tasks/board');
  const id = await myPersonID(page, baseURL);
  await seedJob(page, baseURL, { title: `${tok} printer repair`, stage: 'doing', personID: id, due: '19/09/2026' });
  await seedJob(page, baseURL, {
    title: 'Order toner',
    notes: `The cartridge is low. ${'Filler sentence goes here. '.repeat(8)}Ask about the ${tok} model before ordering. ${'More filler text. '.repeat(8)}`,
    stage: 'todo',
  });
  await seedJob(page, baseURL, { title: `${tok} finished job`, stage: 'done' });
  await seedJob(page, baseURL, { title: `${tok} removed job`, stage: 'todo' });
  // Remove one, the way a person does.
  await page.goto(baseURL + '/tasks/board?q=' + encodeURIComponent(`${tok} removed job`));
  await ready(page);
  const href = await page.locator('.task-card-title a').first().getAttribute('href');
  const res = await page.request.post(baseURL + '/tasks/' + idFromHref(href!) + '/remove', {
    headers: { origin: baseURL },
    maxRedirects: 0,
  });
  expect(res.status()).toBeLessThan(400);
  return me;
}

test('gate 7.20: the box reads Search jobs on every Tasks page, and Search articles elsewhere', async ({ page, server }) => {
  const tok = token();
  await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/tasks/board?q=' + tok);
  await ready(page);
  const href = await page.locator('.task-card-title a').first().getAttribute('href');

  for (const path of ['/tasks', '/tasks/board', '/tasks/team', href!, '/tasks/finished', '/tasks/removed', '/tasks/search?q=zz']) {
    await page.goto(server.baseURL + path);
    await ready(page);
    await expect(box(page), path).toHaveAttribute('placeholder', 'Search jobs');
    await expect(page.locator('label[for="search-box"]')).toHaveText('Search jobs');
  }
  for (const path of ['/kb', '/calendar', '/briefing', '/settings/about']) {
    await page.goto(server.baseURL + path);
    await ready(page);
    await expect(box(page), path).toHaveAttribute('placeholder', 'Search articles');
    await expect(page.locator('form.search')).toHaveAttribute('action', '/kb/search');
  }
});

test('gates 7.21 and 7.22: results as you type, with the words marked; finished found, removed not', async ({ page, server }) => {
  const tok = token();
  const me = await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);

  await box(page).fill(tok);
  await expect(rows(page)).toHaveCount(3);
  // Unfinished before finished; a title match before a notes-only match.
  await expect(rows(page).nth(0).locator('.search-result-title')).toContainText('printer repair');
  await expect(rows(page).nth(1).locator('.search-result-title')).toContainText('Order toner');
  await expect(rows(page).nth(2).locator('.search-result-title')).toContainText('finished job');
  await expect(page.locator('.search-panel')).not.toContainText('removed job');

  const first = rows(page).nth(0);
  await expect(first.locator('.search-result-title mark')).toHaveText(tok);
  await expect(first.locator('.search-job-meta')).toContainText('In progress');
  await expect(first.locator('.search-job-meta')).toContainText(me);
  await expect(first.locator('.search-job-meta')).toContainText('19 Sep');
  // The match is in the notes: a short passage with the word marked.
  const second = rows(page).nth(1);
  await expect(second.locator('.search-snippet mark')).toHaveText(tok);
  expect((await second.locator('.search-snippet').innerText()).length).toBeLessThan(220);
  // Finished is marked.
  await expect(rows(page).nth(2).locator('.stamp')).toHaveText('Finished');
  await expect(rows(page).nth(0).locator('.stamp')).toHaveCount(0);
  await axeCheck(page);

  // Every word has to be there.
  await box(page).fill(`${tok} cartridge`);
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page).first()).toContainText('Order toner');
  await box(page).fill(`${tok} zzzznothing`);
  await expect(page.locator('.search-panel-empty')).toContainText('No jobs match');
  // The last word may be half-typed.
  await box(page).fill(tok.slice(0, tok.length - 2));
  await expect(rows(page)).toHaveCount(3);
  await box(page).fill('printe');
  await expect(rows(page).first()).toBeVisible();
  // One character finds nothing and shows no panel.
  await box(page).fill('q');
  await expect(page.locator('.search-panel')).toHaveCount(0);
});

test('gate 7.23 and 7.24: a result opens the job window; Enter shows the full page; Search articles instead', async ({ page, server }) => {
  const tok = token();
  await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);

  await box(page).fill(tok);
  await expect(rows(page)).toHaveCount(3);
  // The other search is offered at the foot, with the same words.
  await expect(page.locator('.search-panel .search-instead-link')).toHaveText('Search articles instead');
  await expect(page.locator('.search-panel .search-instead-link')).toHaveAttribute('href', `/kb/search?q=${tok}`);

  await rows(page).nth(1).click();
  await expect(page.locator('#task-window[open]')).toBeVisible();
  await expect(page.locator('#task-window')).toContainText('Order toner');
  await page.keyboard.press('Escape');

  // Enter, with nothing picked, is the full page with the same results.
  await box(page).fill(tok);
  await expect(rows(page)).toHaveCount(3);
  await box(page).press('Enter');
  await expect(page).toHaveURL(new RegExp(`/tasks/search\\?q=${tok}`));
  await ready(page);
  await expect(page.locator('.task-search-results .search-result')).toHaveCount(3);
  await expect(page.locator('.task-search-results .stamp')).toHaveText('Finished');
  await expect(page.locator('main')).not.toContainText('removed job');
  await axeCheck(page);

  // Search articles instead runs the same words as an article search.
  await page.getByRole('link', { name: 'Search articles instead' }).click();
  await expect(page).toHaveURL(new RegExp(`/kb/search\\?q=${tok}`));
  await ready(page);
  await expect(box(page)).toHaveAttribute('placeholder', 'Search articles');
});

test('gate 7.23: from a job\'s own page a result goes to the job\'s page', async ({ page, server }) => {
  const tok = token();
  await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/tasks/board?q=' + tok);
  await ready(page);
  const href = (await page.locator('.task-card-title a').first().getAttribute('href'))!;
  await page.goto(server.baseURL + href);
  await ready(page);
  await box(page).fill(`${tok} finished`);
  await expect(rows(page)).toHaveCount(1);
  await rows(page).first().click();
  await expect(page).toHaveURL(/\/tasks\/\d+$/);
  await expect(page.locator('h1')).toContainText('finished job');
});

test('gate 7.26: the Board\'s Filter by word works exactly as before', async ({ page, server }) => {
  const tok = token();
  await seed(page, server.baseURL, tok);
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await page.locator('#filter-q').fill(`${tok} printer`);
  await page.getByRole('button', { name: 'Filter' }).click();
  await ready(page);
  await expect(page.locator('.task-card')).toHaveCount(1);
  await expect(page.locator('.task-card')).toContainText('printer repair');
  await expect(page.locator('.search-panel')).toHaveCount(0);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 7.23: Enter in the box goes to the results page, and a result is a link to the job', async ({ page, server }) => {
    const tok = token();
    await seed(page, server.baseURL, tok);
    await page.goto(server.baseURL + '/tasks/board');
    await box(page).fill(tok);
    await box(page).press('Enter');
    await expect(page).toHaveURL(new RegExp(`/tasks/search\\?q=${tok}`));
    await expect(page.locator('.task-search-results .search-result')).toHaveCount(3);
    await page.locator('.task-search-results a', { hasText: 'Order toner' }).click();
    await expect(page).toHaveURL(/\/tasks\/\d+$/);
  });
});
