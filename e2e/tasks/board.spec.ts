import { expect, test } from '../helpers/fixtures';
import { test as testToday } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { openNewTask } from '../helpers/tasks';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, stage = 'idea'): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
}

// Titles as they appear, top to bottom, within one stage column — used
// to check relative order among a test's own uniquely-named tasks
// without assuming the column is otherwise empty (the shared server
// this test runs against may have other tasks in it already).
async function columnOrder(page: Page, stage: string): Promise<string[]> {
  return page.locator(`.task-column[data-stage="${stage}"] .task-card-title`).allTextContents();
}

function relativeOrder(all: string[], wanted: string[]): string[] {
  return all.filter((title) => wanted.includes(title));
}

// SortableJS drives its own mouse/pointer-based drag (not the HTML5 Drag
// and Drop API), so explicit page.mouse steps — not locator.dragTo,
// which some SortableJS versions don't reliably recognise as a real
// drag — move the mouse over the target card's top or bottom half to
// land before or after it, then release. Waits for board.js's own fetch
// to the move endpoint to finish before returning: SortableJS reorders
// the dragged card's DOM node the instant it's dropped, independent of
// whether that fetch (which is what actually persists the move
// server-side) has completed yet, so asserting on the DOM right after
// mouse.up alone can pass on a move that never really landed.
// Polls (rather than sleeping a guessed duration) for SortableJS's own
// "drag has started" signal — the `sortable-chosen` class it applies to
// the dragged element — so this works regardless of how fast the
// machine is: a fixed sleep tuned against a fast dev machine turned out
// to still be too short on CI's much slower 2-vCPU runner, where the
// drag was never recognised as started at all (zero move requests).
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

async function dragOnce(page: Page, sourceTitle: string, targetTitle: string, dropBelow: boolean): Promise<number> {
  const source = page.locator('.task-card', { hasText: sourceTitle });
  const target = page.locator('.task-card', { hasText: targetTitle });
  const sourceId = await source.getAttribute('data-task-id');
  // A synthetic mouse can only drop on what is inside the window, and how
  // far down a card sits depends on the page above it. Bring both cards
  // into view first (the target last, so the source stays in view above
  // it), then measure.
  await source.scrollIntoViewIfNeeded();
  await target.scrollIntoViewIfNeeded();
  // The source point is the title line specifically, not the card's
  // overall bounding-box centre: a card's actions row (move up/down,
  // the "Move to…" select) is filtered from starting a drag (board.js's
  // `filter: 'button, select'`), and on a card with assignees and a due
  // date the plain centre point can land inside that actions row,
  // silently refusing to start a drag at all.
  const sourceTitleBox = await source.locator('.task-card-title').boundingBox();
  const targetBox = await target.boundingBox();
  if (!sourceId || !sourceTitleBox || !targetBox) throw new Error('dragCardOnto: card not found');

  // A margin well inside the target (not right at its edge): a drop
  // right at a card's boundary is more sensitive to exactly where
  // SortableJS decides "before" ends and "after" begins.
  const targetY = dropBelow ? targetBox.y + targetBox.height * 0.75 : targetBox.y + targetBox.height * 0.25;
  const sourceX = sourceTitleBox.x + sourceTitleBox.width / 2;
  const sourceY = sourceTitleBox.y + sourceTitleBox.height / 2;
  const targetX = targetBox.x + targetBox.width / 2;
  await page.mouse.move(sourceX, sourceY);
  await page.mouse.down();
  await page.mouse.move(sourceX, sourceY + 10, { steps: 5 });

  if (!(await waitForDragStart(page, sourceId, 3000))) {
    // SortableJS never recognised this as a drag at all (no move
    // request will ever fire) — release the button so the page isn't
    // left mid-drag, and signal the caller to retry the whole gesture.
    await page.mouse.up();
    return 0;
  }

  await page.mouse.move(sourceX + (targetX - sourceX) / 2, sourceY + (targetY - sourceY) / 2, { steps: 10 });
  await page.mouse.move(targetX, targetY, { steps: 10 });

  let response;
  try {
    const moved = page.waitForResponse((res) => /\/tasks\/\d+\/move$/.test(new URL(res.url()).pathname) && res.request().method() === 'POST', { timeout: 5000 });
    await page.mouse.up();
    response = await moved;
  } catch {
    await page.mouse.up();
    return 0;
  }
  return response.status();
}

// A bounded retry around the whole gesture, on top of polling for
// SortableJS's own drag-start signal inside dragOnce: even confirming
// the drag started, a single synthetic run can still occasionally miss
// the drop (e.g. a move event landing between two rAF callbacks). Each
// attempt still requires a genuine 302 from the move endpoint, so this
// compensates for automation timing noise without weakening what a
// passing attempt actually proves.
async function dragCardOnto(page: Page, sourceTitle: string, targetTitle: string, dropBelow: boolean): Promise<void> {
  const attempts = 4;
  for (let attempt = 1; attempt <= attempts; attempt++) {
    const status = await dragOnce(page, sourceTitle, targetTitle, dropBelow);
    if (status === 302) return;
    if (status !== 0) throw new Error(`move request failed: ${status}`);
    if (attempt === attempts) throw new Error('dragCardOnto: drag never registered after retries');
  }
}

// SPEC gate 2.01: the Board shows four columns, in order.
test('gate 2.01: Board shows Ideas, To do, In progress, Done in order', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const headings = await page.locator('.task-column h2').allTextContents();
  expect(headings.length).toBe(4);
  expect(headings[0]).toContain('Ideas');
  expect(headings[1]).toContain('To do');
  // Gate 4.17: the hint sits beside the To do heading, not inside it.
  await expect(page.locator('.task-column[data-stage="todo"] .task-column-head')).toContainText('Top = most important');
  expect(headings[2]).toContain('In progress');
  expect(headings[3]).toContain('Done');
});

// SPEC gate 2.02: adding a task with just a title puts it at the bottom
// of the chosen column for everyone.
test('gate 2.02: a title-only task is visible in a second context after reload', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Fix the printer');
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(1);
  // Defaults: bottom of Ideas (the default starting column), size M
  // isn't shown on the card itself (SPEC A6 only shows title, people, due
  // date on a card), but the task must land in Ideas.
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: title })).toHaveCount(1);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await expect(otherPage.locator('.task-card', { hasText: title })).toHaveCount(1);
  await otherContext.close();
});

test('gate 2.02: a task can be added directly into a chosen column', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Plan the offsite');
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', 'doing');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await expect(page.locator('.task-column[data-stage="doing"] .task-card', { hasText: title })).toHaveCount(1);
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: title })).toHaveCount(0);
});

// SPEC gate 2.03 (assignees/due date part) and 2.26 (size default): a
// task with people and a due date shows both on its card.
test('gate 2.03: a card shows its assigned people and due date', async ({ page, server }) => {
  const you = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Team task');
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.fill('#new-task-due-date', '2099-01-01');
  await page.selectOption('#new-task-people', { label: you });
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const card = page.locator('.task-card', { hasText: title });
  await expect(card).toHaveCount(1);
  await expect(card.locator('.av')).toHaveCount(1);
  await expect(card.locator('.av')).toHaveAttribute('title', you);
  await expect(card).toContainText('Thu 1 Jan 2099'); // gate 4.16: the mockup's short date, with the year because it isn't this year
  await expect(card.locator('.task-overdue-stamp')).toHaveCount(0);
});

// SPEC gate 2.03 (OVERDUE part): an unfinished task past its due date
// shows an OVERDUE stamp; a task in Done never does, regardless of date.
// @fresh: needs its own server with a fixed COMPHQ_TEST_TODAY.
testToday.use({ today: '2026-09-19' });
testToday('@fresh gate 2.03: an unfinished task past its due date shows OVERDUE', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const overdueTitle = uniqueName('Overdue task');
  await openNewTask(page);
  await page.fill('#new-task-title', overdueTitle);
  await page.fill('#new-task-due-date', '2026-09-01');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const overdueCard = page.locator('.task-card', { hasText: overdueTitle });
  await expect(overdueCard.locator('.task-overdue-stamp')).toHaveText('OVERDUE');

  const futureTitle = uniqueName('Future task');
  await openNewTask(page);
  await page.fill('#new-task-title', futureTitle);
  await page.fill('#new-task-due-date', '2026-12-01');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: futureTitle }).locator('.task-overdue-stamp')).toHaveCount(0);

  const doneTitle = uniqueName('Done but past due');
  await openNewTask(page);
  await page.fill('#new-task-title', doneTitle);
  await page.fill('#new-task-due-date', '2026-09-01');
  await page.selectOption('#new-task-stage', 'done');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: doneTitle }).locator('.task-overdue-stamp')).toHaveCount(0);
});

// SPEC gate 2.04 (button part): Move up/Move down reorder within a
// stage, consistent on reload from a second context.
test('gate 2.04: Move up and Move down reorder cards, visible in a second context', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const stage = 'todo';
  const a = uniqueName('A');
  const b = uniqueName('B');
  const c = uniqueName('C');
  await addTask(page, a, stage);
  await addTask(page, b, stage);
  await addTask(page, c, stage);

  expect(relativeOrder(await columnOrder(page, stage), [a, b, c])).toEqual([a, b, c]);

  // Move C up one slot at a time until it passes B -> [A, C, B]. The
  // server is shared with tests running at the same time, so a card
  // from another test can sit between B and C; each press must still
  // move C up by exactly one place in the column.
  const cardC = page.locator('.task-card', { hasText: c });
  for (let presses = 0; presses < 5; presses++) {
    const before = (await columnOrder(page, stage)).indexOf(c);
    await cardC.getByRole('button', { name: `Move ${c} up` }).click(); // an icon-only button, named for the task (gate 4.11)
    await ready(page);
    expect((await columnOrder(page, stage)).indexOf(c)).toBe(before - 1);
    if (relativeOrder(await columnOrder(page, stage), [a, b, c]).join() === [a, c, b].join()) break;
  }
  expect(relativeOrder(await columnOrder(page, stage), [a, b, c])).toEqual([a, c, b]);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  expect(relativeOrder(await columnOrder(otherPage, stage), [a, b, c])).toEqual([a, c, b]);
  await otherContext.close();
});

// SPEC gate 2.05: a card moved into a different column by button lands
// at the bottom of that column.
test('gate 2.05: Move to… lands a card at the bottom of the target column', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const existing = uniqueName('Existing');
  await addTask(page, existing, 'todo');
  const moving = uniqueName('Moving');
  await addTask(page, moving, 'idea');

  const movingCard = page.locator('.task-card', { hasText: moving });
  await movingCard.locator('.task-move > summary').click(); // the Move to… button equivalent opens on request
  await movingCard.locator('.task-move-to-form select').selectOption('todo');
  await movingCard.locator('.task-move-to-form').locator('button').click();
  await ready(page);

  expect(relativeOrder(await columnOrder(page, 'todo'), [existing, moving])).toEqual([existing, moving]);
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: moving })).toHaveCount(0);
});

// Keyboard-only E2E (PLAN.md P2-02): move a task by buttons with no mouse.
test('gate 2.04: a task can be moved with keyboard only', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const stage = 'todo';
  const a = uniqueName('KbA');
  const b = uniqueName('KbB');
  await addTask(page, a, stage);
  await addTask(page, b, stage);
  expect(relativeOrder(await columnOrder(page, stage), [a, b])).toEqual([a, b]);

  const moveUpButton = page.locator('.task-card', { hasText: b }).getByRole('button', { name: `Move ${b} up` });
  await moveUpButton.focus();
  await page.keyboard.press('Enter');
  await ready(page);

  expect(relativeOrder(await columnOrder(page, stage), [a, b])).toEqual([b, a]);
});

// SPEC gate 2.04 (drag part), 2.05 (dragged card lands where dropped):
// dragging within a column reorders it, consistent after a reload.
// @fresh: this board's own worker-scoped server is shared and long-lived
// (reused across every other board test, with no way yet to delete a
// task — that arrives with P2-06), so a column could otherwise already
// hold enough cards from earlier tests to push a drag target below the
// fold, off the viewport a synthetic mouse drag can actually reach.
testToday('@fresh dragging within a column reorders it, and the order survives a reload', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const stage = 'todo';
  const a = uniqueName('DragA');
  const b = uniqueName('DragB');
  await addTask(page, a, stage);
  await addTask(page, b, stage);
  expect(relativeOrder(await columnOrder(page, stage), [a, b])).toEqual([a, b]);

  // Drop A onto the bottom half of B: A lands after B -> [B, A].
  await dragCardOnto(page, a, b, true);
  await expect
    .poll(async () => relativeOrder(await columnOrder(page, stage), [a, b]))
    .toEqual([b, a]);

  await page.reload();
  await ready(page);
  expect(relativeOrder(await columnOrder(page, stage), [a, b])).toEqual([b, a]);
});

// SPEC gate 2.04 (drag part): dragging a card to another column moves it.
// @fresh: see the previous test for why this needs its own empty board.
testToday('@fresh dragging a card to another column moves it there', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const anchor = uniqueName('DragAnchor');
  await addTask(page, anchor, 'doing');
  const moving = uniqueName('DragMoving');
  await addTask(page, moving, 'idea');

  await dragCardOnto(page, moving, anchor, false);

  await expect
    .poll(async () => (await columnOrder(page, 'doing')).includes(moving))
    .toBe(true);
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: moving })).toHaveCount(0);

  await page.reload();
  await ready(page);
  await expect(page.locator('.task-column[data-stage="doing"] .task-card', { hasText: moving })).toHaveCount(1);
});
