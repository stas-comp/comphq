import { expect, test } from './fixtures';
import { expectMatchesMockup } from './mockup';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

// My jobs and Team (PLAN P4-10, gates 4.30, 4.31, 4.32): both screens as
// drawn, including the Ideas group, compared with docs/design/mockup.html.

type Page = import('@playwright/test').Page;

const TEXT = ['font-family', 'font-weight', 'font-size', 'letter-spacing', 'text-transform', 'color'];
const BOX = ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding'];

async function addTask(page: Page, title: string, o: { stage: string; size?: string; due?: string; people?: string[] }): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', o.stage);
  if (o.size) await page.selectOption('#new-task-size', o.size);
  if (o.due) await page.fill('#new-task-due-date', o.due);
  if (o.people) await page.selectOption('#new-task-people', o.people.map((label) => ({ label })));
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// One signed-in person with a job in each group, a shared job, and something for grabs.
async function seeded(page: Page, browser: import('@playwright/test').Browser, baseURL: string) {
  const me = await signInAsNewPerson(page, baseURL, '/tasks/board');
  await ready(page);
  const otherCtx = await browser.newContext();
  const otherPage = await otherCtx.newPage();
  const other = await signInAsNewPerson(otherPage, baseURL, '/tasks/board');
  await ready(otherPage);
  await otherCtx.close();
  const t = {
    doing: uniqueName('Working job'),
    todo: uniqueName('Shared up next'),
    idea: uniqueName('My idea'),
    grab: uniqueName('Grab me'),
    grabIdea: uniqueName('Grab this idea'),
    loose: uniqueName('Nobody has this'),
  };
  await page.goto(baseURL + '/tasks/board');
  await ready(page);
  await addTask(page, t.doing, { stage: 'doing', size: 'L', due: '17/10/2027', people: [me] });
  await addTask(page, t.todo, { stage: 'todo', size: 'M', due: '23/09/2027', people: [me, other] });
  await addTask(page, t.idea, { stage: 'idea', people: [me] });
  await addTask(page, t.grab, { stage: 'todo', size: 'S' });
  await addTask(page, t.grabIdea, { stage: 'idea' });
  await addTask(page, t.loose, { stage: 'doing', size: 'M' });
  return { me, other, t };
}

test('gate 4.30: My jobs is tinted, Up for grabs is a dashed outline with no fill, and the drop target has its wording', async ({
  page,
  server,
  browser,
  mockup,
}) => {
  const { t } = await seeded(page, browser, server.baseURL);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);

  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.area.mine', screen: 'tasks', app: '.myjobs-mine' },
    ['background-color', 'border-radius', 'padding', 'display', 'row-gap', 'outline-width', 'outline-style', 'outline-color'],
  );
  await expectMatchesMockup(
    mockup,
    page,
    { mockup: '.area.grabs', screen: 'tasks', app: '.myjobs-grabs' },
    ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding', 'display', 'row-gap'],
  );
  expect(await page.locator('.myjobs-grabs').evaluate((el) => getComputedStyle(el).borderTopStyle)).toBe('dashed');
  expect(await page.locator('.myjobs-grabs').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe('rgba(0, 0, 0, 0)');
  expect(await page.locator('.myjobs-mine').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe('rgb(255, 243, 234)');

  await expectMatchesMockup(mockup, page, { mockup: '.drop', screen: 'tasks', app: '.myjobs-upnext-list .drop' }, [...TEXT, ...BOX, 'text-align']);
  await expect(page.locator('.myjobs-upnext-list .drop')).toHaveText('Drop a job here to take it');
  expect(t.doing).toBeTruthy();
});

test('gate 4.30: the area heads, group labels and cards are the mockup ones', async ({ page, server, browser, mockup }) => {
  await seeded(page, browser, server.baseURL);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  const head = ['display', 'row-gap', 'padding', 'border-bottom-width', 'border-bottom-style', 'border-bottom-color'];
  await expectMatchesMockup(mockup, page, { mockup: '.area.mine .area-head', screen: 'tasks', app: '.myjobs-mine .area-head' }, head);
  await expectMatchesMockup(mockup, page, { mockup: '.area.mine .area-head h2', screen: 'tasks', app: '.myjobs-mine .area-head h2' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.area.grabs .area-head p', screen: 'tasks', app: '.myjobs-grabs .area-head p' }, TEXT);
  await expectMatchesMockup(mockup, page, { mockup: '.area .sub', screen: 'tasks', app: '.myjobs-mine .sub' }, [...TEXT, 'padding']);
  await expectMatchesMockup(mockup, page, { mockup: '.area .card', screen: 'tasks', app: '.myjobs-mine .team-task' }, [...BOX.slice(0, 5), 'padding', 'display', 'row-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.area .card h3', screen: 'tasks', app: '.myjobs-mine .team-task .task-card-title' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.area .card .rank', screen: 'tasks', app: '.myjobs-upnext-list .rank' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.area .card .date', screen: 'tasks', app: '.myjobs-mine .team-task .date' }, [...TEXT, 'font-variant-numeric']);
  await expectMatchesMockup(mockup, page, { mockup: '.area .card .also', screen: 'tasks', app: '.myjobs-mine .team-task .also' }, TEXT);
  // Your own workload is drawn in the deep accent.
  await expectMatchesMockup(mockup, page, { mockup: '.area.mine .load .t i', screen: 'tasks', app: '.myjobs-mine .workload-block' }, ['background-color', 'width', 'height']);
  await expect(page.locator('.myjobs-mine .workload-line')).toHaveText('1 large · 1 medium');
});

test('gate 4.32: Take it, Give back, Start and Done ✓ are worded small buttons, and a card opens the task window', async ({
  page,
  server,
  browser,
  mockup,
}) => {
  const { t } = await seeded(page, browser, server.baseURL);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);

  const working = page.locator('.myjobs-working-list .team-task', { hasText: t.doing });
  const upNext = page.locator('.myjobs-upnext-list .team-task', { hasText: t.todo });
  const grab = page.locator('.myjobs-grabs-list .team-task', { hasText: t.grab });
  await expect(working.getByRole('button', { name: 'Give back' })).toBeVisible();
  await expect(working.getByRole('button', { name: 'Done ✓' })).toBeVisible();
  await expect(upNext.getByRole('button', { name: 'Start' })).toBeVisible();
  await expect(grab.getByRole('button', { name: 'Take it' })).toBeVisible();
  for (const b of [working.getByRole('button', { name: 'Give back' }), working.getByRole('button', { name: 'Done ✓' }), upNext.getByRole('button', { name: 'Start' }), grab.getByRole('button', { name: 'Take it' })]) {
    await expect(b).toHaveClass(/\bmini\b/);
  }
  await expectMatchesMockup(mockup, page, { mockup: '.area.grabs .mini.go', screen: 'tasks', app: '.myjobs-grabs-list .mini.go' }, [...TEXT, ...BOX]);
  await expectMatchesMockup(mockup, page, { mockup: '.area.mine .mini.quiet', screen: 'tasks', app: '.myjobs-working-list .mini.quiet' }, [...TEXT, ...BOX]);
  await expectMatchesMockup(mockup, page, { mockup: '.area.mine .acts .mini.go', screen: 'tasks', app: '.myjobs-working-list .acts .mini.go' }, [...TEXT, ...BOX]);

  // A card opens the same window as on the Board.
  await working.locator('.task-card-title a').click();
  await expect(page.locator('#task-window')).toBeVisible();
  await expect(page.locator('#task-window').getByRole('heading', { name: t.doing })).toBeVisible();
  await expect(page).toHaveURL(/\/tasks$/);
  await page.keyboard.press('Escape');
  await expect(page.locator('#task-window')).toBeHidden();
  await expect(working.locator('.task-card-title a')).toBeFocused();
});

test('gate 4.31: on Team, Unassigned is a dashed outline with no fill, your lane is tinted and says YOU, and lane heads carry circle, name, blocks and summary', async ({
  page,
  server,
  browser,
  mockup,
}) => {
  const { me, other } = await seeded(page, browser, server.baseURL);
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const laneOf = (name: string) => page.locator('.team-lane', { has: page.locator('h2', { hasText: name }) });

  // Lane kinds.
  await expectMatchesMockup(mockup, page, { mockup: '.lane:not(.unassigned):not(.me)', screen: 'team', app: `.team-lane:not(.unassigned):not(.me)` }, ['background-color', 'border-radius', 'padding', 'display', 'row-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.lane.unassigned', screen: 'team', app: '.team-lane.unassigned' }, ['background-color', 'border-top-width', 'border-top-style', 'border-top-color', 'border-radius', 'padding']);
  await expectMatchesMockup(mockup, page, { mockup: '.lane.me', screen: 'team', app: '.team-lane.me' }, ['background-color', 'border-radius', 'padding', 'outline-width', 'outline-style', 'outline-color']);
  const unassigned = page.locator('.team-lane.unassigned');
  expect(await unassigned.evaluate((el) => getComputedStyle(el).borderTopStyle)).toBe('dashed');
  expect(await unassigned.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe('rgba(0, 0, 0, 0)');
  // It no longer looks like a person's lane.
  const person = await laneOf(other).evaluate((el) => getComputedStyle(el).backgroundColor);
  expect(person).not.toBe('rgba(0, 0, 0, 0)');

  // Your own lane says YOU; nobody else's does.
  await expect(laneOf(me).locator('.stamp')).toHaveText('You');
  await expect(laneOf(me)).toHaveClass(/\bme\b/);
  await expect(laneOf(other)).not.toHaveClass(/\bme\b/);
  await expect(page.locator('.team-lane .stamp')).toHaveCount(1);
  await expectMatchesMockup(mockup, page, { mockup: '.lane.me .lane-head .stamp', screen: 'team', app: '.team-lane.me .stamp' }, [...TEXT, ...BOX]);

  // The head: circle, name in condensed capitals, workload blocks, summary line.
  await expectMatchesMockup(mockup, page, { mockup: '.lane .lane-head', screen: 'team', app: '.team-lane.me .lane-head' }, ['display', 'row-gap', 'padding', 'border-bottom-width', 'border-bottom-style', 'border-bottom-color']);
  await expectMatchesMockup(mockup, page, { mockup: '.lane-head .who', screen: 'team', app: '.team-lane.me .lane-head .who' }, ['display', 'align-items', 'column-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.lane-head h2', screen: 'team', app: '.team-lane.me .lane-head h2' }, [...TEXT, 'line-height']);
  await expectMatchesMockup(mockup, page, { mockup: '.lane-head .sum', screen: 'team', app: '.team-lane.me .workload-line' }, TEXT);
  await expect(laneOf(me).locator('.lane-head .av')).toHaveClass(/\bme\b/);
  await expect(laneOf(other).locator('.lane-head .av')).not.toHaveClass(/\bme\b/);
  await expect(laneOf(me).locator('.workload-block').first()).toBeVisible();
  await expect(laneOf(me).locator('.workload-line')).toHaveText('1 large · 1 medium');
  await expect(unassigned.locator('.sum')).toContainText('waiting for someone');
  // Your own blocks are the deep accent; another person's are ink.
  await expectMatchesMockup(mockup, page, { mockup: '.lane.me .load .t i', screen: 'team', app: '.team-lane.me .workload-block' }, ['background-color']);
  await expectMatchesMockup(mockup, page, { mockup: '.lane:not(.me) .load .t i', screen: 'team', app: `.team-lane:not(.me):not(.unassigned) .workload-block` }, ['background-color']);
});

test('gate 4.31: a card on Team shows rank and title, size and date, "Also on it", and opens the task window', async ({
  page,
  server,
  browser,
  mockup,
}) => {
  const { me, other, t } = await seeded(page, browser, server.baseURL);
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const mine = page.locator('.team-lane.me');
  const shared = mine.locator('.team-up-next-list .team-task', { hasText: t.todo });
  await expect(shared.locator('.rank')).toHaveText(/^\d+$/);
  await expect(shared.locator('.size')).toHaveText('Medium');
  await expect(shared.locator('.date')).toHaveText('Thu 23 Sep 2027');
  await expect(shared.locator('.also')).toHaveText(`Also on it: ${other}`);
  await expect(shared.getByRole('button', { name: `Move ${t.todo} up` })).toBeVisible();
  await expectMatchesMockup(mockup, page, { mockup: '.lane .card', screen: 'team', app: '.team-lane.me .team-task' }, [...BOX.slice(0, 5), 'padding', 'display', 'row-gap']);
  await expectMatchesMockup(mockup, page, { mockup: '.lane .sub', screen: 'team', app: '.team-lane.me .sub' }, [...TEXT, 'padding']);

  await shared.locator('.task-card-title a').click();
  await expect(page.locator('#task-window').getByRole('heading', { name: t.todo })).toBeVisible();
  await expect(page).toHaveURL(/\/tasks\/team$/);
  await page.keyboard.press('Escape');
  await expect(page.locator('#task-window')).toBeHidden();
  expect(me).toBeTruthy();
});

test('gates 4.48, 4.49: the Ideas group on My jobs and the Ideas heading on Team are styled to match', async ({
  page,
  server,
  browser,
  mockup,
}) => {
  const { me, t } = await seeded(page, browser, server.baseURL);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  const ideaCard = page.locator('.myjobs-ideas-list .team-task', { hasText: t.idea });
  await expect(ideaCard).toHaveCount(1);
  await expect(page.locator('.myjobs-mine h3.sub', { hasText: "Ideas I'm on" })).toHaveCount(1);
  await expectMatchesMockup(mockup, page, { mockup: '.area .card', screen: 'tasks', app: '.myjobs-ideas-list .team-task' }, [...BOX.slice(0, 5), 'padding', 'display']);
  // The idea adds nothing to the workload.
  await expect(page.locator('.myjobs-mine .workload-line')).toHaveText('1 large · 1 medium');

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const lane = page.locator('.team-lane.me');
  await expect(lane.locator('h3.sub', { hasText: 'Ideas' })).toHaveCount(1);
  await expect(lane.locator('.team-ideas-list .team-task', { hasText: t.idea })).toHaveCount(1);
  await expectMatchesMockup(mockup, page, { mockup: '.lane .sub', screen: 'team', app: '.team-lane.me h3.sub' }, [...TEXT, 'padding']);
  expect(me).toBeTruthy();
});

test('standard page checks for My jobs and Team with cards', async ({ page, server, browser }) => {
  await seeded(page, browser, server.baseURL);
  for (const path of ['/tasks', '/tasks/team']) {
    await page.goto(server.baseURL + path);
    await ready(page);
    await axeCheck(page);
    await expectNoSideScroll(page);
  }
});
