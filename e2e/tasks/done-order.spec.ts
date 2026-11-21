import { expect, test } from '../helpers/fixtures';
import { beginDrag, dragCardTo, dragPointerTo, endDrag } from '../helpers/drag';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { myPersonID, seedJobs } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 8.01-8.05 (B14.1, D-104): Done is in finishing order, the most
// recently finished on top, and can't be put in a hand-made order. The board
// is filtered to "My tasks" so these tests see only their own jobs on a shared
// server. Finishing times have a one-second resolution, so each finish is
// separated from the last by a little over a second: the order of two jobs
// finished in the same second is settled by their ids (a store test covers it).
test.describe.configure({ timeout: 90_000 });

const FINISH_GAP_MS = 1100;

async function doneTitles(page: Page): Promise<string[]> {
  return (await page.locator('.task-column[data-stage="done"] .task-card:not(.sortable-fallback) .task-card-title').allInnerTexts()).map((t) => t.trim());
}

function only(all: string[], wanted: string[]): string[] {
  return all.filter((t) => wanted.includes(t));
}

async function openMine(page: Page, baseURL: string): Promise<void> {
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
}

/** Drags a card into Done and drops it on the lower half of the card named `onto` (or on the column's empty space). */
async function dragIntoDone(page: Page, title: string, onto?: string): Promise<number> {
  const card = page.locator('.task-column:not([data-stage="done"]) .task-card', { hasText: title });
  const moved = page.locator('.task-column[data-stage="done"] .task-card', { hasText: title });
  return dragCardTo(
    page,
    card,
    async () => {
      if (onto) {
        const box = (await page.locator('.task-column[data-stage="done"] .task-card', { hasText: onto }).boundingBox())!;
        return { x: box.x + box.width / 2, y: box.y + box.height * 0.85 };
      }
      const col = (await page.locator('.task-column[data-stage="done"]').boundingBox())!;
      const viewport = page.viewportSize()!;
      return { x: col.x + col.width / 2, y: Math.min(col.y + col.height - 30, viewport.height - 20) };
    },
    { settled: async () => (await moved.count()) > 0 },
  );
}

test('gate 8.01: Done lists the most recently finished job first, after a refresh, on another computer, and with a filter', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Order');
  const [a, b, c] = [`${prefix} A`, `${prefix} B`, `${prefix} C`];
  await seedJobs(page, server.baseURL, [a, b, c], 'doing', me);
  await openMine(page, server.baseURL);

  // Finish A, then C, then B, with the buttons ("Move to…").
  for (const title of [a, c, b]) {
    const card = page.locator('.task-card', { hasText: title });
    await card.locator('.task-move > summary').click();
    await card.locator('.task-move-to-form select').selectOption('done');
    await card.locator('.task-move-to-form button').click();
    await ready(page);
    await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: title })).toHaveCount(1);
    await page.waitForTimeout(FINISH_GAP_MS);
  }
  const want = [b, c, a];
  expect(only(await doneTitles(page), want)).toEqual(want);

  await page.reload();
  await ready(page);
  expect(only(await doneTitles(page), want)).toEqual(want);

  const other = await browser.newContext();
  const otherPage = await other.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await otherPage.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(prefix));
  await ready(otherPage);
  expect(only(await doneTitles(otherPage), want)).toEqual(want);
  await other.close();
});

test('gate 8.02: a job dragged into Done, dropped low, is at the top straight away and after a reload', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Drag');
  const [first, second, third] = [`${prefix} first`, `${prefix} second`, `${prefix} third`];
  await seedJobs(page, server.baseURL, [first, second], 'done', me);
  await seedJobs(page, server.baseURL, [third], 'doing', me);
  await openMine(page, server.baseURL);
  // seedJobs made `first` then `second` (the same second, so by id): second is on top.
  expect(await doneTitles(page)).toEqual([second, first]);
  await page.waitForTimeout(FINISH_GAP_MS);

  // Dropped on the lower half of the last card, the old way of putting it last.
  const status = await dragIntoDone(page, third, first);
  expect(status).toBe(302);
  await expect.poll(() => doneTitles(page)).toEqual([third, second, first]);

  await page.reload();
  await ready(page);
  expect(await doneTitles(page)).toEqual([third, second, first]);

  const other = await browser.newContext();
  const otherPage = await other.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await otherPage.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(prefix));
  await ready(otherPage);
  expect(await doneTitles(otherPage)).toEqual([third, second, first]);
  await other.close();
});

test('gate 8.02: Done ✓ on My jobs puts the job at the top of Done', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Mine');
  const [older, newer] = [`${prefix} older`, `${prefix} newer`];
  await seedJobs(page, server.baseURL, [older, newer], 'doing', me);

  for (const title of [older, newer]) {
    await page.goto(server.baseURL + '/tasks');
    await ready(page);
    await page.locator('.myjobs-working-list .team-task', { hasText: title }).getByRole('button', { name: /Done/ }).click();
    await ready(page);
    await page.waitForTimeout(FINISH_GAP_MS);
  }
  await openMine(page, server.baseURL);
  expect(only(await doneTitles(page), [older, newer])).toEqual([newer, older]);
});

test('gate 8.03: Done cards have no move icons, and a drag up or down inside Done changes nothing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Still');
  const [a, b, c] = [`${prefix} A`, `${prefix} B`, `${prefix} C`];
  await seedJobs(page, server.baseURL, [a, b, c], 'done', me);
  await openMine(page, server.baseURL);
  const before = await doneTitles(page);
  expect(only(before, [a, b, c])).toEqual([c, b, a]);

  const doneCards = page.locator('.task-column[data-stage="done"] .task-card');
  await expect(doneCards).toHaveCount(3);
  await expect(doneCards.getByRole('button', { name: /Move .* (up|down)/ })).toHaveCount(0);
  // The other buttons stay: remove, and each card's own "Move to…".
  await expect(doneCards.getByRole('button', { name: /^Remove / })).toHaveCount(3);
  // The other columns keep their move icons.
  await seedJobs(page, server.baseURL, [`${prefix} todo`], 'todo', me);
  await openMine(page, server.baseURL);
  await expect(page.locator('.task-column[data-stage="todo"] .task-card').getByRole('button', { name: /Move .* up/ })).toHaveCount(1);

  // Try to carry the bottom card up above the top one, and let go.
  const moveRequests: string[] = [];
  page.on('request', (req) => {
    if (/\/tasks\/\d+\/move$/.test(new URL(req.url()).pathname)) moveRequests.push(req.url());
  });
  const bottom = page.locator('.task-column[data-stage="done"] .task-card', { hasText: a });
  const top = page.locator('.task-column[data-stage="done"] .task-card', { hasText: c });
  await beginDrag(page, bottom);
  const box = (await top.boundingBox())!;
  await dragPointerTo(page, box.x + box.width / 2, box.y + box.height * 0.15);
  expect(await endDrag(page)).toBe(0); // nothing is sent: the card goes back to its place
  await page.waitForTimeout(500);
  expect(only(await doneTitles(page), [a, b, c])).toEqual([c, b, a]);
  expect(moveRequests).toEqual([]);
  await page.reload();
  await ready(page);
  expect(only(await doneTitles(page), [a, b, c])).toEqual([c, b, a]);
});

test('gate 8.03: dragging a card out of Done reopens it, and finishing it again puts it back on top (8.04)', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Reopen');
  const [a, b] = [`${prefix} A`, `${prefix} B`];
  await seedJobs(page, server.baseURL, [a, b], 'done', me);
  await seedJobs(page, server.baseURL, [`${prefix} stays`], 'doing', me);
  await openMine(page, server.baseURL);
  expect(only(await doneTitles(page), [a, b])).toEqual([b, a]);

  // Reopen A by dragging it onto the card in Working on now.
  const anchor = page.locator('.task-column[data-stage="doing"] .task-card', { hasText: `${prefix} stays` });
  const status = await dragCardTo(
    page,
    page.locator('.task-column[data-stage="done"] .task-card', { hasText: a }),
    async () => {
      const box = (await anchor.boundingBox())!;
      return { x: box.x + box.width / 2, y: box.y + box.height * 0.8 };
    },
    { settled: async () => (await page.locator('.task-column[data-stage="doing"] .task-card', { hasText: a }).count()) > 0 },
  );
  expect(status).toBe(302);
  await expect(page.locator('.task-column[data-stage="doing"] .task-card', { hasText: a })).toHaveCount(1);
  await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: a })).toHaveCount(0);

  // Finish it again, well after B: it is on top, ahead of B.
  await page.waitForTimeout(FINISH_GAP_MS);
  const status2 = await dragIntoDone(page, a);
  expect(status2).toBe(302);
  await expect.poll(async () => only(await doneTitles(page), [a, b])).toEqual([a, b]);
  await page.reload();
  await ready(page);
  expect(only(await doneTitles(page), [a, b])).toEqual([a, b]);
});
