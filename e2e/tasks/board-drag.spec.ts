import { expect, test } from '../helpers/fixtures';
import { beginDrag, dragCardTo, dragPointerTo, endDrag, holdPointer } from '../helpers/drag';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { myPersonID, seedJobs } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 7.12, 7.13, 7.14 (B13.2). These use "My tasks" as the filter so
// the board holds only this test's own jobs on a shared server.

function titles(prefix: string, n: number): string[] {
  return Array.from({ length: n }, (_, i) => `${prefix} ${String(i + 1).padStart(2, '0')}`);
}

async function columnTitles(page: Page, stage: string): Promise<string[]> {
  return page.locator(`.task-column[data-stage="${stage}"] .task-card-title`).allInnerTexts();
}

test('gate 7.12: a card dragged from low in a long column lands at the bottom of a short one', async ({ page, server }) => {
  test.setTimeout(90_000);
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Long');
  const ideas = titles(`${prefix} idea`, 16);
  const only = `${prefix} only to do`;
  await seedJobs(page, server.baseURL, ideas, 'idea', me);
  await seedJobs(page, server.baseURL, [only], 'todo', me);

  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);

  const moving = ideas[14]; // the 15th card
  const card = page.locator('.task-column[data-stage="idea"] .task-card', { hasText: moving });
  const status = await dragCardTo(page, card, async () => {
    // Below To do's only card, in the empty space that is still the column.
    const col = (await page.locator('.task-column[data-stage="todo"]').boundingBox())!;
    const viewport = page.viewportSize()!;
    return { x: col.x + col.width / 2, y: Math.min(col.y + col.height - 30, viewport.height - 20) };
  });
  expect(status).toBe(302);

  expect(await columnTitles(page, 'todo')).toEqual([only, moving]);
  // Still true for everyone after a refresh.
  await page.reload();
  await ready(page);
  expect(await columnTitles(page, 'todo')).toEqual([only, moving]);
  expect(await columnTitles(page, 'idea')).not.toContain(moving);
});

test('gate 7.13: holding a card near the window edges scrolls the page', async ({ page, server }) => {
  test.setTimeout(120_000);
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Scroll');
  const ideas = titles(`${prefix} idea`, 24);
  const only = `${prefix} only to do`;
  await seedJobs(page, server.baseURL, ideas, 'idea', me);
  await seedJobs(page, server.baseURL, [only], 'todo', me);

  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  const startY = await page.evaluate(() => window.scrollY);
  expect(startY).toBeGreaterThan(1000);

  const moving = ideas[ideas.length - 1];
  const card = page.locator('.task-column[data-stage="idea"] .task-card', { hasText: moving });
  await beginDrag(page, card);

  // Just under the sticky top bar: the page should scroll up on its own.
  const viewport = page.viewportSize()!;
  await dragPointerTo(page, viewport.width / 2, 100);
  await holdPointer(page, viewport.width / 2, 100, async () => (await page.evaluate(() => window.scrollY)) < startY - 800);
  // It keeps going to the top.
  await holdPointer(page, viewport.width / 2, 100, async () => (await page.evaluate(() => window.scrollY)) < 50, 30_000);

  // Carry it onto the top of To do's only card and let go.
  const target = (await page.locator('.task-column[data-stage="todo"] .task-card', { hasText: only }).boundingBox())!;
  await dragPointerTo(page, target.x + target.width / 2, target.y + target.height * 0.25);
  expect(await endDrag(page)).toBe(302);
  // Carried across after the scroll. (Which side of the one card it lands on
  // depends on the exact pixel, which gate 7.12 and the order tests cover.)
  expect((await columnTitles(page, 'todo')).sort()).toEqual([moving, only].sort());
  expect(await columnTitles(page, 'idea')).not.toContain(moving);

  // And back down: near the bottom edge scrolls the other way.
  const again = page.locator('.task-column[data-stage="todo"] .task-card', { hasText: only });
  await beginDrag(page, again);
  const before = await page.evaluate(() => window.scrollY);
  await dragPointerTo(page, viewport.width / 2, viewport.height - 15);
  await holdPointer(page, viewport.width / 2, viewport.height - 15, async () => (await page.evaluate(() => window.scrollY)) > before + 400);
  await dragPointerTo(page, viewport.width / 2, viewport.height / 2);
  await endDrag(page);
});

test('gate 7.13: every Tasks list is set up to scroll while dragging', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  for (const [path, selector] of [
    ['/tasks/board', '.task-card-list'],
    ['/tasks/team', '.team-up-next-list'],
    ['/tasks', '.myjobs-upnext-list'],
  ]) {
    await page.goto(server.baseURL + path);
    await ready(page);
    const options = await page.evaluate((sel) => {
      const list = document.querySelector(sel) as HTMLElement & { _sortable?: unknown };
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const s = (window as any).Sortable.get(list);
      return s ? { scroll: s.options.scroll, force: s.options.forceAutoScrollFallback, fallback: s.options.forceFallback, sensitivity: s.options.scrollSensitivity } : null;
    }, selector);
    expect(options, `${path} has a sortable list`).not.toBeNull();
    expect(options!.scroll).toBe(true);
    expect(options!.force).toBe(true);
    expect(options!.fallback, 'Sortable follows the pointer itself').toBe(true);
    // The sticky top bar covers the top 76px of the window.
    expect(options!.sensitivity).toBeGreaterThanOrEqual(116);
  }
});

test('gate 7.14: a filter in use is still applied, and still shown, after a drag', async ({ page, server, browser }) => {
  test.setTimeout(90_000);
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const me = await myPersonID(page, server.baseURL);
  const prefix = uniqueName('Keep');
  const mine = titles(`${prefix} mine`, 3);
  await seedJobs(page, server.baseURL, mine, 'todo', me);

  // Somebody else's job with the same word in its title.
  const other = await browser.newContext();
  const otherPage = await other.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  const theirs = `${prefix} theirs`;
  await seedJobs(otherPage, server.baseURL, [theirs], 'todo', await myPersonID(otherPage, server.baseURL));
  await other.close();

  const drag = async (): Promise<void> => {
    const card = page.locator('.task-column[data-stage="todo"] .task-card', { hasText: mine[0] });
    const below = page.locator('.task-column[data-stage="todo"] .task-card', { hasText: mine[2] });
    const status = await dragCardTo(page, card, async () => {
      const box = (await below.boundingBox())!;
      return { x: box.x + box.width / 2, y: box.y + box.height * 0.75 };
    });
    expect(status).toBe(302);
  };

  // "My tasks"
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: theirs })).toHaveCount(0);
  await drag();
  await expect(page.locator('.task-column[data-stage="todo"] .task-card', { hasText: mine[0] })).toHaveCount(1);
  const afterFirst = await columnTitles(page, 'todo');
  expect([...afterFirst].sort()).toEqual([...mine].sort());
  expect(afterFirst[0]).not.toBe(mine[0]);
  await expect(page.locator('.task-card', { hasText: theirs })).toHaveCount(0);
  expect(new URL(page.url()).search).toBe('?person=mine');
  await expect(page.locator('#filter-person')).toHaveValue('mine');

  // A word
  await page.goto(server.baseURL + `/tasks/board?q=${encodeURIComponent(prefix + ' mine')}`);
  await ready(page);
  await expect(page.locator('.task-card')).toHaveCount(3);
  await drag();
  await expect(page.locator('.task-card')).toHaveCount(3); // the other person's job is still filtered out, as is everything else
  await expect(page.locator('.task-card', { hasText: theirs })).toHaveCount(0);
  await expect(page.locator('#filter-q')).toHaveValue(`${prefix} mine`);
  expect(new URL(page.url()).search).toContain('q=');
});
