import { expect, test } from '../helpers/fixtures';
import { test as testToday } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { openNewTask } from '../helpers/tasks';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, stage: string, personName?: string): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  if (personName) {
    await page.selectOption('#new-task-people', { label: personName });
  }
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

function laneLocator(page: Page, name: string) {
  return page.locator('.team-lane', { has: page.locator('h2', { hasText: name }) });
}

async function columnOrder(page: Page, stage: string): Promise<string[]> {
  return page.locator(`.task-column[data-stage="${stage}"] .task-card-title`).allTextContents();
}

// The same "poll for SortableJS's own drag-start signal, then retry the
// whole gesture" approach board.spec.ts's drag tests already settled
// on (see D-41) — a synthetic pointer-driven drag against a library
// that isn't the native HTML5 Drag and Drop API needs both.
async function waitForDragStart(page: Page, taskId: string, timeout: number): Promise<boolean> {
  try {
    await page.waitForFunction(
      (id) => document.querySelector(`[data-task-id="${id}"]`)?.classList.contains('sortable-chosen'),
      taskId,
      { timeout },
    );
    return true;
  } catch {
    return false;
  }
}

async function dragOnce(page: Page, taskTitle: string, fromLanePersonName: string, targetLanePersonName: string): Promise<number> {
  // A shared task (assigned to more than one person) appears in every
  // one of their lanes — scope the drag source to the specific lane
  // it's being dragged from, not just its title, which could otherwise
  // match more than one element on the page.
  const source = laneLocator(page, fromLanePersonName).locator('.team-task', { hasText: taskTitle });
  const sourceId = await source.getAttribute('data-task-id');
  const sourceLink = source.locator('a').first();
  const sourceBox = await sourceLink.boundingBox();

  const targetLane = laneLocator(page, targetLanePersonName);
  const targetList = targetLane.locator('.team-up-next-list');
  const targetBox = await targetList.boundingBox();
  if (!sourceId || !sourceBox || !targetBox) throw new Error('dragOnce: element not found');

  const sourceX = sourceBox.x + sourceBox.width / 2;
  const sourceY = sourceBox.y + sourceBox.height / 2;
  const targetX = targetBox.x + targetBox.width / 2;
  const targetY = targetBox.y + Math.max(targetBox.height / 2, 8);

  await page.mouse.move(sourceX, sourceY);
  await page.mouse.down();
  await page.mouse.move(sourceX, sourceY + 10, { steps: 5 });

  if (!(await waitForDragStart(page, sourceId, 3000))) {
    await page.mouse.up();
    return 0;
  }

  // Real wall-clock pauses between waypoints, not just intermediate
  // coordinates: SortableJS's own dragover/container-membership
  // tracking runs off timers/rAF (the same reason board.spec.ts's own
  // drag helper needed this, D-41) — a cross-lane drag landing back in
  // the source list despite geometrically correct coordinates was
  // exactly this, worse over the larger distance between two lanes.
  await page.mouse.move(sourceX + (targetX - sourceX) / 2, sourceY + (targetY - sourceY) / 2, { steps: 15 });
  await page.waitForTimeout(100);
  await page.mouse.move(targetX, targetY, { steps: 15 });
  await page.waitForTimeout(150);

  // Specifically the assign endpoint, not just "some move/assign
  // request fired": a drop that lands back in the source lane despite
  // aiming at another one still gets a 302 (from the move endpoint),
  // which would otherwise look like a false success to the caller.
  let response;
  try {
    const applied = page.waitForResponse(
      (res) => /\/tasks\/\d+\/assign$/.test(new URL(res.url()).pathname) && res.request().method() === 'POST',
      { timeout: 5000 },
    );
    await page.mouse.up();
    response = await applied;
  } catch {
    await page.mouse.up();
    return 0;
  }
  return response.status();
}

async function dragTaskToLane(page: Page, taskTitle: string, fromLanePersonName: string, targetLanePersonName: string): Promise<void> {
  const attempts = 4;
  for (let attempt = 1; attempt <= attempts; attempt++) {
    const status = await dragOnce(page, taskTitle, fromLanePersonName, targetLanePersonName);
    if (status === 302) return;
    if (status !== 0) throw new Error(`request failed: ${status}`);
    if (attempt === attempts) throw new Error('dragTaskToLane: drag never registered after retries');
  }
}

// SPEC gate 2.28: dragging from Unassigned onto a person assigns it to
// them; dragging from one person to another takes the first off and
// puts the second on, and anyone else on the job stays.
// @fresh: the Team view's own lanes accumulate a new one for every
// person any other test in this shared worker has ever created, and
// .team-lanes scrolls sideways once they don't fit (SPEC gate 2.31) —
// on a long-running worker, B's lane can end up scrolled off-screen,
// same root cause as board.spec.ts's own drag flake (D-41).
testToday('@fresh gate 2.28: drag assigns from Unassigned to a person, and from one person to another while keeping a third', async ({ page, server, browser }) => {
  const nameA = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  const nameB = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  const nameC = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await otherContext.close();

  await page.reload();
  await ready(page);

  const task1 = uniqueName('Unassigned to A');
  await addTask(page, task1, 'todo');

  const task2 = uniqueName('Shared reassign');
  await openNewTask(page);
  await page.fill('#new-task-title', task2);
  await page.selectOption('#new-task-stage', 'todo');
  await page.selectOption('#new-task-people', [{ label: nameA }, { label: nameC }]);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);

  await dragTaskToLane(page, task1, 'Unassigned', nameA);
  await expect(laneLocator(page, nameA).locator('.team-task', { hasText: task1 })).toHaveCount(1);
  await expect(laneLocator(page, 'Unassigned').locator('.team-task', { hasText: task1 })).toHaveCount(0);

  await dragTaskToLane(page, task2, nameA, nameB);
  await expect(laneLocator(page, nameB).locator('.team-task', { hasText: task2 })).toHaveCount(1);
  await expect(laneLocator(page, nameA).locator('.team-task', { hasText: task2 })).toHaveCount(0);
  await expect(laneLocator(page, nameC).locator('.team-task', { hasText: task2 })).toHaveCount(1);
});

// SPEC gate 2.28: "An 'Assign to…' button does the same without dragging."
test('gate 2.28: the Assign to… button assigns the same way as dragging', async ({ page, server, browser }) => {
  const nameA = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await otherContext.close();

  await page.reload();
  await ready(page);

  const title = uniqueName('Button assign');
  await addTask(page, title, 'todo');

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);

  const unassignedTask = laneLocator(page, 'Unassigned').locator('.team-task', { hasText: title });
  await unassignedTask.locator('select[name="to_person"]').selectOption({ label: nameA });
  await unassignedTask.locator('button', { hasText: 'Assign to…' }).click();
  await ready(page);

  await expect(laneLocator(page, nameA).locator('.team-task', { hasText: title })).toHaveCount(1);
  await expect(laneLocator(page, 'Unassigned').locator('.team-task', { hasText: title })).toHaveCount(0);
});

// SPEC gate 2.29: reordering a lane's Up next changes the shared
// priority order — on the Board, only the moved task's relative
// position changes, and every other task keeps its place.
test('gate 2.29: reordering a lane\'s Up next changes only the moved task\'s position in the shared order', async ({ page, server }) => {
  const nameA = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const other1 = uniqueName('Other1');
  await addTask(page, other1, 'todo');
  const a1 = uniqueName('A1');
  await addTask(page, a1, 'todo', nameA);
  const other2 = uniqueName('Other2');
  await addTask(page, other2, 'todo');
  const a2 = uniqueName('A2');
  await addTask(page, a2, 'todo', nameA);

  const beforeOrder = await columnOrder(page, 'todo');

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  await laneLocator(page, nameA).locator('.team-task', { hasText: a2 }).locator('button', { hasText: 'Move up' }).click();
  await ready(page);
  await expect(page).toHaveURL(/\/tasks\/team$/);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  const afterOrder = await columnOrder(page, 'todo');

  // "Directly above the job it was placed next to" (gate 2.29) means
  // removing a2 from wherever it was and reinserting it immediately
  // before a1 — not a simple two-element swap, since other1/other2 in
  // between shift by one to make room while keeping their own relative
  // order, exactly as the gate's wording describes.
  const expected = beforeOrder.filter((t) => t !== a2);
  expected.splice(expected.indexOf(a1), 0, a2);
  expect(afterOrder).toEqual(expected);
});

// SPEC gate 2.31: with more people than fit, the lanes container
// scrolls sideways inside its own area, and the page itself doesn't.
//
// Measures document.body's own scrollWidth, not
// document.documentElement's: on a long-running shared server with
// enough accumulated lanes, Chromium can report documentElement.
// scrollWidth as far wider than anything actually visible — every
// individual lane still measures at its correct 232px width and
// position, so this is a measurement quirk tied to a legitimate nested
// overflow:auto region, not a real side-scroll (the same reason
// e2e/helpers/no-side-scroll.ts's shared check now does the same).
test('gate 2.31: with many people, the lanes scroll sideways and the page does not', async ({ page, server, browser }) => {
  await page.setViewportSize({ width: 1024, height: 700 });
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  for (let i = 0; i < 9; i++) {
    const ctx = await browser.newContext();
    const p = await ctx.newPage();
    await signInAsNewPerson(p, server.baseURL, '/tasks/board');
    await ctx.close();
  }

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);

  const pageScrollWidth = await page.evaluate(() => document.body.scrollWidth);
  const pageClientWidth = await page.evaluate(() => document.documentElement.clientWidth);
  expect(pageScrollWidth).toBeLessThanOrEqual(pageClientWidth);

  const lanes = page.locator('.team-lanes');
  const lanesScrollWidth = await lanes.evaluate((el) => el.scrollWidth);
  const lanesClientWidth = await lanes.evaluate((el) => el.clientWidth);
  expect(lanesScrollWidth).toBeGreaterThan(lanesClientWidth);
});
