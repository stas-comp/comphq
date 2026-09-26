import { expect, test } from '../helpers/fixtures';
import { test as testToday } from '../helpers/fixtures-today';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { openNewTask } from '../helpers/tasks';

type Page = import('@playwright/test').Page;
type Locator = import('@playwright/test').Locator;

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

async function columnOrder(page: Page, stage: string): Promise<string[]> {
  return page.locator(`.task-column[data-stage="${stage}"] .task-card-title`).allTextContents();
}

// The same "poll for SortableJS's own drag-start signal, then retry the
// whole gesture" approach the Team view's own drag tests settled on
// (see D-41): a synthetic pointer-driven drag against a library that
// isn't the native HTML5 Drag and Drop API needs both.
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

// Drags an Up for grabs card onto `target` (a specific existing Up next
// row to land just above, or the Up next list itself to land wherever
// it's otherwise empty), and waits specifically for the take endpoint's
// own response — a same-list misfire elsewhere in this app can still
// return a matching redirect, which would otherwise look like a false
// success.
async function dragOnceIntoUpNext(page: Page, taskTitle: string, target: Locator): Promise<number> {
  const source = page.locator('.myjobs-grabs-list .team-task', { hasText: taskTitle });
  const sourceId = await source.getAttribute('data-task-id');
  const sourceLink = source.locator('a').first();
  await sourceLink.scrollIntoViewIfNeeded();
  const sourceBox = await sourceLink.boundingBox();
  await target.scrollIntoViewIfNeeded();
  const targetBox = await target.boundingBox();
  if (!sourceId || !sourceBox || !targetBox) throw new Error('dragOnceIntoUpNext: element not found');

  const sourceX = sourceBox.x + sourceBox.width / 2;
  const sourceY = sourceBox.y + sourceBox.height / 2;
  const targetX = targetBox.x + targetBox.width / 2;
  const targetY = targetBox.y + Math.min(targetBox.height / 3, 10);

  await page.mouse.move(sourceX, sourceY);
  await page.mouse.down();
  await page.mouse.move(sourceX, sourceY + 10, { steps: 5 });

  if (!(await waitForDragStart(page, sourceId, 3000))) {
    await page.mouse.up();
    return 0;
  }

  // Real wall-clock pauses between waypoints, not just intermediate
  // coordinates: SortableJS's own dragover/container-membership
  // tracking runs off timers/rAF (D-41).
  await page.mouse.move(sourceX + (targetX - sourceX) / 2, sourceY + (targetY - sourceY) / 2, { steps: 15 });
  await page.waitForTimeout(100);
  await page.mouse.move(targetX, targetY, { steps: 15 });
  await page.waitForTimeout(150);

  let response;
  try {
    const applied = page.waitForResponse(
      (res) => /\/tasks\/\d+\/take$/.test(new URL(res.url()).pathname) && res.request().method() === 'POST',
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

async function dragTaskIntoUpNext(page: Page, taskTitle: string, target: Locator): Promise<void> {
  const attempts = 4;
  for (let attempt = 1; attempt <= attempts; attempt++) {
    const status = await dragOnceIntoUpNext(page, taskTitle, target);
    if (status === 302) return;
    if (status !== 0) throw new Error(`request failed: ${status}`);
    if (attempt === attempts) throw new Error('dragTaskIntoUpNext: drag never registered after retries');
  }
}

// SPEC gate 2.32: opening Tasks always shows My jobs first.
test('gate 2.32: a fresh name on a computer lands on My jobs', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  await ready(page);

  await expect(page.locator('.myjobs-columns')).toHaveCount(1);
  await expect(page.locator('.tasks-view-switch a[aria-current="page"]')).toHaveText('My jobs');
});

// SPEC gate 2.33: My jobs holds the person's own lane (their own and
// shared jobs) beside Up for grabs (unassigned doing/todo in priority
// order, then unassigned ideas) — never a job belonging only to someone
// else, and never a finished one.
test('gate 2.33: My jobs shows the person\'s own lane and Up for grabs, and nothing else', async ({ page, server, browser }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  const other = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await otherContext.close();

  await page.reload();
  await ready(page);

  const mine = uniqueName('Mine doing');
  await addTask(page, mine, 'doing', me);

  const shared = uniqueName('Shared todo');
  await openNewTask(page);
  await page.fill('#new-task-title', shared);
  await page.selectOption('#new-task-stage', 'todo');
  await page.selectOption('#new-task-people', [{ label: me }, { label: other }]);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const othersOnly = uniqueName('Only other');
  await addTask(page, othersOnly, 'todo', other);

  const grabbableDoing = uniqueName('Grabbable doing');
  await addTask(page, grabbableDoing, 'doing');
  const grabbableTodo = uniqueName('Grabbable todo');
  await addTask(page, grabbableTodo, 'todo');
  const grabbableIdea = uniqueName('Grabbable idea');
  await addTask(page, grabbableIdea, 'idea');

  const finished = uniqueName('Finished job');
  await addTask(page, finished, 'done', me);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);

  const mineArea = page.locator('.myjobs-mine');
  await expect(mineArea.locator('.team-task', { hasText: mine })).toHaveCount(1);
  await expect(mineArea.locator('.team-task', { hasText: shared })).toHaveCount(1);
  await expect(mineArea.locator('.team-task', { hasText: othersOnly })).toHaveCount(0);
  await expect(mineArea.locator('.team-task', { hasText: finished })).toHaveCount(0);

  const grabsArea = page.locator('.myjobs-grabs');
  await expect(grabsArea.locator('.team-task', { hasText: grabbableDoing })).toHaveCount(1);
  await expect(grabsArea.locator('.team-task', { hasText: grabbableTodo })).toHaveCount(1);
  await expect(grabsArea.locator('.team-task', { hasText: grabbableIdea })).toHaveCount(1);
  await expect(grabsArea.locator('.team-task', { hasText: shared })).toHaveCount(0);
  await expect(grabsArea.locator('.team-task', { hasText: othersOnly })).toHaveCount(0);
  await expect(grabsArea.locator('.team-task', { hasText: finished })).toHaveCount(0);

  // In progress and To do first, in priority order, then ideas.
  const grabTitles = await grabsArea.locator('.team-task a').allTextContents();
  const doingIdx = grabTitles.indexOf(grabbableDoing);
  const todoIdx = grabTitles.indexOf(grabbableTodo);
  const ideaIdx = grabTitles.indexOf(grabbableIdea);
  expect(doingIdx).toBeLessThan(todoIdx);
  expect(todoIdx).toBeLessThan(ideaIdx);
});

// SPEC gate 2.34: dragging a To do job from Up for grabs into My jobs
// puts the person on it, placed where it was dropped in their Up next.
// @fresh: Up for grabs accumulates every unassigned task any other test
// in this shared worker has ever created, which can push the dragged
// card far enough down the page for a coordinate-based drag to misfire
// — the same accumulation risk the Team view's own drag test hit
// (D-41), avoided the same way, with an isolated server.
testToday('@fresh gate 2.34: take by drag places the job where it was dropped', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const first = uniqueName('First');
  await addTask(page, first, 'todo', me);
  const second = uniqueName('Second');
  await addTask(page, second, 'todo', me);
  const grabbed = uniqueName('Grabbed by drag');
  await addTask(page, grabbed, 'todo');

  await page.goto(server.baseURL + '/tasks');
  await ready(page);

  const firstRow = page.locator('.myjobs-upnext-list .team-task', { hasText: first });
  await dragTaskIntoUpNext(page, grabbed, firstRow);

  const titles = (await page.locator('.myjobs-upnext-list .team-task a').allTextContents()).map((t) => t.trim());
  expect(titles).toEqual([grabbed, first, second]);
});

// SPEC gate 2.34: pressing Take it does the same but keeps the job's
// current place in the shared priority order.
test('gate 2.34: take by button keeps the job\'s place in the shared priority order', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const before1 = uniqueName('Before1');
  await addTask(page, before1, 'todo');
  const grabbed = uniqueName('Taken by button');
  await addTask(page, grabbed, 'todo');
  const after1 = uniqueName('After1');
  await addTask(page, after1, 'todo');

  // Only this test's own three cards are compared: the server is shared
  // with tests running at the same time, and one of theirs can land in
  // the To do column between the two reads.
  const mine = [before1, grabbed, after1];
  const beforeOrder = (await columnOrder(page, 'todo')).filter((t) => mine.includes(t));
  expect(beforeOrder).toEqual(mine);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await page.locator('.myjobs-grabs-list .team-task', { hasText: grabbed }).locator('button', { hasText: 'Take it' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-upnext-list .team-task', { hasText: grabbed })).toHaveCount(1);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  const afterOrder = (await columnOrder(page, 'todo')).filter((t) => mine.includes(t));
  expect(afterOrder).toEqual(beforeOrder);
});

// SPEC gate 2.35: taking an idea (here, by Take it) puts the person on
// it and moves it to To do.
test('gate 2.35: taking an idea moves it to To do', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const idea = uniqueName('Someday idea');
  await addTask(page, idea, 'idea');

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await page.locator('.myjobs-grabs-list .team-task', { hasText: idea }).locator('button', { hasText: 'Take it' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-upnext-list .team-task', { hasText: idea })).toHaveCount(1);

  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await expect(page.locator('.task-column[data-stage="idea"] .task-card-title', { hasText: idea })).toHaveCount(0);
  await expect(page.locator('.task-column[data-stage="todo"] .task-card-title', { hasText: idea })).toHaveCount(1);
});

test('standard page checks for My jobs', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
