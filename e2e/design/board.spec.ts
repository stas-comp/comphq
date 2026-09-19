import { expect, test } from './fixtures';
import { expectMatchesMockup } from './mockup';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

// The Board (PLAN P4-06, gates 4.15, 4.16, 4.17, 4.21): controls, columns
// and cards, compared with docs/design/mockup.html.

type Page = import('@playwright/test').Page;

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color'];
const BOX = ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding'];

async function addTask(
  page: Page,
  title: string,
  o: { stage?: string; due?: string; people?: string[]; size?: string } = {},
): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  if (o.stage) await page.selectOption('#new-task-stage', o.stage);
  if (o.size) await page.selectOption('#new-task-size', o.size);
  if (o.due) await page.fill('#new-task-due-date', o.due);
  if (o.people) await page.selectOption('#new-task-people', o.people.map((label) => ({ label })));
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// A board with one card of each kind, signed in as the person on them.
async function boardWithCards(page: Page, baseURL: string) {
  const me = await signInAsNewPerson(page, baseURL, '/tasks/board');
  await ready(page);
  const t = {
    idea: uniqueName('Idea card'),
    todo1: uniqueName('First todo'),
    todo2: uniqueName('Second todo'),
    done: uniqueName('Done card'),
  };
  await addTask(page, t.idea, { stage: 'idea' });
  await addTask(page, t.todo1, { stage: 'todo', due: '23/09/2027', people: [me], size: 'M' });
  await addTask(page, t.todo2, { stage: 'todo', size: 'L' });
  await addTask(page, t.done, { stage: 'done' });
  return { me, t };
}

test('gate 4.15: there is no add-a-task form on the Board, and + Add task reaches a page that adds one', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  await expect(page.locator('#new-task-title')).toHaveCount(0);
  await expect(page.locator('.add-task-form')).toHaveCount(0);

  const add = page.getByRole('link', { name: '+ Add task' });
  await expect(add).toHaveClass(/\bbtn\b.*\bprimary\b|\bprimary\b.*\bbtn\b/);
  await add.click();
  await ready(page);
  await expect(page).toHaveURL(/\/tasks\/new$/);

  const title = uniqueName('From the add page');
  await page.fill('#new-task-title', title);
  await page.fill('#new-task-notes', 'A description');
  await page.selectOption('#new-task-size', 'L');
  await page.selectOption('#new-task-stage', 'todo');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await expect(page).toHaveURL(/\/tasks\/board$/);
  await expect(page.locator('.task-column[data-stage="todo"] .task-card', { hasText: title })).toHaveCount(1);
});

test('gate 4.15: the controls are one row — the switch, the filters, the word filter, and + Add task at the right', async ({
  page,
  server,
  mockup,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const tops = await Promise.all(
    ['.tools h1', '.tasks-view-switch', '#filter-person', '.task-filter-search', '#add-task-link'].map(
      async (sel) => (await page.locator(sel).first().boundingBox())!,
    ),
  );
  const centres = tops.map((b) => b.y + b.height / 2);
  expect(Math.max(...centres) - Math.min(...centres), 'all controls share a row').toBeLessThan(30);
  const add = tops[tops.length - 1];
  const row = (await page.locator('.tools').boundingBox())!;
  expect(add.x + add.width).toBeGreaterThan(row.x + row.width - 4); // the right-hand end

  await expectMatchesMockup(mockup, page, { mockup: '.tools h1', screen: 'board', app: '.tools h1' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.tools', screen: 'board', app: '.tools' }, ['display', 'align-items', 'column-gap', 'flex-wrap']);
  await expectMatchesMockup(mockup, page, { mockup: '.tools .btn.primary', screen: 'board', app: '#add-task-link' }, [...TEXT, ...BOX]);
  await expectMatchesMockup(mockup, page, { mockup: '.tools .seg', screen: 'board', app: '.tasks-view-switch' }, ['display', 'border-top-color', 'border-radius']);
  await expectMatchesMockup(mockup, page, { mockup: '.tools .search', screen: 'board', app: '.task-filter-search' }, [...BOX, 'width', 'font-size']);
});

test('gate 4.17: a column is the mockup column, and To do is tinted with the accent wash', async ({ page, server, mockup }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const col = ['background-color', 'border-radius', 'padding', 'display', 'row-gap', 'outline-style'];
  await expectMatchesMockup(mockup, page, { mockup: '.col:not(.todo)', screen: 'board', app: '.task-column[data-stage="idea"]' }, col);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.col.todo', screen: 'board', app: '.task-column[data-stage="todo"]' },
    [...col, 'outline-width', 'outline-color'],
  );
  const wash = await page.locator('.task-column[data-stage="todo"]').evaluate((el) => getComputedStyle(el).backgroundColor);
  expect(wash).toBe('rgb(255, 243, 234)'); // --color-accent-wash
  const plain = await page.locator('.task-column[data-stage="idea"]').evaluate((el) => getComputedStyle(el).backgroundColor);
  expect(plain).not.toBe(wash);
});

test('gate 4.17: column headings are condensed capitals with a count beside them, and To do says Top = most important', async ({
  page,
  server,
  mockup,
}) => {
  const { t } = await boardWithCards(page, server.baseURL);
  await expectMatchesMockup(mockup, page, { mockup: '.col-head', screen: 'board', app: '.task-column[data-stage="idea"] .task-column-head' }, ['display', 'align-items', 'column-gap', 'padding']);
  await expectMatchesMockup(mockup, page, { mockup: '.col-head h2', screen: 'board', app: '.task-column[data-stage="idea"] h2' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.col-head .n', screen: 'board', app: '.task-column[data-stage="idea"] .task-column-count' }, TEXT);
  await expectMatchesMockup(mockup, page, { mockup: '.col-head .hint', screen: 'board', app: '.task-column-hint' }, TEXT);
  // Pushed to the right-hand end of the heading row (an auto margin has no comparable computed value).
  const head = (await page.locator('.task-column[data-stage="todo"] .task-column-head').boundingBox())!;
  const hint = (await page.locator('.task-column-hint').boundingBox())!;
  expect(hint.x + hint.width).toBeGreaterThan(head.x + head.width - 6);
  await expect(page.locator('.task-column-hint')).toHaveText('Top = most important');
  await expect(page.locator('.task-column-hint')).toHaveCount(1);

  for (const stage of ['idea', 'todo', 'doing', 'done']) {
    const col = page.locator(`.task-column[data-stage="${stage}"]`);
    const shown = await col.locator('.task-card').count();
    await expect(col.locator('.task-column-count'), stage).toHaveText(String(shown));
  }
  expect(t.idea).toBeTruthy();
});

test('gate 4.16: a card is the mockup card, with a large accent priority number on To do cards only', async ({ page, server, mockup }) => {
  const { t } = await boardWithCards(page, server.baseURL);
  const todo1 = page.locator('.task-column[data-stage="todo"] .task-card', { hasText: t.todo1 });
  await expectMatchesMockup(mockup, page, { mockup: '.col .card', screen: 'board', app: '.task-column[data-stage="idea"] .task-card' }, [...BOX.slice(0, 5), 'padding', 'display', 'row-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.col .card h3', screen: 'board', app: '.task-column[data-stage="idea"] .task-card-title' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.col.todo .card .rank', screen: 'board', app: '.task-column[data-stage="todo"] .rank' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.col.todo .card .date', screen: 'board', app: '.task-column[data-stage="todo"] .task-card-due' }, [...TEXT, 'font-variant-numeric']);

  await expect(page.locator('.task-column:not([data-stage="todo"]) .rank')).toHaveCount(0);
  const ranks = await page.locator('.task-column[data-stage="todo"] .task-card').evaluateAll((cards) => cards.map((c) => c.querySelector('.rank')?.textContent));
  expect(ranks.slice(-2)).toEqual([String(ranks.length - 1), String(ranks.length)]);
  await expect(todo1.locator('.rank')).not.toHaveText('');
});

test('gate 4.16: a card shows rank, title, due date, people, size chip, then its icon buttons, in that order', async ({ page, server }) => {
  const { t, me } = await boardWithCards(page, server.baseURL);
  const card = page.locator('.task-column[data-stage="todo"] .task-card', { hasText: t.todo1 });
  const box = async (sel: string) => (await card.locator(sel).first().boundingBox())!;
  const rank = await box('.rank');
  const title = await box('.task-card-title');
  const due = await box('.task-card-due');
  const people = await box('.people');
  const size = await box('.size');
  const buttons = await box('.task-card-actions');

  expect(rank.x + rank.width, 'rank before title').toBeLessThanOrEqual(title.x + 2);
  expect(title.y + title.height, 'title above the date row').toBeLessThanOrEqual(due.y + 2);
  expect(due.x + due.width, 'due date before people').toBeLessThanOrEqual(people.x + 2);
  expect(people.y + people.height, 'people above the size row').toBeLessThanOrEqual(size.y + 2);
  expect(size.x + size.width, 'size chip before the icon buttons').toBeLessThanOrEqual(buttons.x + 2);
  await expect(card.locator('.task-card-due')).toHaveText('Thu 23 Sep 2027');
  await expect(card.locator('.av')).toHaveAttribute('title', me);
  await expect(card.locator('.size')).toHaveText('Medium');
  // The icon buttons are named for the task.
  await expect(card.getByRole('button', { name: `Move ${t.todo1} up` })).toBeVisible();
  await expect(card.getByRole('button', { name: `Move ${t.todo1} down` })).toBeVisible();
});

test('gate 4.21: a finished card has its title struck through in grey', async ({ page, server, mockup }) => {
  const { t } = await boardWithCards(page, server.baseURL);
  const done = page.locator('.task-column[data-stage="done"] .task-card', { hasText: t.done });
  await expect(done).toHaveClass(/\bdone\b/);
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.card.done h3', screen: 'board', app: '.task-column[data-stage="done"] .task-card.done .task-card-title' },
    ['color', 'text-decoration-line', 'text-decoration-color'],
  );
  await expect(done.locator('.task-card-title')).toHaveCSS('text-decoration-line', 'line-through');
  await expect(done.locator('.task-card-title')).toHaveCSS('color', 'rgb(90, 96, 114)');
  // An unfinished card is not struck through.
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: t.idea }).locator('.task-card-title')).toHaveCSS('text-decoration-line', 'none');
});

test('the Finished and Removed links sit under the Done column, as the mockup shows', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const done = page.locator('.task-column[data-stage="done"]');
  await expect(done.getByRole('link', { name: 'Finished tasks' })).toBeVisible();
  await expect(done.getByRole('link', { name: 'Removed tasks' })).toBeVisible();
});

test('standard page checks for the Board with cards and for the add-task page', async ({ page, server }) => {
  await boardWithCards(page, server.baseURL);
  await axeCheck(page);
  await expectNoSideScroll(page);
  await openNewTask(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
