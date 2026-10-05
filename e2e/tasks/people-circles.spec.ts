import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { myPersonID, seedJobs } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;
type Locator = import('@playwright/test').Locator;
type Browser = import('@playwright/test').Browser;

// SPEC gates 7.06-7.09 (B13.1, D-92): Team and My jobs get the Board's people
// circles and the same list of names.

test.describe.configure({ timeout: 60_000 });

const lane = (page: Page, name: string): Locator =>
  page.locator('.team-lane', { has: page.locator('h2', { hasText: name }) });
const card = (scope: Locator | Page, title: string): Locator => scope.locator('.team-task', { hasText: title });

async function otherPerson(browser: Browser, baseURL: string) {
  const ctx = await browser.newContext();
  const p = await ctx.newPage();
  const name = await signInAsNewPerson(p, baseURL, '/tasks/team');
  const id = await myPersonID(p, baseURL);
  return { ctx, page: p, name, id };
}

test('gate 7.06: every card on Team and My jobs has the circles, or Assign, and the old drop-down is gone', async ({ page, server, browser }) => {
  const a = await signInAsNewPerson(page, server.baseURL, '/tasks/team');
  const aID = await myPersonID(page, server.baseURL);
  const b = await otherPerson(browser, server.baseURL);
  const [working, upnext, idea, loose] = [uniqueName('Working'), uniqueName('Upnext'), uniqueName('Idea'), uniqueName('Loose')];
  await seedJobs(page, server.baseURL, [working], 'doing', aID);
  await seedJobs(page, server.baseURL, [upnext], 'todo', aID);
  await seedJobs(page, server.baseURL, [idea], 'idea', aID);
  await seedJobs(page, server.baseURL, [loose], 'todo');

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const mine = lane(page, a);
  for (const t of [working, upnext, idea]) {
    const trigger = card(mine, t).locator('.people-menu-trigger');
    await expect(trigger).toBeVisible();
    await expect(trigger.locator('.av')).toHaveCount(1);
  }
  await expect(card(page.locator('.team-lane.unassigned'), loose).locator('.people-menu-empty')).toHaveText('Assign');
  // The old drop-down and its button are gone.
  await expect(page.locator('.team-assign-form')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Assign to…' })).toHaveCount(0);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  for (const t of [working, upnext, idea]) {
    await expect(card(page.locator('.myjobs-mine'), t).locator('.people-menu-trigger .av')).toHaveCount(1);
  }
  await expect(card(page.locator('.myjobs-grabs'), loose).locator('.people-menu-empty')).toHaveText('Assign');
  await b.ctx.close();
});

test('gate 7.06 and 7.08: any number of people can be put on and taken off, without a reload, with History', async ({ page, server, browser }) => {
  const a = await signInAsNewPerson(page, server.baseURL, '/tasks/team');
  const aID = await myPersonID(page, server.baseURL);
  const b = await otherPerson(browser, server.baseURL);
  const title = uniqueName('Shared job');
  await seedJobs(page, server.baseURL, [title], 'todo', aID);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  await page.evaluate(() => {
    (window as unknown as { __kept: number }).__kept = 1;
  });

  // From A's own lane, add B (impossible before 1.4: the old list swapped people).
  await card(lane(page, a), title).locator('.people-menu-trigger').click();
  await card(lane(page, a), title).locator('.people-menu-item', { hasText: b.name }).click();
  await expect(card(lane(page, b.name), title)).toHaveCount(1);
  await expect(card(lane(page, a), title)).toHaveCount(1);
  await expect(card(lane(page, a), title).locator('.people .av')).toHaveCount(2);
  // The menu stays open for the next pick, on the same card.
  await expect(card(lane(page, a), title).locator('.people-menu-panel')).toBeVisible();

  // Take A off: the job leaves A's lane, B keeps it.
  await card(lane(page, a), title).locator('.people-menu-item', { hasText: a }).click();
  await expect(card(lane(page, a), title)).toHaveCount(0);
  await expect(card(lane(page, b.name), title).locator('.people .av')).toHaveCount(1);
  expect(await page.evaluate(() => (window as unknown as { __kept?: number }).__kept)).toBe(1);

  // History is recorded as on the Board.
  const href = await card(lane(page, b.name), title).locator('.task-card-title a').getAttribute('href');
  await page.goto(server.baseURL + href!);
  await ready(page);
  const history = page.locator('.task-activity-list');
  await expect(history).toContainText(`${a} assigned ${b.name}`);
  await expect(history).toContainText(`${a} unassigned ${a}`);
  await b.ctx.close();
});

test('gate 7.06 and 7.08: the same on My jobs, including a job in Up for grabs', async ({ page, server }) => {
  const a = await signInAsNewPerson(page, server.baseURL, '/tasks');
  const title = uniqueName('Grab me');
  await seedJobs(page, server.baseURL, [title], 'todo');

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await card(page.locator('.myjobs-grabs'), title).locator('.people-menu-trigger').click();
  await card(page.locator('.myjobs-grabs'), title).locator('.people-menu-item', { hasText: a }).click();
  // Now it is mine: in Up next, and no longer up for grabs.
  await expect(card(page.locator('.myjobs-upnext-list'), title)).toHaveCount(1);
  await expect(card(page.locator('.myjobs-grabs'), title)).toHaveCount(0);
  await expect(page).toHaveURL(/\/tasks$/);
});

test('gate 7.07: the list opens fully in view on the last card of a long column in the right-most lane, at 1024 x 700', async ({ page, server, browser }) => {
  test.setTimeout(120_000);
  await page.setViewportSize({ width: 1024, height: 700 });
  // Five other people so the lanes scroll sideways; the last person created is the right-most lane.
  const others = [];
  for (let i = 0; i < 5; i++) others.push(await otherPerson(browser, server.baseURL));
  const last = await signInAsNewPerson(page, server.baseURL, '/tasks/team');
  const lastID = await myPersonID(page, server.baseURL);
  const titles = Array.from({ length: 9 }, (_, i) => `${uniqueName('Long')}-${i}`);
  await seedJobs(page, server.baseURL, titles, 'todo', lastID);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const target = card(lane(page, last), titles[titles.length - 1]);
  // Scroll the lanes all the way right and the page all the way down.
  await page.locator('.team-lanes').evaluate((el) => {
    el.scrollLeft = el.scrollWidth;
  });
  await target.scrollIntoViewIfNeeded();
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  await target.locator('.people-menu-trigger').click();
  const panel = target.locator('.people-menu-panel');
  await expect(panel).toBeVisible();
  const box = (await panel.boundingBox())!;
  const vp = page.viewportSize()!;
  const bar = (await page.locator('.topbar').boundingBox())!;
  expect(box.x, 'left edge inside the window').toBeGreaterThanOrEqual(0);
  expect(box.x + box.width, 'right edge inside the window').toBeLessThanOrEqual(vp.width);
  expect(box.y, 'not under the top bar').toBeGreaterThanOrEqual(bar.y + bar.height - 1);
  expect(box.y + box.height, 'bottom edge inside the window').toBeLessThanOrEqual(vp.height);
  // And it is really on top: its first name is what a click at its centre would hit.
  const hit = await page.evaluate(
    ([x, y]) => document.elementFromPoint(x, y)?.closest('.people-menu-panel') !== null,
    [box.x + box.width / 2, box.y + 20],
  );
  expect(hit).toBe(true);
  for (const o of others) await o.ctx.close();
});

test('gate 7.06: the list passes the accessibility check open on Team and on My jobs', async ({ page, server }) => {
  const a = await signInAsNewPerson(page, server.baseURL, '/tasks/team');
  const aID = await myPersonID(page, server.baseURL);
  const title = uniqueName('Axe job');
  await seedJobs(page, server.baseURL, [title], 'todo', aID);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  await card(lane(page, a), title).locator('.people-menu-trigger').click();
  await expect(card(lane(page, a), title).locator('.people-menu-panel')).toBeVisible();
  await axeCheck(page);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await card(page.locator('.myjobs-mine'), title).locator('.people-menu-trigger').click();
  await expect(card(page.locator('.myjobs-mine'), title).locator('.people-menu-panel')).toBeVisible();
  await axeCheck(page);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 7.09: the circles are a link to a page where people are put on and taken off, and it returns to Team or My jobs', async ({ page, server }) => {
    const a = await signInAsNewPerson(page, server.baseURL, '/tasks/team');
    const title = uniqueName('Plain job');
    await seedJobs(page, server.baseURL, [title], 'todo');

    await page.goto(server.baseURL + '/tasks/team');
    await card(page.locator('.team-lane.unassigned'), title).locator('.people-menu-trigger').click();
    await expect(page).toHaveURL(/\/tasks\/\d+\/people\?redirect_to=team$/);
    await expect(page.getByRole('link', { name: 'Back to Team' })).toBeVisible();
    await page.locator('.people-menu-item', { hasText: a }).click();
    await expect(page).toHaveURL(/\/tasks\/team$/);
    await expect(card(lane(page, a), title)).toHaveCount(1);

    await page.goto(server.baseURL + '/tasks');
    await card(page.locator('.myjobs-mine'), title).locator('.people-menu-trigger').click();
    await expect(page).toHaveURL(/\/tasks\/\d+\/people\?redirect_to=myjobs$/);
    await page.locator('.people-menu-item', { hasText: a }).click(); // take themselves off
    await expect(page).toHaveURL(/\/tasks$/);
    await expect(card(page.locator('.myjobs-grabs'), title)).toHaveCount(1);
  });
});
