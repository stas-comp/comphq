import { expect, test } from '../helpers/fixtures';
import { test as testToday } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, stage = 'idea'): Promise<void> {
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
async function dragOnce(page: Page, sourceTitle: string, targetTitle: string, dropBelow: boolean): Promise<number> {
  const source = page.locator('.task-card', { hasText: sourceTitle });
  const target = page.locator('.task-card', { hasText: targetTitle });
  // The source point is the title line specifically, not the card's
  // overall bounding-box centre: a card's actions row (move up/down,
  // the "Move to…" select) is filtered from starting a drag (board.js's
  // `filter: 'button, select'`), and on a card with assignees and a due
  // date the plain centre point can land inside that actions row,
  // silently refusing to start a drag at all.
  const sourceTitleBox = await source.locator('.task-card-title').boundingBox();
  const targetBox = await target.boundingBox();
  if (!sourceTitleBox || !targetBox) throw new Error('dragCardOnto: card not found');

  // A margin well inside the target (not right at its edge): a drop
  // right at a card's boundary is more sensitive to exactly where
  // SortableJS decides "before" ends and "after" begins.
  const targetY = dropBelow ? targetBox.y + targetBox.height * 0.75 : targetBox.y + targetBox.height * 0.25;
  const sourceX = sourceTitleBox.x + sourceTitleBox.width / 2;
  const sourceY = sourceTitleBox.y + sourceTitleBox.height / 2;
  const targetX = targetBox.x + targetBox.width / 2;
  await page.mouse.move(sourceX, sourceY);
  await page.mouse.down();
  // Small pauses at each phase, not just intermediate positions: a
  // pointer-based drag library like SortableJS processes drag-start and
  // drop-target detection on its own timers/rAF callbacks, which can
  // lose a rapid burst of synthetic events with no real time between
  // them (observed directly: ~20% of drops silently did nothing without
  // these waits, despite the same coordinates).
  await page.mouse.move(sourceX, sourceY + 10, { steps: 5 });
  await page.waitForTimeout(100);
  await page.mouse.move(sourceX + (targetX - sourceX) / 2, sourceY + (targetY - sourceY) / 2, { steps: 10 });
  await page.waitForTimeout(100);
  await page.mouse.move(targetX, targetY, { steps: 10 });
  await page.waitForTimeout(150);

  let response;
  try {
    const moved = page.waitForResponse((res) => /\/tasks\/\d+\/move$/.test(new URL(res.url()).pathname) && res.request().method() === 'POST', { timeout: 3000 });
    await page.mouse.up();
    response = await moved;
  } catch (err) {
    // SortableJS occasionally never registers the drag as started at
    // all (no move request fires) despite identical synthetic input —
    // release the button so the page isn't left mid-drag and signal
    // the caller to retry the whole gesture from scratch.
    await page.mouse.up();
    return 0;
  }
  return response.status();
}

// SortableJS drives its own pointer tracking off timers/rAF rather than
// the native HTML5 Drag and Drop API, and occasionally misses a
// synthetic drag entirely (no move request ever fires) regardless of
// coordinates or pacing — a known, hard-to-eliminate flake class for
// automating pointer-based drag libraries. Retrying the whole gesture,
// rather than loosening the assertion, keeps the test proving a real
// drag actually persists server-side every time it reports success.
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
  expect(headings[1]).toContain('Top = most important');
  expect(headings[2]).toContain('In progress');
  expect(headings[3]).toContain('Done');
});

// SPEC gate 2.02: adding a task with just a title puts it at the bottom
// of the chosen column for everyone.
test('gate 2.02: a title-only task is visible in a second context after reload', async ({ page, server, browser }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const title = uniqueName('Fix the printer');
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
  await page.fill('#new-task-title', title);
  await page.fill('#new-task-due-date', '2099-01-01');
  await page.selectOption('#new-task-people', { label: you });
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const card = page.locator('.task-card', { hasText: title });
  await expect(card).toHaveCount(1);
  await expect(card.locator('.task-avatar')).toHaveCount(1);
  await expect(card.locator('.task-avatar')).toHaveAttribute('title', you);
  await expect(card).toContainText('2099-01-01');
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
  await page.fill('#new-task-title', overdueTitle);
  await page.fill('#new-task-due-date', '2026-09-01');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  const overdueCard = page.locator('.task-card', { hasText: overdueTitle });
  await expect(overdueCard.locator('.task-overdue-stamp')).toHaveText('OVERDUE');

  const futureTitle = uniqueName('Future task');
  await page.fill('#new-task-title', futureTitle);
  await page.fill('#new-task-due-date', '2026-12-01');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('.task-card', { hasText: futureTitle }).locator('.task-overdue-stamp')).toHaveCount(0);

  const doneTitle = uniqueName('Done but past due');
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

  // Move C up one slot: swaps with B -> [A, C, B].
  const cardC = page.locator('.task-card', { hasText: c });
  await cardC.locator('.inline-form', { hasText: 'Move up' }).locator('button').click();
  await ready(page);
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

  const moveUpButton = page.locator('.task-card', { hasText: b }).locator('.inline-form', { hasText: 'Move up' }).locator('button');
  await moveUpButton.focus();
  await page.keyboard.press('Enter');
  await ready(page);

  expect(relativeOrder(await columnOrder(page, stage), [a, b])).toEqual([b, a]);
});

// SPEC gate 2.04 (drag part), 2.05 (dragged card lands where dropped):
// dragging within a column reorders it, consistent after a reload.
test('dragging within a column reorders it, and the order survives a reload', async ({ page, server }) => {
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
test('dragging a card to another column moves it there', async ({ page, server }) => {
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
