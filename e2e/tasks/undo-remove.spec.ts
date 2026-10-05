import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { myPersonID, seedJobs } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 7.15-7.18 (B13.3, D-93): Undo after removing a job.

const todoTitles = (page: Page) =>
  page.locator('.task-column[data-stage="todo"] .task-card-title').allInnerTexts();
const card = (page: Page, title: string) => page.locator('.task-card', { hasText: title });
const toast = (page: Page) => page.locator('#toast-region .toast');

async function setup(page: Page, baseURL: string, n = 3) {
  const name = await signInAsNewPerson(page, baseURL, '/tasks/board');
  const id = await myPersonID(page, baseURL);
  const prefix = uniqueName('Undo');
  const titles = Array.from({ length: n }, (_, i) => `${prefix} ${String.fromCharCode(65 + i)}`);
  await seedJobs(page, baseURL, titles, 'todo', id);
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
  return { name, id, titles };
}

test('gate 7.15: removing a job shows Removed "title" and Undo in a live region', async ({ page, server }) => {
  const { titles } = await setup(page, server.baseURL);
  // The region exists before there is anything in it, so a reader announces what arrives.
  await expect(page.locator('#toast-region')).toHaveAttribute('role', 'status');
  await expect(toast(page)).toHaveCount(0);

  await card(page, titles[1]).getByRole('button', { name: `Remove ${titles[1]}` }).click();
  await expect(toast(page)).toHaveCount(1);
  await expect(toast(page)).toContainText(`Removed “${titles[1]}”`);
  await expect(toast(page).getByRole('button', { name: 'Undo' })).toBeVisible();
  await expect(card(page, titles[1])).toHaveCount(0);
  await axeCheck(page);
});

test('gate 7.16: Undo puts the job back in the same place, with its people, steps and History', async ({ page, server }) => {
  const { name, titles } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  // B has a step; it already has me on it.
  await page.goto(server.baseURL + (await card(page, b).locator('.task-card-title a').getAttribute('href')));
  await ready(page);
  await page.getByPlaceholder('Add a step').fill('Buy toner');
  await page.keyboard.press('Enter');
  await expect(page.locator('.task-steps .step-words', { hasText: 'Buy toner' })).toHaveCount(1);
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);

  await card(page, b).getByRole('button', { name: `Remove ${b}` }).click();
  await expect(toast(page)).toHaveCount(1);
  expect(await todoTitles(page)).toEqual([a, c]);

  await toast(page).getByRole('button', { name: 'Undo' }).click();
  await expect(toast(page)).toHaveCount(0);
  expect(await todoTitles(page)).toEqual([a, b, c]);
  await expect(card(page, b).locator('.people .av')).toHaveCount(1);

  // Still true after a refresh, and History has both rows.
  await page.reload();
  await ready(page);
  expect(await todoTitles(page)).toEqual([a, b, c]);
  await page.goto(server.baseURL + (await card(page, b).locator('.task-card-title a').getAttribute('href')));
  await ready(page);
  await expect(page.locator('.task-steps .step-words', { hasText: 'Buy toner' })).toHaveCount(1);
  const history = page.locator('.task-activity-list');
  await expect(history).toContainText(`${name} removed this`);
  await expect(history).toContainText(`${name} undid the removal`);
});

test('gate 7.16: if the order changed meanwhile, the job goes back next to the one it sat beside', async ({ page, server, browser }) => {
  const { titles } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  await card(page, b).getByRole('button', { name: `Remove ${b}` }).click(); // B sat just before C
  await expect(toast(page)).toHaveCount(1);

  // On another computer C is moved to the top, above A.
  const other = await browser.newContext();
  await other.addCookies(await page.context().cookies(server.baseURL));
  const op = await other.newPage();
  await op.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(op);
  await op.locator('.task-card', { hasText: c }).getByRole('button', { name: `Move ${c} up` }).click();
  await ready(op);
  // (The button's plain form returns to the whole Board; look at mine again.)
  await op.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(op);
  expect(await todoTitles(op)).toEqual([c, a]);
  await other.close();

  await toast(page).getByRole('button', { name: 'Undo' }).click();
  await expect(toast(page)).toHaveCount(0);
  expect(await todoTitles(page)).toEqual([b, c, a]); // B directly before C, wherever C now is
});

test('gate 7.16: Undo after somebody else restored it is not an error', async ({ page, server, browser }) => {
  const { titles } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  await card(page, b).getByRole('button', { name: `Remove ${b}` }).click();
  await expect(toast(page)).toHaveCount(1);

  // Someone else brings it back from Removed tasks first.
  const other = await browser.newContext();
  await other.addCookies(await page.context().cookies(server.baseURL));
  const op = await other.newPage();
  await op.goto(server.baseURL + '/tasks/removed');
  await ready(op);
  await op.locator('.task-simple-list li', { hasText: b }).getByRole('button', { name: 'Restore' }).click();
  await other.close();

  await toast(page).getByRole('button', { name: 'Undo' }).click();
  await expect(toast(page)).toHaveCount(0);
  await expect(page.locator('[role="alert"]')).toHaveCount(0);
  const now = await todoTitles(page);
  expect(now.filter((t) => t === b)).toHaveLength(1); // once, not twice
  expect(now).toContain(a);
  expect(now).toContain(c);
});

test('gate 7.17: from a job\'s own page, Remove returns to the Board with the message, and Undo works', async ({ page, server }) => {
  const { name, titles } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  await page.goto(server.baseURL + (await card(page, b).locator('.task-card-title a').getAttribute('href')));
  await ready(page);
  await page.getByRole('button', { name: 'Remove task' }).click();
  await ready(page);
  await expect(page).toHaveURL(/\/tasks\/board/);
  await expect(toast(page)).toContainText(`Removed “${b}”`);
  // The address is cleaned, so a reload doesn't offer it again.
  await expect.poll(() => new URL(page.url()).searchParams.has('removed')).toBe(false);

  await toast(page).getByRole('button', { name: 'Undo' }).click();
  await expect(toast(page)).toHaveCount(0);
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  expect(await todoTitles(page)).toEqual([a, b, c]);
  expect(name).toBeTruthy();
});

test('gate 7.15: the message goes after 10 seconds', async ({ page, server }) => {
  const { titles } = await setup(page, server.baseURL);
  await page.clock.install();
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await card(page, titles[0]).getByRole('button', { name: `Remove ${titles[0]}` }).click();
  await expect(toast(page)).toHaveCount(1);

  await page.clock.runFor(5_000);
  await expect(toast(page)).toHaveCount(1); // still there well before ten seconds
  await page.clock.runFor(6_000);
  await expect(toast(page)).toHaveCount(0); // gone after
});

test('gate 7.18: Removed tasks is unchanged: a job not undone can still be restored there, to the bottom', async ({ page, server }) => {
  const { titles } = await setup(page, server.baseURL);
  const [a, b, c] = titles;
  await card(page, a).getByRole('button', { name: `Remove ${a}` }).click();
  await expect(toast(page)).toHaveCount(1);
  await page.goto(server.baseURL + '/tasks/removed');
  await ready(page);
  await expect(page.locator('main')).toContainText(a);
  await page.locator('.task-simple-list li', { hasText: a }).getByRole('button', { name: 'Restore' }).click();
  await ready(page);
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  expect(await todoTitles(page)).toEqual([b, c, a]);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 7.17: the Board shows the message after the reload, with an Undo button', async ({ page, server }) => {
    const { titles } = await setup(page, server.baseURL);
    const [a, b, c] = titles;
    await card(page, b).getByRole('button', { name: `Remove ${b}` }).click();
    await expect(toast(page)).toContainText(`Removed “${b}”`);
    await page.goto(page.url()); // still there until the next action
    await toast(page).getByRole('button', { name: 'Undo' }).click();
    await page.goto(server.baseURL + '/tasks/board?person=mine');
    expect(await todoTitles(page)).toEqual([a, b, c]);
    await expect(toast(page)).toHaveCount(0);
  });
});
