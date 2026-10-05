import { expect, test } from '../helpers/fixtures';
import { dragCardTo } from '../helpers/drag';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { myPersonID, seedJobs } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;
type Locator = import('@playwright/test').Locator;

// SPEC gates 7.04 and 7.05 (B13.1, D-59): the Unassigned lane lists ideas nobody
// is on, and an idea can be dragged into any lane's Up next on Team.

test.describe.configure({ timeout: 60_000 });

const lane = (page: Page, name: string): Locator =>
  page.locator('.team-lane', { has: page.locator('h2', { hasText: name }) });
const unassigned = (page: Page): Locator => page.locator('.team-lane.unassigned');

async function titlesIn(l: Locator, list: string): Promise<string[]> {
  return l.locator(`${list} .team-task:not(.sortable-fallback) .task-card-title`).allInnerTexts();
}
const upNext = (l: Locator) => titlesIn(l, '.team-up-next-list');
const ideas = (l: Locator) => titlesIn(l, '.team-ideas-list');

async function twoPeople(page: Page, server: { baseURL: string }, browser: import('@playwright/test').Browser) {
  const a = await signInAsNewPerson(page, server.baseURL, '/tasks/team');
  const aID = await myPersonID(page, server.baseURL);
  const other = await browser.newContext();
  const otherPage = await other.newPage();
  const b = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/team');
  const bID = await myPersonID(otherPage, server.baseURL);
  return { a, aID, b, bID, other, otherPage };
}

async function dropPoint(l: Locator, top: boolean) {
  const box = (await l.boundingBox())!;
  return { x: box.x + box.width / 2, y: box.y + box.height * (top ? 0.1 : 0.9) };
}

test('gate 7.04: Unassigned lists ideas nobody is on, in their own group, and they are not workload', async ({ page, server, browser }) => {
  const { a, aID, other } = await twoPeople(page, server, browser);
  await other.close();
  const loose = uniqueName('Loose idea');
  const mine = uniqueName('My idea');
  await seedJobs(page, server.baseURL, [loose], 'idea');
  await seedJobs(page, server.baseURL, [mine], 'idea', aID);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const un = unassigned(page);
  await expect(un.locator('h3.sub', { hasText: 'Ideas' })).toBeVisible();
  expect(await ideas(un)).toContain(loose);
  expect(await upNext(un)).not.toContain(loose);
  // Ideas never count: no workload row on Unassigned, and none for a person with only an idea.
  await expect(un.locator('.workload')).toHaveCount(0);
  await expect(lane(page, a).locator('.workload-block')).toHaveCount(0);
  expect(await ideas(lane(page, a))).toEqual([mine]);
});

test('gate 7.05: an idea dragged into its own lane\'s Up next becomes a To do job where it was dropped', async ({ page, server, browser }) => {
  const { a, aID, other } = await twoPeople(page, server, browser);
  await other.close();
  const [t1, t2] = [uniqueName('First'), uniqueName('Second')];
  const idea = uniqueName('Maybe');
  await seedJobs(page, server.baseURL, [t1, t2], 'todo', aID);
  await seedJobs(page, server.baseURL, [idea], 'idea', aID);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const mine = lane(page, a);
  const status = await dragCardTo(
    page,
    mine.locator('.team-ideas-list .team-task', { hasText: idea }),
    async () => dropPoint(mine.locator('.team-up-next-list .team-task:not(.sortable-fallback)', { hasText: t2 }), true),
    { settled: async () => (await upNext(mine)).join() === [t1, idea, t2].join(), restart: false },
  );
  expect(status).toBe(302);
  await expect.poll(() => upNext(mine)).toEqual([t1, idea, t2]);
  expect(await ideas(mine)).toEqual([]);
  // The workload grew by the idea's size now that it is a job (medium = 2 blocks, x3 jobs).
  await expect(mine.locator('.workload-block')).toHaveCount(6);

  // The same on another computer, and History says so.
  const second = await browser.newContext();
  await second.addCookies(await page.context().cookies(server.baseURL));
  const p2 = await second.newPage();
  await p2.goto(server.baseURL + '/tasks/team');
  await ready(p2);
  expect(await upNext(lane(p2, a))).toEqual([t1, idea, t2]);
  const href = await lane(p2, a).locator('.team-up-next-list .team-task', { hasText: idea }).locator('.task-card-title a').getAttribute('href');
  await p2.goto(server.baseURL + href!);
  await ready(p2);
  await expect(p2.locator('.task-activity-list')).toContainText(`${a} moved this to To do`);
  await second.close();
});

test('gate 7.05: an idea dragged into another person\'s Up next is handed over and becomes To do there', async ({ page, server, browser }) => {
  const { a, aID, b, bID, other } = await twoPeople(page, server, browser);
  await other.close();
  const [theirs1, theirs2] = [uniqueName('Theirs one'), uniqueName('Theirs two')];
  const idea = uniqueName('Hand me over');
  await seedJobs(page, server.baseURL, [theirs1, theirs2], 'todo', bID);
  await seedJobs(page, server.baseURL, [idea], 'idea', aID);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const mineLane = lane(page, a);
  const theirLane = lane(page, b);
  const status = await dragCardTo(
    page,
    mineLane.locator('.team-ideas-list .team-task', { hasText: idea }),
    async () => dropPoint(theirLane.locator('.team-up-next-list .team-task:not(.sortable-fallback)', { hasText: theirs2 }), true),
    { settled: async () => (await upNext(theirLane)).join() === [theirs1, idea, theirs2].join(), restart: false },
  );
  expect(status).toBe(302);
  await expect.poll(() => upNext(lane(page, b))).toEqual([theirs1, idea, theirs2]);
  expect(await ideas(lane(page, a))).toEqual([]);
  expect(await upNext(lane(page, a))).not.toContain(idea);
});

test('gate 7.05: an idea can be dropped into an empty Up next, and an unassigned idea is handed to that person', async ({ page, server, browser }) => {
  const { a, aID, b, other } = await twoPeople(page, server, browser);
  await other.close();
  const mineIdea = uniqueName('Mine');
  const looseIdea = uniqueName('Nobody');
  await seedJobs(page, server.baseURL, [mineIdea], 'idea', aID);
  await seedJobs(page, server.baseURL, [looseIdea], 'idea');

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  // B has nothing at all: their Up next is empty.
  expect(await upNext(lane(page, b))).toEqual([]);
  const status = await dragCardTo(
    page,
    lane(page, a).locator('.team-ideas-list .team-task', { hasText: mineIdea }),
    async () => dropPoint(lane(page, b).locator('.team-up-next-list'), true),
    { settled: async () => (await upNext(lane(page, b))).includes(mineIdea), restart: false },
  );
  expect(status).toBe(302);
  await expect.poll(() => upNext(lane(page, b))).toEqual([mineIdea]);

  // An idea from Unassigned into A's lane (A has an empty Up next now).
  const status2 = await dragCardTo(
    page,
    unassigned(page).locator('.team-ideas-list .team-task', { hasText: looseIdea }),
    async () => dropPoint(lane(page, a).locator('.team-up-next-list'), true),
    { settled: async () => (await upNext(lane(page, a))).includes(looseIdea), restart: false },
  );
  expect(status2).toBe(302);
  await expect.poll(() => upNext(lane(page, a))).toEqual([looseIdea]);
  expect(await ideas(unassigned(page))).not.toContain(looseIdea);
});

test('gate 7.05: every idea card on Team has Move to To do, which puts it at the bottom of To do', async ({ page, server, browser }) => {
  const { a, aID, other } = await twoPeople(page, server, browser);
  await other.close();
  const t1 = uniqueName('Existing');
  const mine = uniqueName('Mine idea');
  const loose = uniqueName('Loose idea');
  await seedJobs(page, server.baseURL, [t1], 'todo', aID);
  await seedJobs(page, server.baseURL, [mine], 'idea', aID);
  await seedJobs(page, server.baseURL, [loose], 'idea');

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  await lane(page, a).locator('.team-ideas-list .team-task', { hasText: mine }).getByRole('button', { name: 'Move to To do' }).click();
  await ready(page);
  expect(await upNext(lane(page, a))).toEqual([t1, mine]);
  expect(await ideas(lane(page, a))).toEqual([]);

  await unassigned(page).locator('.team-ideas-list .team-task', { hasText: loose }).getByRole('button', { name: 'Move to To do' }).click();
  await ready(page);
  expect(await upNext(unassigned(page))).toContain(loose);
  expect(await ideas(unassigned(page))).not.toContain(loose);

  const href = await unassigned(page).locator('.team-up-next-list .team-task', { hasText: loose }).locator('.task-card-title a').getAttribute('href');
  await page.goto(server.baseURL + href!);
  await ready(page);
  await expect(page.locator('.task-activity-list')).toContainText(`${a} moved this to To do`);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 7.05: Move to To do is an ordinary form on Team', async ({ page, server }) => {
    const a = await signInAsNewPerson(page, server.baseURL, '/tasks/team');
    const aID = await myPersonID(page, server.baseURL);
    const idea = uniqueName('Plain idea');
    await seedJobs(page, server.baseURL, [idea], 'idea', aID);
    await page.goto(server.baseURL + '/tasks/team');
    await lane(page, a).locator('.team-ideas-list .team-task', { hasText: idea }).getByRole('button', { name: 'Move to To do' }).click();
    expect(await upNext(lane(page, a))).toEqual([idea]);
  });
});
