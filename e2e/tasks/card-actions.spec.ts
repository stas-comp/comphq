import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// Card actions (PLAN P4-07, gates 4.18, 4.19, 4.20): remove, give back and
// assign, straight from a card, over endpoints that already exist.

async function addTask(page: Page, title: string, stage: string, people: string[] = []): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  if (people.length) await page.selectOption('#new-task-people', people.map((label) => ({ label })));
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

const card = (page: Page, title: string) => page.locator('.task-card', { hasText: title });

// A second person, signed in in their own browser context.
async function secondPerson(browser: import('@playwright/test').Browser, baseURL: string) {
  const context = await browser.newContext();
  const page = await context.newPage();
  const name = await signInAsNewPerson(page, baseURL, '/tasks/board');
  await ready(page);
  return { context, page, name };
}

async function activityOf(page: Page, baseURL: string, title: string): Promise<string> {
  await card(page, title).locator('.task-card-title a').click();
  await ready(page);
  const text = await page.locator('.task-activity-list').innerText();
  await page.goto(baseURL + '/tasks/board');
  await ready(page);
  return text;
}

test('gate 4.18: the remove icon takes a card to Removed tasks, and it can be brought back', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Remove me');
  await addTask(page, title, 'todo');

  const remove = card(page, title).getByRole('button', { name: `Remove ${title}` });
  await expect(remove).toHaveAttribute('title', `Remove ${title}`);
  await remove.click();
  await ready(page);
  await expect(card(page, title)).toHaveCount(0);

  await page.goto(server.baseURL + '/tasks/removed');
  await ready(page);
  await expect(card(page, title)).toHaveCount(1);
  await card(page, title).getByRole('button', { name: 'Restore' }).click();
  await ready(page);
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await expect(page.locator('.task-column[data-stage="todo"]').locator('.task-card', { hasText: title })).toHaveCount(1);
});

test('gate 4.19: give back takes only that person off; the job returns to Up for grabs only when nobody is left', async ({
  page,
  server,
  browser,
}) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const other = await secondPerson(browser, server.baseURL);
  await page.reload(); // so the new person is in the list
  await ready(page);

  const title = uniqueName('Shared job');
  await addTask(page, title, 'todo', [me, other.name]);

  // Only a person who is on the job is offered give back.
  await other.page.reload();
  await ready(other.page);
  await expect(card(other.page, title).getByRole('button', { name: `Give back ${title}` })).toHaveCount(1);
  await expect(card(page, title).getByRole('button', { name: `Give back ${title}` })).toHaveCount(1);
  const stranger = await secondPerson(browser, server.baseURL);
  await stranger.page.reload();
  await ready(stranger.page);
  await expect(card(stranger.page, title).getByRole('button', { name: /Give back/ })).toHaveCount(0);

  // I give it back: I am off it, the other person stays, and it is not up for grabs.
  await card(page, title).getByRole('button', { name: `Give back ${title}` }).click();
  await ready(page);
  await expect(card(page, title).locator('.av')).toHaveCount(1);
  await expect(card(page, title).locator('.av')).toHaveAttribute('title', other.name);
  await expect(card(page, title).getByRole('button', { name: /Give back/ })).toHaveCount(0);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await expect(page.locator('.myjobs-grabs-list .team-task', { hasText: title })).toHaveCount(0);

  // The last person gives it back: now nobody is on it, so it is up for grabs.
  await other.page.reload();
  await ready(other.page);
  await card(other.page, title).getByRole('button', { name: `Give back ${title}` }).click();
  await ready(other.page);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await expect(page.locator('.myjobs-grabs-list .team-task', { hasText: title })).toHaveCount(1);

  await other.context.close();
  await stranger.context.close();
});

test('gate 4.20: the people circles open a menu; picking puts a person on, picking again takes them off, several at once, without leaving the page', async ({
  page,
  server,
  browser,
}) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const other = await secondPerson(browser, server.baseURL);
  await page.reload();
  await ready(page);

  const title = uniqueName('Assign me');
  await addTask(page, title, 'idea');
  await expect(card(page, title).locator('.av')).toHaveCount(0);

  // The page must never reload: mark it, and check the mark survives every change.
  await page.evaluate(() => ((window as unknown as { __kept: number }).__kept = 1));
  const kept = () => page.evaluate(() => (window as unknown as { __kept?: number }).__kept);

  const menu = () => card(page, title).locator('details.people-menu');
  await menu().locator('summary').click();
  await expect(menu()).toHaveJSProperty('open', true);
  const item = (name: string) => card(page, title).locator('.people-menu-item', { hasText: name });
  await expect(item(me)).toHaveAttribute('aria-pressed', 'false');

  await item(me).click();
  await expect(card(page, title).locator('.people > .av')).toHaveCount(1);
  await expect(menu()).toHaveJSProperty('open', true); // stays open for the next pick
  await item(other.name).click();
  await expect(card(page, title).locator('.people > .av')).toHaveCount(2); // several people on one job
  await expect(item(me)).toHaveAttribute('aria-pressed', 'true');
  await expect(item(other.name)).toHaveAttribute('aria-pressed', 'true');

  await item(other.name).click(); // a selected name takes them off
  await expect(card(page, title).locator('.people > .av')).toHaveCount(1);
  await expect(card(page, title).locator('.people > .av')).toHaveAttribute('title', me);
  expect(await kept(), 'the page did not reload').toBe(1);

  // Recorded in Activity like any other assignment.
  await page.reload();
  await ready(page);
  const activity = await activityOf(page, server.baseURL, title);
  expect(activity).toContain(`${me} assigned ${me}`);
  expect(activity).toContain(`${me} assigned ${other.name}`);
  expect(activity).toContain(`${me} unassigned ${other.name}`);
  await other.context.close();
});

test('gate 4.20: a job nobody is on offers Assign, and the menu passes the accessibility check open', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Nobody yet');
  await addTask(page, title, 'todo');
  await expect(card(page, title).locator('.people-menu-empty')).toHaveText('Assign');
  await card(page, title).locator('details.people-menu > summary').click();
  await expect(card(page, title).locator('.people-menu-panel')).toBeVisible();
  await axeCheck(page);
  await expectNoSideScroll(page);
});

// Every action is a plain form, so it works with no script at all.
test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('remove, give back and assign each work as a plain form submit', async ({ page, server }) => {
    // Signing in and adding a task use plain forms too.
    const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
    const keepTitle = uniqueName('No script keep');
    const removeTitle = uniqueName('No script remove');
    for (const [title, people] of [[keepTitle, [me]], [removeTitle, []]] as const) {
      await page.goto(server.baseURL + '/tasks/new');
      await page.fill('#new-task-title', title);
      await page.selectOption('#new-task-stage', 'todo');
      if (people.length) await page.selectOption('#new-task-people', people.map((label) => ({ label })));
      await page.click('.add-task-form button[type="submit"]');
    }
    await expect(card(page, keepTitle)).toHaveCount(1);

    // Give back: I was on it, and now I'm not.
    await card(page, keepTitle).getByRole('button', { name: `Give back ${keepTitle}` }).click();
    await expect(page).toHaveURL(/\/tasks\/board$/);
    await expect(card(page, keepTitle).locator('.av')).toHaveCount(0);

    // Assign: a native <details> opens with no script, and the choice posts.
    await card(page, keepTitle).locator('details.people-menu > summary').click();
    await card(page, keepTitle).locator('.people-menu-item', { hasText: me }).click();
    await expect(page).toHaveURL(/\/tasks\/board$/);
    await expect(card(page, keepTitle).locator('.people > .av')).toHaveCount(1);

    // Remove.
    await card(page, removeTitle).getByRole('button', { name: `Remove ${removeTitle}` }).click();
    await expect(card(page, removeTitle)).toHaveCount(0);
    await page.goto(server.baseURL + '/tasks/removed');
    await expect(card(page, removeTitle)).toHaveCount(1);
  });
});
