import { expect, test } from '../helpers/fixtures';
import { dragCardTo } from '../helpers/drag';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { myPersonID, seedJobs } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// A drag with real pauses, retried, can take a while on a busy machine.
test.describe.configure({ timeout: 60_000 });

// SPEC gates 7.01, 7.02, 7.03 (B13.1, D-91): reorder your own Up next on My
// jobs, and move ideas you are on into To do.

async function upNext(page: Page): Promise<string[]> {
  // (While a drag is in progress SortableJS keeps a floating copy of the card
  // in the list too; it is not part of the order.)
  return page.locator('.myjobs-upnext-list .team-task:not(.sortable-fallback) .task-card-title').allInnerTexts();
}

async function ideasOnMe(page: Page): Promise<string[]> {
  return page.locator('.myjobs-ideas-list .team-task .task-card-title').allInnerTexts();
}

async function boardTodoRelativeTo(page: Page, wanted: string[]): Promise<string[]> {
  const all = await page.locator('.task-column[data-stage="todo"] .task-card-title').allInnerTexts();
  return all.filter((t) => wanted.includes(t.trim()));
}

async function dropOn(page: Page, titleOfTarget: string, top: boolean) {
  const target = page.locator('.myjobs-upnext-list .team-task', { hasText: titleOfTarget });
  const box = (await target.boundingBox())!;
  return { x: box.x + box.width / 2, y: box.y + box.height * (top ? 0.1 : 0.9) };
}

test('gate 7.01: dragging in Up next reorders it, for everyone, after a refresh', async ({ page, server, browser }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks');
  const id = await myPersonID(page, server.baseURL);
  const [a, b, c] = [uniqueName('Alpha'), uniqueName('Bravo'), uniqueName('Charlie')];
  await seedJobs(page, server.baseURL, [a, b, c], 'todo', id);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  expect(await upNext(page)).toEqual([a, b, c]);

  const status = await dragCardTo(
    page,
    page.locator('.myjobs-upnext-list .team-task', { hasText: c }),
    () => dropOn(page, a, true),
    { settled: async () => (await upNext(page))[0] === c },
  );
  expect(status).toBe(302);
  await expect.poll(() => upNext(page)).toEqual([c, a, b]);

  await page.reload();
  await ready(page);
  expect(await upNext(page)).toEqual([c, a, b]);

  // Another computer, the same person.
  const other = await browser.newContext();
  await other.addCookies(await page.context().cookies(server.baseURL));
  const otherPage = await other.newPage();
  await otherPage.goto(server.baseURL + '/tasks');
  await ready(otherPage);
  expect(await upNext(otherPage)).toEqual([c, a, b]);
  await other.close();
  expect(me).toBeTruthy();
});

test('gate 7.01: each Up next card has move up and move down icons that do the same', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  const id = await myPersonID(page, server.baseURL);
  const [a, b, c] = [uniqueName('Alpha'), uniqueName('Bravo'), uniqueName('Charlie')];
  await seedJobs(page, server.baseURL, [a, b, c], 'todo', id);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);

  const card = (t: string) => page.locator('.myjobs-upnext-list .team-task', { hasText: t });
  // Icon-only, with a spoken name, and disabled at the ends of the list.
  await expect(card(a).getByRole('button', { name: `Move ${a} up` })).toBeDisabled();
  await expect(card(c).getByRole('button', { name: `Move ${c} down` })).toBeDisabled();
  await expect(card(b).getByRole('button', { name: `Move ${b} up` })).toBeEnabled();

  await card(c).getByRole('button', { name: `Move ${c} up` }).click();
  await ready(page);
  expect(await upNext(page)).toEqual([a, c, b]);
  await card(a).getByRole('button', { name: `Move ${a} down` }).click();
  await ready(page);
  expect(await upNext(page)).toEqual([c, a, b]);
});

test('gate 7.02: reordering My jobs moves the job beside the one it was placed by, and no other job moves', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  const id = await myPersonID(page, server.baseURL);
  const [a, x, b, c] = [uniqueName('Alpha'), uniqueName('Xray unassigned'), uniqueName('Bravo'), uniqueName('Charlie')];
  await seedJobs(page, server.baseURL, [a], 'todo', id);
  await seedJobs(page, server.baseURL, [x], 'todo'); // nobody's: Up for grabs, between mine on the Board
  await seedJobs(page, server.baseURL, [b, c], 'todo', id);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  expect(await boardTodoRelativeTo(page, [a, x, b, c])).toEqual([a, x, b, c]);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await dragCardTo(
    page,
    page.locator('.myjobs-upnext-list .team-task', { hasText: c }),
    () => dropOn(page, a, true),
    { settled: async () => (await upNext(page))[0] === c },
  );
  await expect.poll(() => upNext(page)).toEqual([c, a, b]);

  // On the Board: Charlie sits directly above Alpha, and Xray and Bravo
  // keep their places.
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  expect(await boardTodoRelativeTo(page, [a, x, b, c])).toEqual([c, a, x, b]);
});

test('gate 7.03: an idea dragged into Up next becomes a To do job where it was dropped, and History says so', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/tasks');
  const id = await myPersonID(page, server.baseURL);
  const [a, b] = [uniqueName('Alpha'), uniqueName('Bravo')];
  const idea = uniqueName('Maybe later');
  await seedJobs(page, server.baseURL, [a, b], 'todo', id);
  await seedJobs(page, server.baseURL, [idea], 'idea', id);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  expect(await ideasOnMe(page)).toEqual([idea]);

  const status = await dragCardTo(
    page,
    page.locator('.myjobs-ideas-list .team-task', { hasText: idea }),
    () => dropOn(page, b, true),
    { settled: async () => (await upNext(page)).join() === [a, idea, b].join(), restart: false },
  );
  expect(status).toBe(302);
  await expect.poll(() => upNext(page)).toEqual([a, idea, b]);
  expect(await ideasOnMe(page)).toEqual([]);

  const href = await page.locator('.myjobs-upnext-list .team-task', { hasText: idea }).locator('.task-card-title a').getAttribute('href');
  await page.goto(server.baseURL + href!);
  await ready(page);
  await expect(page.locator('.task-activity-list')).toContainText(`${name} moved this to To do`);
});

test('gate 7.03: Move to To do puts an idea at the bottom of To do', async ({ page, server }) => {
  const name = await signInAsNewPerson(page, server.baseURL, '/tasks');
  const id = await myPersonID(page, server.baseURL);
  const [a, b] = [uniqueName('Alpha'), uniqueName('Bravo')];
  const idea = uniqueName('Maybe later');
  await seedJobs(page, server.baseURL, [a, b], 'todo', id);
  await seedJobs(page, server.baseURL, [idea], 'idea', id);
  await page.goto(server.baseURL + '/tasks');
  await ready(page);

  const button = page.locator('.myjobs-ideas-list .team-task', { hasText: idea }).getByRole('button', { name: 'Move to To do' });
  await expect(button).toBeVisible();
  await button.click();
  await ready(page);
  expect(await upNext(page)).toEqual([a, b, idea]);
  expect(await ideasOnMe(page)).toEqual([]);

  const href = await page.locator('.myjobs-upnext-list .team-task', { hasText: idea }).locator('.task-card-title a').getAttribute('href');
  await page.goto(server.baseURL + href!);
  await ready(page);
  await expect(page.locator('.task-activity-list')).toContainText(`${name} moved this to To do`);
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gates 7.01 and 7.03: the move icons and Move to To do are ordinary forms', async ({ page, server }) => {
    await signInAsNewPerson(page, server.baseURL, '/tasks');
    const id = await myPersonID(page, server.baseURL);
    const [a, b, c] = [uniqueName('Alpha'), uniqueName('Bravo'), uniqueName('Charlie')];
    const idea = uniqueName('Maybe later');
    await seedJobs(page, server.baseURL, [a, b, c], 'todo', id);
    await seedJobs(page, server.baseURL, [idea], 'idea', id);
    await page.goto(server.baseURL + '/tasks');

    const card = (t: string) => page.locator('.myjobs-upnext-list .team-task', { hasText: t });
    await card(c).getByRole('button', { name: `Move ${c} up` }).click();
    expect(await upNext(page)).toEqual([a, c, b]);

    await page.locator('.myjobs-ideas-list .team-task', { hasText: idea }).getByRole('button', { name: 'Move to To do' }).click();
    expect(await upNext(page)).toEqual([a, c, b, idea]);
  });
});
