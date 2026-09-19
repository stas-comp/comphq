import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask, openTaskPage } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// The task window, reading and editing a task, and its history (PLAN P4-09,
// gates 4.23, 4.24, 4.27, 4.29).

const win = (page: Page) => page.locator('#task-window');
const titleLink = (page: Page, title: string) => page.locator('.task-card', { hasText: title }).locator('.task-card-title a');
const card = (page: Page, title: string) => page.locator('.task-card', { hasText: title });

async function addTask(
  page: Page,
  title: string,
  o: { stage?: string; notes?: string; size?: string; due?: string; people?: string[] } = {},
): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  if (o.notes) await page.fill('#new-task-notes', o.notes);
  if (o.stage) await page.selectOption('#new-task-stage', o.stage);
  if (o.size) await page.selectOption('#new-task-size', o.size);
  if (o.due) await page.fill('#new-task-due-date', o.due);
  if (o.people) await page.selectOption('#new-task-people', o.people.map((label) => ({ label })));
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

test('gate 4.23: clicking a card opens the window to read: title, description as text, people, size, column and due date', async ({
  page,
  server,
}) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Readable job');
  await addTask(page, title, { stage: 'doing', notes: 'Print three sets per room.\nStaple each set.', size: 'L', due: '23/09/2027', people: [me] });

  await titleLink(page, title).click();
  await expect(win(page)).toBeVisible();
  await expect(page).toHaveURL(/\/tasks\/board$/); // over the Board, not a new page
  await expect(win(page).getByRole('heading', { name: title })).toBeVisible();

  const notes = win(page).locator('.task-read-notes');
  await expect(notes).toContainText('Print three sets per room.');
  await expect(notes).toContainText('Staple each set.');
  // Laid out as text, not sitting in a box: no form control, no border, no fill.
  await expect(win(page).locator('textarea, input:not([type="hidden"])')).toHaveCount(0);
  await expect(notes).toHaveCSS('border-top-width', '0px');
  await expect(notes).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
  expect(await notes.evaluate((el) => getComputedStyle(el).whiteSpace)).toBe('pre-line'); // its line breaks are kept

  await expect(win(page).locator('.size')).toHaveText('Large');
  await expect(win(page).locator('.task-read-facts')).toContainText('In progress');
  await expect(win(page).locator('.task-read-facts .date')).toHaveText('Thu 23 Sep 2027');
  await expect(win(page).locator('.task-read-people .av')).toHaveAttribute('title', me);
  await expect(win(page).locator('.task-read-names')).toHaveText(me);
  await expect(win(page).getByRole('button', { name: 'Edit' })).toHaveClass(/\bprimary\b/);
  await expect(win(page).getByRole('button', { name: 'Edit' })).toBeVisible();
});

test('gate 4.24: History sits at the foot behind an expander, closed when the window opens, listing what happened', async ({
  page,
  server,
}) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('History job');
  await addTask(page, title, { stage: 'todo' });
  await titleLink(page, title).click();

  const history = win(page).locator('details.task-history');
  await expect(history).toHaveJSProperty('open', false);
  await expect(history.locator('.task-activity-list')).toBeHidden();
  // It is the last thing in the reading face, after the description.
  const order = await win(page).locator('.task-read').evaluate((el) => Array.from(el.children).map((c) => c.className || c.tagName));
  expect(order.findIndex((c) => /task-history/.test(c))).toBeGreaterThan(order.findIndex((c) => /task-read-notes/.test(c)));

  await history.locator('summary').click();
  await expect(history).toHaveJSProperty('open', true);
  await expect(history.locator('.task-activity-list')).toContainText(`${me} created this`);
});

test('gate 4.23, 4.29: Edit turns the window into the form filled in; saving updates the board without a reload and records the same history as the page', async ({
  page,
  server,
}) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Editable job');
  await addTask(page, title, { stage: 'todo', notes: 'First draft', size: 'M' });
  await page.evaluate(() => ((window as unknown as { __kept: number }).__kept = 1));

  await titleLink(page, title).click();
  await win(page).getByRole('button', { name: 'Edit' }).click();

  // The form of the new-task window, filled in.
  await expect(win(page).getByRole('heading', { name: 'Edit task' })).toBeVisible();
  await expect(win(page).locator('#details-title')).toHaveValue(title);
  await expect(win(page).locator('#details-notes')).toHaveValue('First draft');
  await expect(win(page).locator('#details-size')).toHaveValue('M');
  await expect(win(page).locator('#details-stage')).toHaveValue('todo');
  await win(page).locator('#details-due-date').fill('4/12/2027');
  await win(page).locator('#details-people').selectOption([{ label: me }]);
  await win(page).getByRole('button', { name: 'Save' }).click();

  // Back to reading, updated; the board behind it too, with no reload.
  await expect(win(page).locator('.task-read-facts .date')).toHaveText('Sat 4 Dec 2027');
  await expect(win(page).locator('.task-read-names')).toHaveText(me);
  await expect(card(page, title).locator('.task-card-due')).toHaveText('Sat 4 Dec 2027');
  await expect(card(page, title).locator('.people > .av')).toHaveCount(1);
  expect(await page.evaluate(() => (window as unknown as { __kept?: number }).__kept), 'no reload').toBe(1);

  // History records it, the same rows editing from the page makes.
  await win(page).locator('details.task-history summary').click();
  const windowRows = (await win(page).locator('.task-activity-list li').allInnerTexts()).map((t) => t.replace(/ · .*$/, '').trim());
  expect(windowRows).toEqual(expect.arrayContaining([`${me} changed the due date to Sat 4 Dec 2027`, `${me} assigned ${me}`, `${me} created this`]));
  await win(page).getByRole('button', { name: 'Close' }).click();
  await openTaskPage(page, title);
  const pageRows = (await page.locator('.task-activity-list li').allInnerTexts()).map((t) => t.replace(/ · .*$/, '').trim());
  expect(pageRows.sort()).toEqual(windowRows.sort());
});

test('gate 4.23: Cancel goes back to reading, asking first if something was typed; a refused save keeps the form and what was typed', async ({
  page,
  server,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Cancel job');
  await addTask(page, title, { notes: 'Original' });
  await titleLink(page, title).click();
  await win(page).getByRole('button', { name: 'Edit' }).click();

  let asked = 0;
  page.on('dialog', async (d) => {
    asked++;
    await d.dismiss();
  });
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  await expect(win(page).locator('.task-read')).toBeVisible();
  expect(asked, 'nothing typed, so no question').toBe(0);

  await win(page).getByRole('button', { name: 'Edit' }).click();
  await win(page).locator('#details-notes').fill('Changed my mind');
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  await expect.poll(() => asked).toBe(1);
  await expect(win(page).locator('#details-notes')).toHaveValue('Changed my mind'); // stayed
  page.removeAllListeners('dialog');
  page.on('dialog', (d) => d.accept());
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  await expect(win(page).locator('.task-read-notes')).toHaveText('Original'); // nothing was saved

  // A date that isn't a day is refused with a plain message, and what was typed stays.
  await win(page).getByRole('button', { name: 'Edit' }).click();
  await win(page).locator('#details-notes').fill('Kept as typed');
  await win(page).locator('#details-due-date').fill('31/02/2027');
  await win(page).getByRole('button', { name: 'Save' }).click();
  await expect(win(page).getByRole('alert')).toContainText('day/month/year');
  await expect(win(page).locator('#details-notes')).toHaveValue('Kept as typed');
  await expect(win(page).locator('#details-due-date')).toHaveValue('31/02/2027');
});

test('gate 4.25, 4.26: closing puts you back on the card you opened, with the board as it was', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Focus job');
  await addTask(page, title);
  const link = titleLink(page, title);
  await link.focus();
  await page.keyboard.press('Enter');
  await expect(win(page)).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(win(page)).toBeHidden();
  await expect(link).toBeFocused();
  await expect(page).toHaveURL(/\/tasks\/board$/);
});

test('gate 4.27: every task still has its own page, which works directly and after a refresh, and Calendar links land there', async ({
  page,
  server,
}) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Page job');
  await addTask(page, title, { stage: 'todo', due: '23/09/2027', notes: 'On its own page' });
  const href = (await titleLink(page, title).getAttribute('href'))!;
  expect(href).toMatch(/^\/tasks\/\d+$/);

  await page.goto(server.baseURL + href);
  await ready(page);
  await expect(page.locator('h1')).toHaveText(title);
  await expect(page.locator('#details-notes')).toHaveValue('On its own page');
  await expect(page.locator('.task-activity-list')).toContainText('created this');
  await page.reload(); // refreshing it still works
  await ready(page);
  await expect(page.locator('h1')).toHaveText(title);
  await expect(page.locator('.task-details-form')).toBeVisible();

  // A Calendar link to a task still lands there.
  await page.goto(server.baseURL + '/calendar?month=2027-09');
  await ready(page);
  await page.locator('.calendar-day[data-date="2027-09-23"] .calendar-chip', { hasText: title }).click();
  await ready(page);
  await expect(page).toHaveURL(new RegExp(href.replace('/', '\\/') + '$'));
  await expect(page.locator('h1')).toHaveText(title);
});

test('gate 4.24: the reading face and the History expanded pass the accessibility check', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Axe job');
  await addTask(page, title, { notes: 'Some text', people: [me], due: '1/1/2028' });
  await titleLink(page, title).click();
  await expect(win(page)).toBeVisible();
  await axeCheck(page);
  await win(page).locator('details.task-history summary').click();
  await axeCheck(page);
  await win(page).getByRole('button', { name: 'Edit' }).click();
  await expect(win(page).locator('#details-title')).toBeVisible();
  await axeCheck(page);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 4.28: a card title is an ordinary link to the task page, which reads and edits it', async ({ page, server }) => {
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const title = uniqueName('No script job');
    await page.goto(server.baseURL + '/tasks/new');
    await page.fill('#new-task-title', title);
    await page.click('.add-task-form button[type="submit"]');
    await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').click();
    await expect(page).toHaveURL(/\/tasks\/\d+$/);
    await expect(page.locator('h1')).toHaveText(title);
    await page.fill('#details-notes', 'Edited with no script');
    await page.selectOption('#details-people', [{ label: me }]);
    await page.click('.task-details-form button[type="submit"]');
    await expect(page.locator('#details-notes')).toHaveValue('Edited with no script');
  });
});
